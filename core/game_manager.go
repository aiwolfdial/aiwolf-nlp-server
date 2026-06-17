package core

import (
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/aiwolfdial/aiwolf-nlp-server/logic"
	"github.com/aiwolfdial/aiwolf-nlp-server/model"
	"github.com/aiwolfdial/aiwolf-nlp-server/observer"
)

// 待機部屋・マッチオプティマイザ・進行中ゲームの登録簿を持ち、マッチングとゲームの
// 生成・破棄を担う。終了したゲームは登録簿から取り除かれるため、登録簿は実行中の
// ゲームのみを表す。
type GameManager struct {
	config          model.Config
	gameSetting     *model.Setting
	waitingRoom     *WaitingRoom
	matchOptimizer  *MatchOptimizer
	observerFactory func() observer.GameObserver
	games           sync.Map
	shuttingDown    atomic.Bool
}

type gameEntry struct {
	game      *logic.Game
	agents    []model.AgentView
	startedAt time.Time
}

func (e *gameEntry) snapshot() model.GameSnapshot {
	agents := make([]model.AgentView, len(e.agents))
	copy(agents, e.agents)
	return model.GameSnapshot{
		ID:       e.game.GetID(),
		Finished: e.game.IsFinished(),
		Agents:   agents,
	}
}

func NewGameManager(config model.Config, gameSetting *model.Setting, waitingRoom *WaitingRoom, matchOptimizer *MatchOptimizer, observerFactory func() observer.GameObserver) *GameManager {
	return &GameManager{
		config:          config,
		gameSetting:     gameSetting,
		waitingRoom:     waitingRoom,
		matchOptimizer:  matchOptimizer,
		observerFactory: observerFactory,
	}
}

// 接続を待機部屋へ追加し、マッチが成立すればゲームを生成して開始する。
func (m *GameManager) TryStartGame(conn model.Connection) {
	m.waitingRoom.AddConnection(conn.TeamName, conn)

	var game *logic.Game
	if m.matchOptimizer != nil {
		m.waitingRoom.connections.Range(func(key, value any) bool {
			team := key.(string)
			m.matchOptimizer.updateTeam(team)
			return true
		})
		matches := m.matchOptimizer.getMatches()
		roleMapConns, err := m.waitingRoom.GetConnectionsWithMatchOptimizer(matches)
		if err != nil {
			slog.Error("待機部屋からの接続の取得に失敗しました", "error", err)
			return
		}
		game = logic.NewGameWithRole(&m.config, m.gameSetting, roleMapConns)
	} else {
		connections, err := m.waitingRoom.GetConnections()
		if err != nil {
			slog.Error("待機部屋からの接続の取得に失敗しました", "error", err)
			return
		}
		game = logic.NewGame(&m.config, m.gameSetting, connections)
	}

	game.SetObserver(m.observerFactory())
	m.games.Store(game.GetID(), &gameEntry{
		game:      game,
		agents:    game.AgentViews(),
		startedAt: time.Now(),
	})

	go func() {
		winSide := game.Start()
		if m.matchOptimizer != nil {
			if winSide != model.T_NONE {
				m.matchOptimizer.setMatchEnd(game.GetRoleTeamNamesMap())
			} else {
				m.matchOptimizer.setMatchWeight(game.GetRoleTeamNamesMap(), 0)
			}
		}
		// 終了したゲームを登録簿から取り除く。これがないとプロセス終了まで残り続ける。
		m.games.Delete(game.GetID())
	}()
}

func (m *GameManager) ActiveCount() int {
	count := 0
	m.games.Range(func(_, _ any) bool {
		count++
		return true
	})
	return count
}

func (m *GameManager) ListGames() []model.GameSnapshot {
	snapshots := make([]model.GameSnapshot, 0)
	m.games.Range(func(_, value any) bool {
		snapshots = append(snapshots, value.(*gameEntry).snapshot())
		return true
	})
	return snapshots
}

func (m *GameManager) GetGame(id string) (model.GameSnapshot, bool) {
	value, ok := m.games.Load(id)
	if !ok {
		return model.GameSnapshot{}, false
	}
	return value.(*gameEntry).snapshot(), true
}

func (m *GameManager) BeginShutdown() {
	m.shuttingDown.Store(true)
}

func (m *GameManager) IsShuttingDown() bool {
	return m.shuttingDown.Load()
}

// 終了したゲームは自ら登録簿から抜けるため、登録簿が空になるまで待つ。
func (m *GameManager) WaitAllFinished() {
	for m.ActiveCount() > 0 {
		time.Sleep(15 * time.Second)
	}
	slog.Info("全てのゲームが終了しました")
}
