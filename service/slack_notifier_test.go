package service

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/aiwolfdial/aiwolf-nlp-server/model"
	"github.com/aiwolfdial/aiwolf-nlp-server/observer/teamhealth"
)

// Webhook を受け取ったペイロードを溜めるテスト用サーバ。
func newFakeSlack(t *testing.T) (*httptest.Server, func() []slackPayload) {
	t.Helper()
	var mu sync.Mutex
	var received []slackPayload
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var p slackPayload
		json.NewDecoder(r.Body).Decode(&p)
		mu.Lock()
		received = append(received, p)
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)
	return srv, func() []slackPayload {
		mu.Lock()
		defer mu.Unlock()
		return append([]slackPayload(nil), received...)
	}
}

func newTestNotifier(t *testing.T, url string) *SlackNotifier {
	t.Helper()
	t.Setenv("SLACK_WEBHOOK_URL", url)
	n := NewSlackNotifier(model.SlackNotifierConfig{
		Enable: true, Timeout: 2 * time.Second, MinInterval: time.Millisecond,
	})
	if n == nil {
		t.Fatal("SlackNotifier を作成できませんでした")
	}
	return n
}

// URL は設定ファイルではなく環境変数からのみ読む。未設定なら無効化する。
func TestNotifierRequiresEnvWebhook(t *testing.T) {
	t.Setenv("SLACK_WEBHOOK_URL", "")
	if n := NewSlackNotifier(model.SlackNotifierConfig{Enable: true}); n != nil {
		t.Fatal("URLが無いのに有効化されました")
	}
}

// Close 後に通知しても、閉じたチャネルへ送って panic しないこと。
// シャットダウン中も watchdog などが通知しうる。
func TestNotifyAfterCloseDoesNotPanic(t *testing.T) {
	srv, _ := newFakeSlack(t)
	n := newTestNotifier(t, srv.URL)
	n.Close()

	n.NotifyProgress(1, 2, 0)
	n.NotifyGameAborted("id", nil, nil)
	n.NotifyMatchmakingStalled(time.Minute, 0, []string{"alpha"}, nil)
	n.NotifyServerEvent(":x:", "title", "")
	n.Close() // 二度目の Close も安全であること
}

// Close は送信中の通知を送り終えるまで待つ。シャットダウン通知が消えないため。
func TestCloseFlushesQueuedNotifications(t *testing.T) {
	srv, received := newFakeSlack(t)
	n := newTestNotifier(t, srv.URL)

	n.NotifyServerEvent(":octagonal_sign:", "シャットダウンを開始しました", "")
	n.Close()

	if got := received(); len(got) != 1 {
		t.Fatalf("Close 前の通知が送信されていません: %v", len(got))
	}
}

// 外部由来のチーム名が本文にもフォールバック文言にも生のまま出ないこと。
func TestNotificationsEscapeTeamNames(t *testing.T) {
	srv, received := newFakeSlack(t)
	n := newTestNotifier(t, srv.URL)

	n.NotifyGameAborted("01ABC", []string{"<!channel>"}, []string{"<!here>"})
	n.NotifyMatchmakingStalled(time.Minute, 1, []string{"<!channel>"}, []string{"<!here>"})
	n.NotifyTeamQuarantined([]teamhealth.Snapshot{{Team: "<!channel>", Games: 1}})
	n.Close()

	got := received()
	if len(got) != 3 {
		t.Fatalf("通知数が想定と異なります: %v", len(got))
	}
	for _, p := range got {
		raw, err := json.Marshal(p)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		// JSON へ落とした状態で <! が残っていれば Slack がメンションとして解釈する。
		if strings.Contains(string(raw), "<!") {
			t.Fatalf("メンション記法が素通りしています: %s", raw)
		}
	}
}

// events に挙げた種類だけを送ること。
func TestEventFiltering(t *testing.T) {
	srv, received := newFakeSlack(t)
	t.Setenv("SLACK_WEBHOOK_URL", srv.URL)
	n := NewSlackNotifier(model.SlackNotifierConfig{
		Enable: true, Timeout: 2 * time.Second, MinInterval: time.Millisecond,
		Events: []string{EventAbort},
	})
	if n == nil {
		t.Fatal("SlackNotifier を作成できませんでした")
	}

	n.NotifyProgress(1, 2, 0)
	n.NotifyGameAborted("01ABC", nil, nil)
	n.Close()

	got := received()
	if len(got) != 1 {
		t.Fatalf("フィルタが効いていません: %v", len(got))
	}
	if !strings.Contains(got[0].Text, "異常終了") {
		t.Fatalf("送信された通知が想定と異なります: %v", got[0].Text)
	}
}
