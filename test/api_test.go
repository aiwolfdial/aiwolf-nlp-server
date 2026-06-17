package test

import (
	"bufio"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/aiwolfdial/aiwolf-nlp-server/model"
)

func TestAPIGameSnapshotAndStream(t *testing.T) {
	config, err := model.LoadFromPath("./config/talk.yml")
	if err != nil {
		t.Fatalf("設定ファイルの読み込みに失敗しました: %v", err)
	}

	listedCh := make(chan bool, 1)
	eventsCh := make(chan int, 1)
	var once sync.Once

	probe := func(base, gameID string) {
		listed := false
		if resp, err := http.Get(base + "/api/v1/games"); err == nil {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			listed = strings.Contains(string(body), gameID)
		}
		listedCh <- listed

		count := 0
		req, _ := http.NewRequest(http.MethodGet, base+"/api/v1/games/"+gameID+"/events", nil)
		if resp, err := http.DefaultClient.Do(req); err == nil {
			defer resp.Body.Close()
			scanner := bufio.NewScanner(resp.Body)
			deadline := time.Now().Add(3 * time.Second)
			for time.Now().Before(deadline) && scanner.Scan() {
				if strings.HasPrefix(scanner.Text(), "data:") {
					count++
					if count >= 2 {
						break
					}
				}
			}
		}
		eventsCh <- count
	}

	handlers := map[model.Request]func(tc TestClient) (string, error){
		model.R_INITIALIZE: func(tc TestClient) (string, error) {
			if gid, ok := tc.info["game_id"].(string); ok && gid != "" {
				once.Do(func() {
					base := "http://" + config.Server.WebSocket.Host + ":" + strconv.Itoa(config.Server.WebSocket.Port)
					go probe(base, gid)
				})
			}
			return "", nil
		},
		model.R_TALK: func(tc TestClient) (string, error) {
			// プローブが接続する余裕を作るためトークを少し遅らせる。
			time.Sleep(30 * time.Millisecond)
			return "hello", nil
		},
		model.R_DAILY_FINISH: func(tc TestClient) (string, error) {
			return "", nil
		},
	}

	executeGame(t, []string{"WEREWOLF", "POSSESSED", "SEER", "VILLAGER-A", "VILLAGER-B"}, config, handlers)

	select {
	case listed := <-listedCh:
		if !listed {
			t.Error("game was not listed in /api/v1/games during play")
		}
	case <-time.After(10 * time.Second):
		t.Error("probe did not report game listing")
	}

	select {
	case count := <-eventsCh:
		if count < 1 {
			t.Error("no SSE broadcast events received from /api/v1/games/:id/events")
		}
	case <-time.After(10 * time.Second):
		t.Error("probe did not report SSE events")
	}
}
