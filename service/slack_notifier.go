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
	queue    chan slackPayload
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
		queue:   make(chan slackPayload, slackQueueSize),
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
	for payload := range n.queue {
		n.post(payload)
		// Webhook の流量制限に当たらないよう、送信の間隔を空ける。
		time.Sleep(n.config.MinInterval)
	}
}

// summary はモバイルのプッシュ通知に出る文言なので、装飾を含めない平文にする。
func (n *SlackNotifier) enqueue(event string, summary string, color string, blocks ...slackBlock) {
	if !n.enabled(event) {
		return
	}
	payload := slackPayload{
		Username:    n.config.Username,
		IconEmoji:   n.config.IconEmoji,
		Text:        summary,
		Attachments: []slackAttachment{{Color: color, Blocks: blocks}},
	}
	select {
	case n.queue <- payload:
	default:
		// 送信が詰まってもゲームの進行を止めないため、溢れた通知は捨てる。
		slog.Warn("Slack通知のキューが一杯のため破棄しました", "event", event)
	}
}

func (n *SlackNotifier) post(payload slackPayload) {
	body, err := json.Marshal(payload)
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
	blocks := []slackBlock{sectionBlock(":warning: *チームを隔離しました*")}
	shown := snapshots
	if len(shown) > maxQuarantineBlocks {
		shown = shown[:maxQuarantineBlocks]
	}
	for _, s := range shown {
		blocks = append(blocks, sectionBlock(fmt.Sprintf(
			"*`%s`*\n%s  失敗率 *%.0f%%*  ・  マッチの重み *%.2f*",
			s.Team, progressBar(s.FailureRate, 12), s.FailureRate*100, s.Weight)))
		detail := fmt.Sprintf("直近 %d 試合  ・  脱落 %d  ・  異常終了 %d  ・  応答エラー %d 件",
			s.Games, s.FatalGames, s.AbortedGames, s.RequestErrors)
		if s.QuarantinedUntil != nil {
			// 期限は延長されないので「予定」ではなく確定した時刻として書く。
			detail += fmt.Sprintf("  ・  %s に自動解除", time.Unix(*s.QuarantinedUntil, 0).Format("15:04:05"))
		}
		blocks = append(blocks, contextBlock(detail))
	}
	if len(shown) < len(snapshots) {
		blocks = append(blocks, contextBlock(fmt.Sprintf("ほか %d チーム", len(snapshots)-len(shown))))
	}
	names := make([]string, 0, len(snapshots))
	for _, s := range snapshots {
		names = append(names, s.Team)
	}
	n.enqueue(EventQuarantine, "チームを隔離しました: "+strings.Join(names, ", "), colorWarning, blocks...)
}

func (n *SlackNotifier) NotifyGameAborted(gameID string, teams []string, fatalTeams []string) {
	if n == nil {
		return
	}
	blocks := []slackBlock{sectionBlock(":x: *ゲームが異常終了しました*")}
	dropped := "(特定できず)"
	if len(fatalTeams) > 0 {
		dropped = "*" + strings.Join(fatalTeams, "*, *") + "*"
	}
	blocks = append(blocks, fieldsBlock("*ゲームID*\n`"+gameID+"`", "*脱落したチーム*\n"+dropped))
	if len(teams) > 0 {
		blocks = append(blocks, contextBlock(fmt.Sprintf("参加 %d チーム: %s", len(teams), strings.Join(teams, ", "))))
	}
	summary := "ゲームが異常終了しました"
	if len(fatalTeams) > 0 {
		summary += " (脱落: " + strings.Join(fatalTeams, ", ") + ")"
	}
	n.enqueue(EventAbort, summary, colorDanger, blocks...)
}

// NotifyMatchmakingStalled は接続済みのチームと、対戦表に載っているのに接続していない
// チームを並べる。マッチが組めない原因は後者にあるため、両方を分けて出す。
func (n *SlackNotifier) NotifyMatchmakingStalled(idle time.Duration, connected []string, awaiting []string) {
	if n == nil {
		return
	}
	blocks := []slackBlock{
		sectionBlock(fmt.Sprintf(":hourglass: *マッチが %s 成立していません*", idle.Round(time.Second))),
		fieldsBlock(
			fmt.Sprintf(":large_green_circle: *接続中* (%d)\n%s", len(connected), teamList(connected)),
			fmt.Sprintf(":white_circle: *接続待ち* (%d)\n%s", len(awaiting), teamList(awaiting)),
		),
	}
	summary := fmt.Sprintf("マッチが %s 成立していません (接続中 %d / 接続待ち %d)",
		idle.Round(time.Second), len(connected), len(awaiting))
	n.enqueue(EventStall, summary, colorWarning, blocks...)
}

// NotifyProgress は消化率をバーで示す。数字だけだと残量が直感的に掴めないため。
func (n *SlackNotifier) NotifyProgress(done int, total int, active int) {
	if n == nil {
		return
	}
	ratio := ratioOf(done, total)
	blocks := []slackBlock{
		sectionBlock(fmt.Sprintf(":chart_with_upwards_trend: *進捗*  %d / %d 試合\n%s  *%.1f%%*",
			done, total, progressBar(ratio, 20), ratio*100)),
		contextBlock(fmt.Sprintf("残り %d 試合  ・  実行中 %d 試合", max(total-done, 0), active)),
	}
	n.enqueue(EventMilestone, fmt.Sprintf("進捗 %d / %d 試合 (%.1f%%)", done, total, ratio*100), colorGood, blocks...)
}

// NotifyServerEvent は起動・シャットダウンなど本文が固定の通知に使う。
func (n *SlackNotifier) NotifyServerEvent(emoji string, title string, detail string) {
	if n == nil {
		return
	}
	blocks := []slackBlock{sectionBlock(emoji + " *" + title + "*")}
	if detail != "" {
		blocks = append(blocks, contextBlock(detail))
	}
	n.enqueue(EventMilestone, title, colorInfo, blocks...)
}
