package teamhealth

import (
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/aiwolfdial/aiwolf-nlp-server/model"
)

// 時刻を固定した Tracker を返す。隔離の期限判定を実時間に依存させないため。
func newFixed(t *testing.T, config model.TeamHealthConfig) (*Tracker, *time.Time) {
	t.Helper()
	now := time.Date(2026, 8, 11, 12, 0, 0, 0, time.UTC)
	tracker := New(config)
	tracker.now = func() time.Time { return now }
	return tracker, &now
}

func agents(teams ...string) []model.AgentView {
	views := make([]model.AgentView, 0, len(teams))
	for i, team := range teams {
		views = append(views, model.AgentView{Idx: i, TeamName: team})
	}
	return views
}

// エラーが1件も無ければ重みは下がらない。
func TestHealthyTeamKeepsFullWeight(t *testing.T) {
	tracker, _ := newFixed(t, model.TeamHealthConfig{MinGames: 1})
	for i := range 3 {
		id := string(rune('a' + i))
		tracker.OnGameStart(id, agents("alpha", "beta"), model.GameState{})
		tracker.OnResponse(id, model.AgentView{TeamName: "alpha"}, "ok", nil)
		tracker.FinishGame(id, false)
	}
	if got := tracker.TeamWeight("alpha"); got != 1.0 {
		t.Fatalf("失敗していないチームの重みが下がりました: %v", got)
	}
	if tracker.IsQuarantined("alpha") {
		t.Fatal("失敗していないチームが隔離されました")
	}
}

// 実績が min_games に満たないうちは減点しない。1試合の事故で除外しないため。
func TestWeightIsNotAppliedBeforeMinGames(t *testing.T) {
	tracker, _ := newFixed(t, model.TeamHealthConfig{MinGames: 3})
	tracker.OnGameStart("g1", agents("alpha"), model.GameState{})
	tracker.OnAgentFatal("g1", model.AgentView{TeamName: "alpha"}, errors.New("切断"))
	tracker.FinishGame("g1", true)

	if got := tracker.TeamWeight("alpha"); got != 1.0 {
		t.Fatalf("実績が足りないのに減点されました: %v", got)
	}
	if tracker.IsQuarantined("alpha") {
		t.Fatal("実績が足りないのに隔離されました")
	}
}

// 脱落が続けば重みが下がり、閾値を超えたところで隔離される。
func TestRepeatedFatalsQuarantineTeam(t *testing.T) {
	tracker, now := newFixed(t, model.TeamHealthConfig{
		MinGames:           2,
		QuarantineRate:     0.5,
		QuarantineDuration: 10 * time.Minute,
	})

	var quarantined []Snapshot
	for i := range 3 {
		id := string(rune('a' + i))
		tracker.OnGameStart(id, agents("alpha", "beta"), model.GameState{})
		tracker.OnAgentFatal(id, model.AgentView{TeamName: "alpha"}, errors.New("切断"))
		outcome := tracker.FinishGame(id, true)
		quarantined = append(quarantined, outcome.Quarantined...)
	}

	if len(quarantined) == 0 {
		t.Fatal("失敗し続けたチームが隔離されませんでした")
	}
	if quarantined[0].Team != "alpha" {
		t.Fatalf("隔離されたチームが想定と異なります: %v", quarantined[0].Team)
	}
	if !tracker.IsQuarantined("alpha") {
		t.Fatal("隔離状態が維持されていません")
	}
	if w := tracker.TeamWeight("alpha"); w >= 1.0 {
		t.Fatalf("失敗したチームの重みが下がっていません: %v", w)
	}
	// 同じゲームに居ただけの beta は自分の失敗ではないので隔離されない。
	if tracker.IsQuarantined("beta") {
		t.Fatal("失敗していない同席チームまで隔離されました")
	}

	// 期限を過ぎれば手を入れずに復帰する。
	*now = now.Add(11 * time.Minute)
	if tracker.IsQuarantined("alpha") {
		t.Fatal("期限を過ぎても隔離が解除されません")
	}
}

// 隔離中に更に失敗しても期限は延びない。延びると失敗が失敗を呼んで復帰できなくなる。
func TestQuarantineDeadlineIsNotExtended(t *testing.T) {
	tracker, now := newFixed(t, model.TeamHealthConfig{
		MinGames:           1,
		QuarantineRate:     0.1,
		QuarantineDuration: 10 * time.Minute,
	})
	fail := func(id string) []Snapshot {
		tracker.OnGameStart(id, agents("alpha"), model.GameState{})
		tracker.OnAgentFatal(id, model.AgentView{TeamName: "alpha"}, errors.New("切断"))
		return tracker.FinishGame(id, true).Quarantined
	}

	first := fail("g1")
	if len(first) != 1 || first[0].QuarantinedUntil == nil {
		t.Fatalf("隔離されませんでした: %v", first)
	}
	deadline := *first[0].QuarantinedUntil

	// 隔離中に更に失敗させる。通知は繰り返さず、期限も動かないこと。
	*now = now.Add(5 * time.Minute)
	if again := fail("g2"); len(again) != 0 {
		t.Fatalf("隔離中に通知が繰り返されました: %v", again)
	}
	snap, _ := tracker.Get("alpha")
	if snap.QuarantinedUntil == nil || *snap.QuarantinedUntil != deadline {
		t.Fatalf("隔離の期限が延長されました: %v -> %v", deadline, snap.QuarantinedUntil)
	}
	if snap.QuarantineCount != 1 {
		t.Fatalf("隔離回数が二重に数えられています: %v", snap.QuarantineCount)
	}

	// 期限を過ぎれば復帰し、その後の失敗で改めて隔離される。
	*now = now.Add(6 * time.Minute)
	if tracker.IsQuarantined("alpha") {
		t.Fatal("期限を過ぎても復帰しません")
	}
	if len(fail("g3")) != 1 {
		t.Fatal("復帰後の失敗で再隔離されませんでした")
	}
}

// 重みは weight_floor より下がらない。0 にすると事実上の永久除外になるため。
func TestWeightFloorIsRespected(t *testing.T) {
	tracker, _ := newFixed(t, model.TeamHealthConfig{
		MinGames:       1,
		WeightFloor:    0.2,
		QuarantineRate: 2.0, // 隔離させずに重みだけを見る
	})
	for i := range 5 {
		id := string(rune('a' + i))
		tracker.OnGameStart(id, agents("alpha"), model.GameState{})
		tracker.OnResponse(id, model.AgentView{TeamName: "alpha"}, "", errors.New("タイムアウト"))
		tracker.OnAgentFatal(id, model.AgentView{TeamName: "alpha"}, errors.New("切断"))
		tracker.FinishGame(id, true)
	}
	if got := tracker.TeamWeight("alpha"); got != 0.2 {
		t.Fatalf("重みが下限を下回るか届いていません: %v", got)
	}
}

// ウィンドウから溢れた古い失敗は評価から外れ、直近の成績で回復できる。
func TestWindowDropsOldFailures(t *testing.T) {
	tracker, _ := newFixed(t, model.TeamHealthConfig{
		Window:         2,
		MinGames:       1,
		QuarantineRate: 2.0,
	})
	tracker.OnGameStart("g1", agents("alpha"), model.GameState{})
	tracker.OnAgentFatal("g1", model.AgentView{TeamName: "alpha"}, errors.New("切断"))
	tracker.FinishGame("g1", true)

	for _, id := range []string{"g2", "g3"} {
		tracker.OnGameStart(id, agents("alpha"), model.GameState{})
		tracker.OnResponse(id, model.AgentView{TeamName: "alpha"}, "ok", nil)
		tracker.FinishGame(id, false)
	}

	snap, ok := tracker.Get("alpha")
	if !ok {
		t.Fatal("チームのスナップショットが取れませんでした")
	}
	if snap.Games != 2 {
		t.Fatalf("ウィンドウ外の記録が残っています: %v", snap.Games)
	}
	if snap.FatalGames != 0 {
		t.Fatalf("ウィンドウ外の脱落が数えられています: %v", snap.FatalGames)
	}
	if got := tracker.TeamWeight("alpha"); got != 1.0 {
		t.Fatalf("直近が健全なのに重みが戻りません: %v", got)
	}
}

// 自己対戦では同じチームが複数席を占めるが、1ゲームは1件として数える。
func TestSelfMatchCountsGameOnce(t *testing.T) {
	tracker, _ := newFixed(t, model.TeamHealthConfig{MinGames: 1})
	tracker.OnGameStart("g1", agents("alpha", "alpha", "alpha"), model.GameState{})
	outcome := tracker.FinishGame("g1", false)

	if len(outcome.Teams) != 1 {
		t.Fatalf("同一チームが重複して数えられています: %v", outcome.Teams)
	}
	snap, _ := tracker.Get("alpha")
	if snap.Games != 1 {
		t.Fatalf("1ゲームが複数件として記録されています: %v", snap.Games)
	}
}

// チーム名はクライアントが名乗った文字列なので、名前を変え続けられると記録が
// 際限なく増える。上限を超えたら最終参加が古いものから捨てること。
func TestTrackedTeamsAreBounded(t *testing.T) {
	tracker, now := newFixed(t, model.TeamHealthConfig{
		MinGames:           1,
		QuarantineRate:     0.1,
		QuarantineDuration: time.Hour,
	})

	// 隔離中のチームは判定に使うため、捨てられずに残ること。
	tracker.OnGameStart("q", agents("quarantined"), model.GameState{})
	tracker.OnAgentFatal("q", model.AgentView{TeamName: "quarantined"}, errors.New("切断"))
	tracker.FinishGame("q", true)
	if !tracker.IsQuarantined("quarantined") {
		t.Fatal("前提となる隔離が発生していません")
	}

	for i := range maxTrackedTeams + 200 {
		*now = now.Add(time.Second)
		id := "g" + strconv.Itoa(i)
		team := "team" + strconv.Itoa(i)
		tracker.OnGameStart(id, agents(team), model.GameState{})
		tracker.FinishGame(id, false)
	}

	if got := len(tracker.Snapshots()); got > maxTrackedTeams {
		t.Fatalf("チーム記録が上限を超えています: %v", got)
	}
	if _, ok := tracker.Get("quarantined"); !ok {
		t.Fatal("隔離中のチームが捨てられました")
	}
	// 最後に参加したチームは残っていること。
	last := "team" + strconv.Itoa(maxTrackedTeams+199)
	if _, ok := tracker.Get(last); !ok {
		t.Fatalf("直近のチームが捨てられました: %s", last)
	}
	// 最初のほうのチームは捨てられていること。
	if _, ok := tracker.Get("team0"); ok {
		t.Fatal("最も古いチームが捨てられていません")
	}
}

// max_day 到達による引き分けは異常終了として数えない。
func TestDrawIsNotCountedAsAbort(t *testing.T) {
	tracker, _ := newFixed(t, model.TeamHealthConfig{MinGames: 1})
	tracker.OnGameStart("g1", agents("alpha"), model.GameState{})
	tracker.FinishGame("g1", false)

	snap, _ := tracker.Get("alpha")
	if snap.AbortedGames != 0 {
		t.Fatalf("引き分けが異常終了として数えられました: %v", snap.AbortedGames)
	}
	if got := tracker.TeamWeight("alpha"); got != 1.0 {
		t.Fatalf("引き分けで減点されました: %v", got)
	}
}
