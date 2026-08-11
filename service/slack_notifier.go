package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/aiwolfdial/aiwolf-nlp-server/model"
	"github.com/aiwolfdial/aiwolf-nlp-server/observer/teamhealth"
)

// 設定の events で個別に切れる通知の種類。
const (
	EventQuarantine = "quarantine"
	EventAbort      = "abort"
	EventStall      = "stall"
	EventMilestone  = "milestone"
)

const slackQueueSize = 64

// Slack の Incoming Webhook へ運用イベントを送る。送信はワーカ goroutine に任せ、
// 呼び出し側（ゲームの進行やマッチング）を Slack の遅延で止めない。
type SlackNotifier struct {
	config   model.SlackNotifierConfig
	webhook  string
	events   map[string]bool
	client   *http.Client
	queue    chan string
	done     chan struct{}
	closeOne sync.Once
}

// 有効かつ Webhook URL を解決できたときだけ実体を返す。URL が無いのは設定漏れなので警告する。
func NewSlackNotifier(config model.SlackNotifierConfig) *SlackNotifier {
	if !config.Enable {
		return nil
	}
	webhook := config.WebhookURL
	if webhook == "" {
		webhook = os.Getenv("SLACK_WEBHOOK_URL")
	}
	if webhook == "" {
		slog.Warn("Slack通知が有効ですが、webhook_url も環境変数 SLACK_WEBHOOK_URL も設定されていません")
		return nil
	}
	if config.Timeout <= 0 {
		config.Timeout = 10 * time.Second
	}
	if config.MinInterval <= 0 {
		config.MinInterval = time.Second
	}
	if config.Username == "" {
		config.Username = "aiwolf-nlp-server"
	}
	events := make(map[string]bool, len(config.Events))
	for _, e := range config.Events {
		events[strings.TrimSpace(e)] = true
	}
	n := &SlackNotifier{
		config:  config,
		webhook: webhook,
		events:  events,
		client:  &http.Client{Timeout: config.Timeout},
		queue:   make(chan string, slackQueueSize),
		done:    make(chan struct{}),
	}
	go n.run()
	slog.Info("Slack通知を有効にしました", "events", config.Events)
	return n
}

// events が空なら全種類を送る。設定を書かずに有効化しただけでも通知が届くようにする。
func (n *SlackNotifier) enabled(event string) bool {
	if n == nil {
		return false
	}
	if len(n.events) == 0 {
		return true
	}
	return n.events[event]
}

func (n *SlackNotifier) run() {
	defer close(n.done)
	for text := range n.queue {
		n.post(text)
		// Webhook の流量制限に当たらないよう、送信の間隔を空ける。
		time.Sleep(n.config.MinInterval)
	}
}

func (n *SlackNotifier) enqueue(event string, text string) {
	if !n.enabled(event) {
		return
	}
	select {
	case n.queue <- text:
	default:
		// 送信が詰まってもゲームの進行を止めないため、溢れた通知は捨てる。
		slog.Warn("Slack通知のキューが一杯のため破棄しました", "event", event)
	}
}

func (n *SlackNotifier) post(text string) {
	body, err := json.Marshal(map[string]string{
		"text":       text,
		"username":   n.config.Username,
		"icon_emoji": n.config.IconEmoji,
	})
	if err != nil {
		slog.Error("Slack通知の組み立てに失敗しました", "error", err)
		return
	}
	res, err := n.client.Post(n.webhook, "application/json", bytes.NewReader(body))
	if err != nil {
		slog.Error("Slack通知の送信に失敗しました", "error", err)
		return
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		detail, _ := io.ReadAll(io.LimitReader(res.Body, 512))
		slog.Error("Slack通知が拒否されました", "status", res.StatusCode, "body", string(detail))
	}
}

// Close はキューを閉じ、残っている通知を送り終えるまで待つ。シャットダウン通知が
// 送信前にプロセスごと消えるのを防ぐため、待ち時間には上限を設ける。
func (n *SlackNotifier) Close() {
	if n == nil {
		return
	}
	n.closeOne.Do(func() { close(n.queue) })
	select {
	case <-n.done:
	case <-time.After(n.config.Timeout + n.config.MinInterval):
		slog.Warn("Slack通知の送信待ちが終わらないため打ち切ります")
	}
}

func (n *SlackNotifier) NotifyTeamQuarantined(snapshots []teamhealth.Snapshot) {
	if n == nil || len(snapshots) == 0 {
		return
	}
	var b strings.Builder
	b.WriteString(":warning: *チームを隔離しました*")
	for _, s := range snapshots {
		fmt.Fprintf(&b, "\n• `%s`  失敗率 %.0f%%  重み %.2f  (直近%d試合 / 脱落%d / 異常終了%d / 応答エラー%d件)",
			s.Team, s.FailureRate*100, s.Weight, s.Games, s.FatalGames, s.AbortedGames, s.RequestErrors)
		if s.QuarantinedUntil != nil {
			fmt.Fprintf(&b, "  解除予定 %s", time.Unix(*s.QuarantinedUntil, 0).Format("15:04:05"))
		}
	}
	n.enqueue(EventQuarantine, b.String())
}

func (n *SlackNotifier) NotifyGameAborted(gameID string, teams []string, fatalTeams []string) {
	if n == nil {
		return
	}
	var b strings.Builder
	fmt.Fprintf(&b, ":x: *ゲームが異常終了しました*\nID: `%s`", gameID)
	if len(teams) > 0 {
		fmt.Fprintf(&b, "\n参加: %s", strings.Join(teams, ", "))
	}
	if len(fatalTeams) > 0 {
		fmt.Fprintf(&b, "\n脱落: %s", strings.Join(fatalTeams, ", "))
	}
	n.enqueue(EventAbort, b.String())
}

func (n *SlackNotifier) NotifyMatchmakingStalled(idle time.Duration, waiting []string) {
	if n == nil {
		return
	}
	var b strings.Builder
	fmt.Fprintf(&b, ":hourglass: *マッチが %s 成立していません*", idle.Round(time.Minute))
	if len(waiting) > 0 {
		fmt.Fprintf(&b, "\n待機中 %d チーム: %s", len(waiting), strings.Join(waiting, ", "))
	} else {
		b.WriteString("\n待機部屋は空です")
	}
	n.enqueue(EventStall, b.String())
}

func (n *SlackNotifier) NotifyProgress(done int, total int) {
	if n == nil {
		return
	}
	text := fmt.Sprintf(":chart_with_upwards_trend: *進捗* %d / %d 試合", done, total)
	if total > 0 {
		text += fmt.Sprintf(" (%.0f%%)", float64(done)/float64(total)*100)
	}
	n.enqueue(EventMilestone, text)
}

func (n *SlackNotifier) NotifyServerEvent(text string) {
	if n == nil {
		return
	}
	n.enqueue(EventMilestone, text)
}
