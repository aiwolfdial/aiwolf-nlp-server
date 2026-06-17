package service

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/aiwolfdial/aiwolf-nlp-server/model"
)

type RealtimeBroadcaster struct {
	config model.RealtimeBroadcasterConfig
	data   sync.Map
}

type RealtimeBroadcasterLog struct {
	id        string
	filename  string
	agents    []any
	logs      []string
	packetIdx int
	logsMu    sync.Mutex
	updatedAt time.Time
}

func NewRealtimeBroadcaster(config model.Config) *RealtimeBroadcaster {
	rb := &RealtimeBroadcaster{
		config: config.RealtimeBroadcaster,
	}
	if err := os.MkdirAll(rb.config.OutputDir, 0755); err != nil {
		slog.Error("出力ディレクトリの作成に失敗しました", "error", err)
		return nil
	}
	if err := os.WriteFile(filepath.Join(rb.config.OutputDir, "games.json"), []byte("[]"), 0644); err != nil {
		slog.Error("ゲーム一覧ファイルの初期化に失敗しました", "error", err)
		return nil
	}
	slog.Info("リアルタイムブロードキャスターを初期化しました", "output_dir", rb.config.OutputDir)
	return rb
}

func (rb *RealtimeBroadcaster) TrackStartGame(id string, agents []model.AgentView) {
	agentData := make([]any, 0, len(agents))
	teamNames := make([]string, 0, len(agents))

	for _, agent := range agents {
		agentInfo := map[string]any{
			"idx":  agent.Idx,
			"team": agent.TeamName,
			"name": agent.OriginalName,
			"role": agent.Role,
		}
		agentData = append(agentData, agentInfo)
		teamNames = append(teamNames, agent.TeamName)
	}

	sort.Strings(teamNames)
	filename := strings.ReplaceAll(rb.config.Filename, "{game_id}", id)
	filename = strings.ReplaceAll(filename, "{timestamp}", fmt.Sprintf("%d", time.Now().Unix()))
	filename = strings.ReplaceAll(filename, "{teams}", strings.Join(teamNames, "_"))

	gameLog := &RealtimeBroadcasterLog{
		id:        id,
		filename:  filename,
		agents:    agentData,
		logs:      make([]string, 0),
		updatedAt: time.Now(),
	}

	rb.data.Store(id, gameLog)
}

func (rb *RealtimeBroadcaster) TrackEndGame(id string) {
	if gameLogInterface, exists := rb.data.Load(id); exists {
		gameLog := gameLogInterface.(*RealtimeBroadcasterLog)
		gameLog.logsMu.Lock()
		logs := make([]string, len(gameLog.logs))
		copy(logs, gameLog.logs)
		filename := gameLog.filename
		gameLog.logsMu.Unlock()

		rb.writeGameFile(filename, logs)
		rb.writeGamesListFile()
		rb.data.Delete(id)
	}
}

// Emit はゲーム状態スナップショットとイベント固有の情報からパケットを組み立てて配信する。
// パケットのidxはゲーム単位で連番。
func (rb *RealtimeBroadcaster) Emit(id string, state model.GameState, event string, message *string, fromIdx *int, toIdx *int, bubbleIdx *int) {
	gameLogInterface, exists := rb.data.Load(id)
	if !exists {
		return
	}
	gameLog := gameLogInterface.(*RealtimeBroadcasterLog)

	gameLog.logsMu.Lock()
	gameLog.packetIdx++
	packet := model.BroadcastPacket{
		Id:        id,
		Idx:       gameLog.packetIdx,
		Day:       state.Day,
		IsDay:     state.IsDaytime,
		Agents:    state.Agents,
		Event:     event,
		Message:   message,
		FromIdx:   fromIdx,
		ToIdx:     toIdx,
		BubbleIdx: bubbleIdx,
		Timestamp: time.Now().Unix(),
	}
	data, err := json.Marshal(packet)
	if err != nil {
		gameLog.logsMu.Unlock()
		slog.Error("パケットのJSON化に失敗しました", "error", err)
		return
	}
	// 毎パケットでの全書き換え（パケット数に対しO(n^2)）を避け、新規行のみ追記する。
	// 結果のファイルは従来と同一（"\n"区切り）。1ゲームの書き込みは所有goroutineで直列。
	firstLine := len(gameLog.logs) == 0
	gameLog.logs = append(gameLog.logs, string(data))
	gameLog.updatedAt = time.Now()
	filename := gameLog.filename
	gameLog.logsMu.Unlock()

	rb.appendGameFileLine(filename, string(data), firstLine)
	rb.writeGamesListFile()
	slog.Info("JSONLファイルにブロードキャストを保存しました", "game_id", id)
}

func (rb *RealtimeBroadcaster) appendGameFileLine(filename string, line string, firstLine bool) {
	filePath := filepath.Join(rb.config.OutputDir, fmt.Sprintf("%s.jsonl", filename))
	flag := os.O_APPEND | os.O_CREATE | os.O_WRONLY
	content := "\n" + line
	if firstLine {
		// ファイルを新規化し、先頭行は改行なしで書く。
		flag = os.O_CREATE | os.O_TRUNC | os.O_WRONLY
		content = line
	}
	file, err := os.OpenFile(filePath, flag, 0644)
	if err != nil {
		slog.Error("ゲームファイルのオープンに失敗しました", "error", err, "path", filePath)
		return
	}
	defer file.Close()
	if _, err := file.WriteString(content); err != nil {
		slog.Error("ゲームファイルへの追記に失敗しました", "error", err, "path", filePath)
	}
}

func (rb *RealtimeBroadcaster) writeGamesListFile() {
	type Item struct {
		ID        string    `json:"id"`
		Filename  string    `json:"filename"`
		UpdatedAt time.Time `json:"updated_at"`
	}
	items := make([]Item, 0)
	rb.data.Range(func(_, value any) bool {
		gameLog := value.(*RealtimeBroadcasterLog)
		// updatedAtは所有ゲームのgoroutineがlogsMu下で更新するため、別ゲームからの
		// 一覧更新でも安全に読めるようロックする。
		gameLog.logsMu.Lock()
		item := Item{
			ID:        gameLog.id,
			Filename:  gameLog.filename,
			UpdatedAt: gameLog.updatedAt,
		}
		gameLog.logsMu.Unlock()
		items = append(items, item)
		return true
	})

	data, err := json.Marshal(items)
	if err != nil {
		slog.Error("ゲーム一覧のJSON生成に失敗しました", "error", err)
		return
	}
	filePath := filepath.Join(rb.config.OutputDir, "games.json")
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		slog.Error("ゲーム一覧ファイルの作成に失敗しました", "error", err)
		return
	}
	slog.Info("ゲーム一覧ファイルを更新しました", "path", filePath)
}

func (rb *RealtimeBroadcaster) writeGameFile(filename string, logs []string) {
	filePath := filepath.Join(rb.config.OutputDir, fmt.Sprintf("%s.jsonl", filename))
	content := strings.Join(logs, "\n")
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		slog.Error("ゲームファイルの保存に失敗しました", "error", err, "path", filePath)
		return
	}
	slog.Info("ゲームファイルを保存しました", "path", filePath)
}
