package test

import (
	"io"
	"math/rand"
	"net"
	"net/url"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/aiwolfdial/aiwolf-nlp-server/model"
	"github.com/aiwolfdial/aiwolf-nlp-server/transport"
)

const WebSocketExternalHost = "0.0.0.0"
const TestClientName = "aiwolf-nlp-viewer"

func launchAsyncServer(t *testing.T, config *model.Config) url.URL {
	if _, exists := os.LookupEnv("GITHUB_ACTIONS"); exists {
		config.Server.WebSocket.Host = WebSocketExternalHost
	}
	port := getAvailableTcpPort(config.Server.WebSocket.Host)
	config.Server.WebSocket.Port = port
	go func() {
		server, err := transport.NewServer(*config)
		if err != nil {
			return
		}
		server.Run()
	}()
	t.Parallel()
	return url.URL{Scheme: "ws", Host: config.Server.WebSocket.Host + ":" + strconv.Itoa(config.Server.WebSocket.Port), Path: "/ws"}
}

func getAvailableTcpPort(host string) int {
	rand := rand.New(rand.NewSource(time.Now().UnixNano()))
	port := rand.Intn(65535-49152+1) + 49152
	for {
		listener, err := net.Listen("tcp", host+":"+strconv.Itoa(port))
		if err == nil {
			listener.Close()
			break
		}
		port = rand.Intn(65535-49152+1) + 49152
	}
	return port
}

func executeSelfMatchGame(t *testing.T, config *model.Config, handlers map[model.Request]func(tc TestClient) (string, error)) {
	u := launchAsyncServer(t, config)
	t.Logf("サーバを起動しました: %s", u.String())
	time.Sleep(1 * time.Second)

	clients := make([]*TestClient, config.Game.AgentCount)
	for i := range config.Game.AgentCount {
		client, err := NewTestClient(t, u, TestClientName, handlers)
		if err != nil {
			t.Fatalf("クライアントの初期化に失敗しました: %v", err)
		}
		clients[i] = client
		defer clients[i].close()
	}

	for _, client := range clients {
		select {
		case <-client.done:
			t.Log("done")
		case <-time.After(5 * time.Minute):
			t.Fatalf("timeout")
		}
	}
	time.Sleep(3 * time.Second)
	t.Log("ゲームが終了しました")
}

func executeGame(t *testing.T, names []string, config *model.Config, handlers map[model.Request]func(tc TestClient) (string, error)) {
	if config.Matching.IsOptimize {
		dstFile, err := os.CreateTemp("", "*.json")
		if err != nil {
			t.Fatalf("一時ファイルの作成に失敗しました: %v", err)
		}

		srcFile, err := os.Open(config.Matching.OutputPath)
		if err != nil {
			t.Fatalf("設定ファイルの読み込みに失敗しました: %v", err)
		}
		defer srcFile.Close()

		io.Copy(dstFile, srcFile)
		config.Matching.OutputPath = dstFile.Name()

		defer os.Remove(dstFile.Name())
	}

	u := launchAsyncServer(t, config)
	t.Logf("サーバを起動しました: %s", u.String())
	time.Sleep(1 * time.Second)

	clients := make([]*TestClient, config.Game.AgentCount)
	for i := range config.Game.AgentCount {
		client, err := NewTestClient(t, u, names[i], handlers)
		if err != nil {
			t.Fatalf("クライアントの初期化に失敗しました: %v", err)
		}
		clients[i] = client
		defer clients[i].close()
	}

	for _, client := range clients {
		select {
		case <-client.done:
			t.Log("done")
		case <-time.After(5 * time.Minute):
			t.Fatalf("timeout")
		}
	}
	time.Sleep(3 * time.Second)
	t.Log("ゲームが終了しました")
}
