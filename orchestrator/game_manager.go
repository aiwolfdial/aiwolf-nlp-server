package orchestrator

import (
	"log/slog"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/aiwolfdial/aiwolf-nlp-server/logic"
	"github.com/aiwolfdial/aiwolf-nlp-server/matchmaking"
	"github.com/aiwolfdial/aiwolf-nlp-server/model"
	"github.com/aiwolfdial/aiwolf-nlp-server/observer"
	"github.com/aiwolfdial/aiwolf-nlp-server/observer/teamhealth"
)

// GameManager が使う通知先。service.SlackNotifier が実装する。
// 上位パッケージへ依存しないよう、必要なメソッドだけをここで宣言する。
type Notifier interface {
	NotifyGameAborted(gameID string, teams []string, fatalTeams []string)
	NotifyTeamQuarantined(snapshots []teamhealth.Snapshot)
	NotifyMatchmakingStalled(idle time.Duration, active int, connected []string, awaiting []string)
	NotifyProgress(done int, total int, active int)
}

// GameManager が待機部屋に求める操作。matchmaking.WaitingRoom が実装する。
// 実体は接続に実 WebSocket を要求するため、マッチングまわりの判断をテストできるよう挟む。
type WaitingRoom interface {
	AddConnection(team string, connection model.Connection)
	Teams() []string
	GetConnections() ([]model.Connection, error)
	GetConnectionsWithMatchOptimizer(matches []map[model.Role][]string) (map[model.Role][]model.Connection, error)
}

// 監視まわりの差し込み。すべて任意で、未設定なら従来どおりの挙動になる。
type Observability struct {
	TeamHealth     *teamhealth.Tracker
	Notifier       Notifier
	MilestoneEvery int
	StallThreshold time.Duration
}

// 待機部屋・マッチオプティマイザ・進行中ゲームの登録簿を持ち、マッチングとゲームの
// 生成・破棄を担う。終了したゲームは登録簿から取り除かれるため、登録簿は実行中の
// ゲームのみを表す。
type GameManager struct {
	config          model.Config
	gameSetting     *model.Setting
	waitingRoom     WaitingRoom
	matchOptimizer  *matchmaking.MatchOptimizer
	observerFactory func() observer.GameObserver
	games           sync.Map
	shuttingDown    atomic.Bool

	obs           Observability
	lastMatchUnix atomic.Int64
	stallNotified atomic.Bool
	milestoneMu   sync.Mutex
	lastMilestone int
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

func NewGameManager(config model.Config, gameSetting *model.Setting, waitingRoom WaitingRoom, matchOptimizer *matchmaking.MatchOptimizer, observerFactory func() observer.GameObserver) *GameManager {
	m := &GameManager{
		config:          config,
		gameSetting:     gameSetting,
		waitingRoom:     waitingRoom,
		matchOptimizer:  matchOptimizer,
		observerFactory: observerFactory,
	}
	// 停滞の起点は起動時刻。1件目が成立しないまま放置されている状態も検知したい。
	m.lastMatchUnix.Store(time.Now().Unix())
	return m
}

func (m *GameManager) SetObservability(o Observability) {
	m.obs = o
}

// 接続を待機部屋へ追加し、マッチが成立すればゲームを生成して開始する。
func (m *GameManager) TryStartGame(conn model.Connection) {
	m.waitingRoom.AddConnection(conn.TeamName, conn)

	var game *logic.Game
	if m.matchOptimizer != nil {
		for _, team := range m.waitingRoom.Teams() {
			m.matchOptimizer.UpdateTeam(team)
		}
		matches := m.matchOptimizer.GetMatches()
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
	// マッチが成立したので停滞の起点を進め、次の停滞をまた通知できるようにする。
	m.lastMatchUnix.Store(time.Now().Unix())
	m.stallNotified.Store(false)

	go func() {
		winSide := game.Start()
		if m.matchOptimizer != nil {
			if winSide != model.T_NONE {
				m.matchOptimizer.SetMatchEnd(game.GetRoleTeamNamesMap())
			} else {
				m.matchOptimizer.PenalizeMatch(game.GetRoleTeamNamesMap())
			}
		}
		m.recordOutcome(game.GetID(), game.FinishReason())
		// 終了したゲームを登録簿から取り除く。これがないとプロセス終了まで残り続ける。
		m.games.Delete(game.GetID())
	}()
}

// ゲームの確定結果をチームの成績へ積み、隔離や異常終了を通知する。
// 引き分けも winSide は T_NONE になるため、責任を問える打ち切りだけを異常終了として扱う。
func (m *GameManager) recordOutcome(id string, reason model.FinishReason) {
	abortedByError := reason == model.F_ERROR
	var outcome teamhealth.GameOutcome
	if m.obs.TeamHealth != nil {
		outcome = m.obs.TeamHealth.FinishGame(id, abortedByError)
	}
	if m.obs.Notifier == nil {
		return
	}
	if abortedByError {
		m.obs.Notifier.NotifyGameAborted(id, outcome.Teams, outcome.FatalTeams)
	}
	if len(outcome.Quarantined) > 0 {
		for _, s := range outcome.Quarantined {
			slog.Warn("チームを隔離しました", "team", s.Team, "failure_rate", s.FailureRate, "games", s.Games)
		}
		m.obs.Notifier.NotifyTeamQuarantined(outcome.Quarantined)
	}
	m.notifyMilestone()
}

// 一定試合数ごとに進捗を通知する。同じ節目を二度通知しないよう到達点を覚えておく。
func (m *GameManager) notifyMilestone() {
	if m.matchOptimizer == nil || m.obs.MilestoneEvery <= 0 {
		return
	}
	done, total := m.matchOptimizer.Progress()
	m.milestoneMu.Lock()
	// 節目をまたいだかどうかで判定する。剰余で見ると、同時終了で消化数が節目を
	// 飛び越えたときにその節目の通知が丸ごと消える。
	if done <= m.lastMilestone ||
		done/m.obs.MilestoneEvery == m.lastMilestone/m.obs.MilestoneEvery {
		m.milestoneMu.Unlock()
		return
	}
	m.lastMilestone = done
	m.milestoneMu.Unlock()
	m.obs.Notifier.NotifyProgress(done, total, m.ActiveCount())
}

// StartWatchdog はマッチが成立しない状態が続いていないかを定期的に確認する。
// 待機部屋にチームがいるのに1件も組めない状態は、放置すると誰も気づけないため通知する。
func (m *GameManager) StartWatchdog() {
	threshold := m.obs.StallThreshold
	if threshold <= 0 || m.obs.Notifier == nil {
		return
	}
	interval := max(threshold/2, 30*time.Second)
	go func() {
		for range time.Tick(interval) {
			m.checkStall(threshold)
		}
	}()
}

func (m *GameManager) checkStall(threshold time.Duration) {
	if m.IsShuttingDown() || m.stallNotified.Load() {
		return
	}
	idle := time.Since(time.Unix(m.lastMatchUnix.Load(), 0))
	if idle < threshold {
		return
	}
	connected, awaiting := m.waitingTeams()
	// 待機しているチームが1つも無ければ、誰も繋いでいないだけで停滞ではない。
	// 逆に進行中のゲームがあっても、待たされているチームがいるなら停滞として扱う。
	// 大会中はほぼ常に何かが走っているため、対戦中を除外すると永遠に発火しない。
	if len(connected) == 0 {
		return
	}
	active := m.ActiveCount()
	slog.Warn("マッチが成立していません", "idle", idle.String(),
		"active", active, "connected", len(connected), "awaiting", len(awaiting))
	m.stallNotified.Store(true)
	m.obs.Notifier.NotifyMatchmakingStalled(idle, active, connected, awaiting)
}

// 待機部屋にいるチームと、対戦表に載っているのに接続していないチームに分ける。
// マッチが組めない原因は後者にあるため、名前が分からないと対応できない。
func (m *GameManager) waitingTeams() (connected []string, awaiting []string) {
	connected = m.waitingRoom.Teams()
	sort.Strings(connected)
	if m.matchOptimizer == nil {
		// 対戦表が無い場合は名簿そのものが存在せず、誰を待っているかを特定できない。
		return connected, nil
	}
	present := make(map[string]bool, len(connected))
	for _, team := range connected {
		present[team] = true
	}
	for _, team := range m.matchOptimizer.Teams() {
		if !present[team] {
			awaiting = append(awaiting, team)
		}
	}
	return connected, awaiting
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
