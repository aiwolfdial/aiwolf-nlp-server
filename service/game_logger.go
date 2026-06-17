package service

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/aiwolfdial/aiwolf-nlp-server/model"
)

type GameLogger struct {
	config model.GameLoggerConfig
	data   sync.Map
}

type GameLog struct {
	id       string
	filename string
	agents   []any
	logs     []string
	mu       sync.Mutex
}

func NewGameLogger(config model.Config) *GameLogger {
	return &GameLogger{
		config: config.GameLogger,
	}
}

func (g *GameLogger) TrackStartGame(id string, agents []model.AgentView) {
	data := &GameLog{
		id:     id,
		logs:   make([]string, 0),
		agents: make([]any, 0),
	}

	for _, agent := range agents {
		data.agents = append(data.agents,
			map[string]any{
				"idx":  agent.Idx,
				"team": agent.TeamName,
				"name": agent.OriginalName,
				"role": agent.Role,
			},
		)
	}

	filename := strings.ReplaceAll(g.config.Filename, "{game_id}", data.id)
	filename = strings.ReplaceAll(filename, "{timestamp}", fmt.Sprintf("%d", time.Now().Unix()))

	teams := make([]string, 0)
	for _, agent := range data.agents {
		team := agent.(map[string]any)["team"].(string)
		teams = append(teams, team)
	}
	sort.Strings(teams)
	filename = strings.ReplaceAll(filename, "{teams}", strings.Join(teams, "_"))

	data.filename = filename
	g.data.Store(id, data)
}

func (g *GameLogger) TrackEndGame(id string) {
	if _, exists := g.data.Load(id); exists {
		g.saveLog(id)
		g.data.Delete(id)
	}
}

func (g *GameLogger) AppendLog(id string, log string) {
	if dataInterface, exists := g.data.Load(id); exists {
		data := dataInterface.(*GameLog)

		data.mu.Lock()
		data.logs = append(data.logs, log)
		logsCopy := make([]string, len(data.logs))
		copy(logsCopy, data.logs)
		data.mu.Unlock()

		g.saveLogWithData(data.filename, logsCopy)
	}
}

func (g *GameLogger) saveLog(id string) {
	if dataInterface, exists := g.data.Load(id); exists {
		data := dataInterface.(*GameLog)

		data.mu.Lock()
		logsCopy := make([]string, len(data.logs))
		copy(logsCopy, data.logs)
		filename := data.filename
		data.mu.Unlock()

		g.saveLogWithData(filename, logsCopy)
	}
}

func (g *GameLogger) saveLogWithData(filename string, logs []string) {
	str := strings.Join(logs, "\n")

	if _, err := os.Stat(g.config.OutputDir); os.IsNotExist(err) {
		os.MkdirAll(g.config.OutputDir, 0755)
	}

	filePath := filepath.Join(g.config.OutputDir, fmt.Sprintf("%s.log", filename))
	file, err := os.Create(filePath)
	if err != nil {
		return
	}
	defer file.Close()

	file.WriteString(str)
}
