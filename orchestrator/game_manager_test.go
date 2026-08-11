package orchestrator

import (
	"testing"
	"time"

	"github.com/aiwolfdial/aiwolf-nlp-server/matchmaking"
	"github.com/aiwolfdial/aiwolf-nlp-server/model"
	"github.com/aiwolfdial/aiwolf-nlp-server/observer/teamhealth"
)

// 待機部屋の実体は接続に実 WebSocket を要求するため、チーム名だけを返す差し替えを使う。
type stubWaitingRoom struct {
	teams []string
}

func (s *stubWaitingRoom) AddConnection(string, model.Connection) {}
func (s *stubWaitingRoom) Teams() []string                        { return s.teams }
func (s *stubWaitingRoom) GetConnections() ([]model.Connection, error) {
	return nil, nil
}
func (s *stubWaitingRoom) GetConnectionsWithMatchOptimizer([]map[model.Role][]string) (map[model.Role][]model.Connection, error) {
	return nil, nil
}

type stallCall struct {
	idle      time.Duration
	active    int
	connected []string
	awaiting  []string
}

type progressCall struct {
	done, total, active int
}

type stubNotifier struct {
	stalls    []stallCall
	progress  []progressCall
	aborts    int
	quarantin int
}

func (n *stubNotifier) NotifyGameAborted(string, []string, []string) { n.aborts++ }
func (n *stubNotifier) NotifyTeamQuarantined([]teamhealth.Snapshot)  { n.quarantin++ }
func (n *stubNotifier) NotifyMatchmakingStalled(idle time.Duration, active int, connected []string, awaiting []string) {
	n.stalls = append(n.stalls, stallCall{idle: idle, active: active, connected: connected, awaiting: awaiting})
}
func (n *stubNotifier) NotifyProgress(done, total, active int) {
	n.progress = append(n.progress, progressCall{done: done, total: total, active: active})
}

func testOptimizer(t *testing.T, gameCount int) *matchmaking.MatchOptimizer {
	t.Helper()
	var config model.Config
	config.Game.AgentCount = 5
	config.Logic.Roles = map[int]map[string]int{
		5: {"WEREWOLF": 1, "POSSESSED": 1, "SEER": 1, "VILLAGER": 2},
	}
	config.Matching.TeamCount = 5
	config.Matching.GameCount = gameCount
	config.Matching.OutputPath = t.TempDir() + "/match_optimizer.json"
	mo, err := matchmaking.NewMatchOptimizerFromConfig(config)
	if err != nil {
		t.Fatalf("マッチオプティマイザの作成に失敗しました: %v", err)
	}
	return mo
}

func newTestManager(t *testing.T, room WaitingRoom, mo *matchmaking.MatchOptimizer, obs Observability) *GameManager {
	t.Helper()
	m := NewGameManager(model.Config{}, nil, room, mo, nil)
	m.SetObservability(obs)
	return m
}

// 消化数が節目を飛び越えても、その節目の通知が消えないこと。
// 同時に終了したゲームが両方とも節目より先の値を観測するため、剰余で判定すると落ちる。
func TestNotifyMilestoneDoesNotSkipWhenProgressJumps(t *testing.T) {
	mo := testOptimizer(t, 30)
	notifier := &stubNotifier{}
	m := newTestManager(t, &stubWaitingRoom{}, mo, Observability{Notifier: notifier, MilestoneEvery: 10})

	// 9件消化した状態から、2件同時に終わって11件になった状況を作る。
	for range 11 {
		mo.EndedMatches = append(mo.EndedMatches, map[model.Role][]int{})
	}
	m.notifyMilestone()

	if len(notifier.progress) != 1 {
		t.Fatalf("節目をまたいだのに通知されませんでした: %v", notifier.progress)
	}
	if notifier.progress[0].done != 11 {
		t.Fatalf("通知された消化数が想定と異なります: %v", notifier.progress[0].done)
	}
}

// 同じ節目では二度通知せず、次の節目に達したら通知すること。
func TestNotifyMilestoneFiresOncePerThreshold(t *testing.T) {
	mo := testOptimizer(t, 30)
	notifier := &stubNotifier{}
	m := newTestManager(t, &stubWaitingRoom{}, mo, Observability{Notifier: notifier, MilestoneEvery: 10})

	for range 10 {
		mo.EndedMatches = append(mo.EndedMatches, map[model.Role][]int{})
	}
	m.notifyMilestone()
	// 同じ節目の中で更に消化しても通知しない。
	for range 5 {
		mo.EndedMatches = append(mo.EndedMatches, map[model.Role][]int{})
	}
	m.notifyMilestone()
	if len(notifier.progress) != 1 {
		t.Fatalf("同じ節目で複数回通知されました: %v", notifier.progress)
	}
	// 次の節目に達したら通知する。
	for range 5 {
		mo.EndedMatches = append(mo.EndedMatches, map[model.Role][]int{})
	}
	m.notifyMilestone()
	if len(notifier.progress) != 2 {
		t.Fatalf("次の節目で通知されませんでした: %v", notifier.progress)
	}
}

func TestNotifyMilestoneIsDisabledWhenEveryIsZero(t *testing.T) {
	mo := testOptimizer(t, 30)
	notifier := &stubNotifier{}
	m := newTestManager(t, &stubWaitingRoom{}, mo, Observability{Notifier: notifier})
	mo.EndedMatches = append(mo.EndedMatches, map[model.Role][]int{})
	m.notifyMilestone()
	if len(notifier.progress) != 0 {
		t.Fatalf("無効なのに通知されました: %v", notifier.progress)
	}
}

// 大会中はほぼ常に何かが走っているため、対戦中でも待たされているチームがいれば通知する。
func TestCheckStallNotifiesWhileGamesAreRunning(t *testing.T) {
	notifier := &stubNotifier{}
	room := &stubWaitingRoom{teams: []string{"beta", "alpha"}}
	m := newTestManager(t, room, nil, Observability{Notifier: notifier})
	m.games.Store("running", &gameEntry{})
	m.lastMatchUnix.Store(time.Now().Add(-10 * time.Minute).Unix())

	m.checkStall(time.Minute)

	if len(notifier.stalls) != 1 {
		t.Fatalf("進行中のゲームがあると通知されません: %v", notifier.stalls)
	}
	if notifier.stalls[0].active != 1 {
		t.Fatalf("実行中の試合数が伝わっていません: %v", notifier.stalls[0].active)
	}
	// 出力がぶれないよう並べ替えられていること。
	if got := notifier.stalls[0].connected; got[0] != "alpha" || got[1] != "beta" {
		t.Fatalf("接続中のチームが並べ替えられていません: %v", got)
	}
}

// 誰も待っていなければ、単に接続が無いだけなので通知しない。
func TestCheckStallStaysQuietWhenNobodyIsWaiting(t *testing.T) {
	notifier := &stubNotifier{}
	m := newTestManager(t, &stubWaitingRoom{}, nil, Observability{Notifier: notifier})
	m.lastMatchUnix.Store(time.Now().Add(-10 * time.Minute).Unix())

	m.checkStall(time.Minute)

	if len(notifier.stalls) != 0 {
		t.Fatalf("待機チームが無いのに通知されました: %v", notifier.stalls)
	}
}

func TestCheckStallStaysQuietBeforeThreshold(t *testing.T) {
	notifier := &stubNotifier{}
	room := &stubWaitingRoom{teams: []string{"alpha"}}
	m := newTestManager(t, room, nil, Observability{Notifier: notifier})
	m.lastMatchUnix.Store(time.Now().Unix())

	m.checkStall(time.Minute)

	if len(notifier.stalls) != 0 {
		t.Fatalf("閾値前に通知されました: %v", notifier.stalls)
	}
}

// 同じ停滞で通知を繰り返さないこと。復旧の通知は別途マッチ成立時にリセットされる。
func TestCheckStallNotifiesOnlyOncePerEpisode(t *testing.T) {
	notifier := &stubNotifier{}
	room := &stubWaitingRoom{teams: []string{"alpha"}}
	m := newTestManager(t, room, nil, Observability{Notifier: notifier})
	m.lastMatchUnix.Store(time.Now().Add(-10 * time.Minute).Unix())

	m.checkStall(time.Minute)
	m.checkStall(time.Minute)

	if len(notifier.stalls) != 1 {
		t.Fatalf("同じ停滞で繰り返し通知されました: %v", len(notifier.stalls))
	}
}

func TestCheckStallStaysQuietWhileShuttingDown(t *testing.T) {
	notifier := &stubNotifier{}
	room := &stubWaitingRoom{teams: []string{"alpha"}}
	m := newTestManager(t, room, nil, Observability{Notifier: notifier})
	m.lastMatchUnix.Store(time.Now().Add(-10 * time.Minute).Unix())
	m.BeginShutdown()

	m.checkStall(time.Minute)

	if len(notifier.stalls) != 0 {
		t.Fatalf("シャットダウン中に通知されました: %v", notifier.stalls)
	}
}

// 対戦表に載っているのに接続していないチームが接続待ちとして出ること。
func TestWaitingTeamsSplitsConnectedAndAwaiting(t *testing.T) {
	mo := testOptimizer(t, 5)
	for _, team := range []string{"alpha", "beta", "gamma"} {
		mo.UpdateTeam(team)
	}
	m := newTestManager(t, &stubWaitingRoom{teams: []string{"beta"}}, mo, Observability{})

	connected, awaiting := m.waitingTeams()

	if len(connected) != 1 || connected[0] != "beta" {
		t.Fatalf("接続中のチームが想定と異なります: %v", connected)
	}
	if len(awaiting) != 2 || awaiting[0] != "alpha" || awaiting[1] != "gamma" {
		t.Fatalf("接続待ちのチームが想定と異なります: %v", awaiting)
	}
}

// 対戦表が無い場合は名簿そのものが存在しないため、接続待ちを出せない。
func TestWaitingTeamsWithoutOptimizer(t *testing.T) {
	m := newTestManager(t, &stubWaitingRoom{teams: []string{"alpha"}}, nil, Observability{})

	connected, awaiting := m.waitingTeams()

	if len(connected) != 1 {
		t.Fatalf("接続中のチームが取れていません: %v", connected)
	}
	if awaiting != nil {
		t.Fatalf("名簿が無いのに接続待ちが出ました: %v", awaiting)
	}
}

// 異常終了のときだけ通知し、引き分けでは通知しないこと。
func TestRecordOutcomeNotifiesOnlyOnErrorAbort(t *testing.T) {
	tracker := teamhealth.New(model.TeamHealthConfig{MinGames: 1})
	notifier := &stubNotifier{}
	m := newTestManager(t, &stubWaitingRoom{}, nil, Observability{TeamHealth: tracker, Notifier: notifier})

	tracker.OnGameStart("g1", []model.AgentView{{TeamName: "alpha"}}, model.GameState{})
	m.recordOutcome("g1", model.F_MAX_DAY)
	if notifier.aborts != 0 {
		t.Fatalf("引き分けで異常終了が通知されました: %v", notifier.aborts)
	}

	tracker.OnGameStart("g2", []model.AgentView{{TeamName: "alpha"}}, model.GameState{})
	m.recordOutcome("g2", model.F_ERROR)
	if notifier.aborts != 1 {
		t.Fatalf("異常終了が通知されませんでした: %v", notifier.aborts)
	}

	snap, ok := tracker.Get("alpha")
	if !ok {
		t.Fatal("チームの記録がありません")
	}
	if snap.AbortedGames != 1 {
		t.Fatalf("異常終了の集計が想定と異なります: %v", snap.AbortedGames)
	}
}

// 通知先が未設定でも集計だけは進み、nil 参照で落ちないこと。
func TestRecordOutcomeWithoutNotifier(t *testing.T) {
	tracker := teamhealth.New(model.TeamHealthConfig{MinGames: 1})
	m := newTestManager(t, &stubWaitingRoom{}, nil, Observability{TeamHealth: tracker})
	tracker.OnGameStart("g1", []model.AgentView{{TeamName: "alpha"}}, model.GameState{})
	m.recordOutcome("g1", model.F_ERROR)

	if snap, ok := tracker.Get("alpha"); !ok || snap.Games != 1 {
		t.Fatalf("通知先が無いと集計されません: %v", snap)
	}
}
