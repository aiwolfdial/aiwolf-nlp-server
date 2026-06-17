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

// GameManager owns the lifecycle of games: the waiting room, the optional match
// optimizer, and the registry of running games. It contains the matchmaking
// logic that used to live inline in the WebSocket handler, and it exposes
// read-only snapshots for the API layer. Games are removed from the registry
// when they finish, so the registry reflects only active games.
type GameManager struct {
	config          model.Config
	gameSetting     *model.Setting
	waitingRoom     *WaitingRoom
	matchOptimizer  *MatchOptimizer // nil unless matching optimization is enabled
	observerFactory func() observer.GameObserver
	games           sync.Map // game id -> *gameEntry
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

// NewGameManager builds a GameManager. observerFactory produces a fresh
// composite observer for each game (so each game gets its own sinks wired).
func NewGameManager(config model.Config, gameSetting *model.Setting, waitingRoom *WaitingRoom, matchOptimizer *MatchOptimizer, observerFactory func() observer.GameObserver) *GameManager {
	return &GameManager{
		config:          config,
		gameSetting:     gameSetting,
		waitingRoom:     waitingRoom,
		matchOptimizer:  matchOptimizer,
		observerFactory: observerFactory,
	}
}

// TryStartGame registers a newly connected client and, if a match can be
// formed, creates and starts a game. The matchmaking trigger semantics are
// unchanged from the original inline implementation; only their location moved.
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
		// Remove the finished game from the registry; without this the map would
		// grow unbounded for the lifetime of the process.
		m.games.Delete(game.GetID())
	}()
}

// ActiveCount returns the number of games currently in the registry.
func (m *GameManager) ActiveCount() int {
	count := 0
	m.games.Range(func(_, _ any) bool {
		count++
		return true
	})
	return count
}

// ListGames returns a read-only snapshot of every active game.
func (m *GameManager) ListGames() []model.GameSnapshot {
	snapshots := make([]model.GameSnapshot, 0)
	m.games.Range(func(_, value any) bool {
		snapshots = append(snapshots, value.(*gameEntry).snapshot())
		return true
	})
	return snapshots
}

// GetGame returns a read-only snapshot of the game with the given id.
func (m *GameManager) GetGame(id string) (model.GameSnapshot, bool) {
	value, ok := m.games.Load(id)
	if !ok {
		return model.GameSnapshot{}, false
	}
	return value.(*gameEntry).snapshot(), true
}

// BeginShutdown marks the manager as draining so no new games are accepted.
func (m *GameManager) BeginShutdown() {
	m.shuttingDown.Store(true)
}

// IsShuttingDown reports whether the manager is draining.
func (m *GameManager) IsShuttingDown() bool {
	return m.shuttingDown.Load()
}

// WaitAllFinished blocks until every active game has finished. Because finished
// games remove themselves from the registry, this waits for the registry to
// drain to empty.
func (m *GameManager) WaitAllFinished() {
	for m.ActiveCount() > 0 {
		time.Sleep(15 * time.Second)
	}
	slog.Info("全てのゲームが終了しました")
}
