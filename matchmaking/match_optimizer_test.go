package matchmaking

import (
	"testing"
	"time"

	"github.com/aiwolfdial/aiwolf-nlp-server/model"
)

// TeamWeight と IsQuarantined だけを差し替える最小のスコアラ。
type stubScorer struct {
	weights     map[string]float64
	quarantined map[string]bool
}

func (s stubScorer) TeamWeight(team string) float64 {
	if w, ok := s.weights[team]; ok {
		return w
	}
	return 1.0
}

func (s stubScorer) IsQuarantined(team string) bool { return s.quarantined[team] }

// idx 0..3 を alpha/beta/gamma/delta に対応させ、2件のマッチを持つ状態を作る。
// match0 は alpha 対 beta、match1 は gamma 対 delta。
func newTestOptimizer() *MatchOptimizer {
	return &MatchOptimizer{
		TeamCount: 4,
		GameCount: 2,
		IdxTeamMap: map[int]string{
			0: "alpha", 1: "beta", 2: "gamma", 3: "delta",
		},
		ScheduledMatches: []model.MatchWeight{
			{RoleIdxs: map[model.Role][]int{model.R_WEREWOLF: {0}, model.R_VILLAGER: {1}}, Weight: 1.0},
			{RoleIdxs: map[model.Role][]int{model.R_WEREWOLF: {2}, model.R_VILLAGER: {3}}, Weight: 1.0},
		},
	}
}

func firstWerewolf(match map[model.Role][]string) string {
	if teams := match[model.R_WEREWOLF]; len(teams) > 0 {
		return teams[0]
	}
	return ""
}

// スコアラが未設定なら従来どおりマッチ単位の重みだけで並ぶ。
func TestGetMatchesWithoutScorer(t *testing.T) {
	mo := newTestOptimizer()
	mo.ScheduledMatches[1].Weight = 2.0

	matches := mo.GetMatches()
	if len(matches) != 2 {
		t.Fatalf("マッチ数が想定と異なります: %v", len(matches))
	}
	if got := firstWerewolf(matches[0]); got != "gamma" {
		t.Fatalf("重みの大きいマッチが先頭に来ていません: %v", got)
	}
}

// 信頼度の低いチームを含むマッチは、マッチ単位の重みが同じでも後ろへ回る。
func TestGetMatchesOrdersByEffectiveWeight(t *testing.T) {
	mo := newTestOptimizer()
	mo.SetTeamScorer(stubScorer{weights: map[string]float64{"alpha": 0.1}})

	matches := mo.GetMatches()
	if got := firstWerewolf(matches[0]); got != "gamma" {
		t.Fatalf("不安定なチームを含むマッチが優先されました: %v", got)
	}
	if got := firstWerewolf(matches[1]); got != "alpha" {
		t.Fatalf("後ろへ回るはずのマッチが見当たりません: %v", got)
	}
}

// 隔離中のチームを含むマッチは候補から外れる。
func TestGetMatchesExcludesQuarantinedTeams(t *testing.T) {
	mo := newTestOptimizer()
	mo.SetTeamScorer(stubScorer{quarantined: map[string]bool{"beta": true}})

	matches := mo.GetMatches()
	if len(matches) != 1 {
		t.Fatalf("隔離チームを含むマッチが除外されていません: %v", len(matches))
	}
	if got := firstWerewolf(matches[0]); got != "gamma" {
		t.Fatalf("残るべきマッチが想定と異なります: %v", got)
	}
}

// 全マッチが隔離で消えるとゲームが二度と成立しないため、そのときだけ隔離を無視する。
func TestGetMatchesFallsBackWhenEverythingQuarantined(t *testing.T) {
	mo := newTestOptimizer()
	mo.SetTeamScorer(stubScorer{quarantined: map[string]bool{
		"alpha": true, "beta": true, "gamma": true, "delta": true,
	}})

	if matches := mo.GetMatches(); len(matches) != 2 {
		t.Fatalf("候補が全滅したときに退避していません: %v", len(matches))
	}
}

// PenalizeMatch は abort_weight_factor を掛ける。0 なら一度で最下位まで落ちる。
func TestPenalizeMatchMultipliesWeight(t *testing.T) {
	mo := newTestOptimizer()
	mo.abortWeightFactor = 0.5
	match := map[model.Role][]string{model.R_WEREWOLF: {"alpha"}, model.R_VILLAGER: {"beta"}}

	mo.PenalizeMatch(match)
	if got := mo.ScheduledMatches[0].Weight; got != 0.5 {
		t.Fatalf("重みが係数どおりに下がっていません: %v", got)
	}
	mo.PenalizeMatch(match)
	if got := mo.ScheduledMatches[0].Weight; got != 0.25 {
		t.Fatalf("失敗のたびに段階的に下がっていません: %v", got)
	}
}

// 重みの変更は次の GetMatches から効く。並べ替えが返却後に行われていた頃は1回遅れていた。
func TestWeightAffectsTheNextGetMatches(t *testing.T) {
	mo := newTestOptimizer()
	mo.abortWeightFactor = 0.0
	mo.PenalizeMatch(map[model.Role][]string{model.R_WEREWOLF: {"alpha"}, model.R_VILLAGER: {"beta"}})

	matches := mo.GetMatches()
	if got := firstWerewolf(matches[0]); got != "gamma" {
		t.Fatalf("重みの更新が次の呼び出しに反映されていません: %v", got)
	}
}

// RoleIdxs が同一のマッチは Equal で区別できないため、どちらが消化されるかは格納順で
// 決まる。重みを下げた方が残り、消化されるのは重みの高い方であること。
// マッチオプティマイザは同じ組み合わせを複数スケジュールすることが実際にある。
func TestDuplicateMatchKeepsThePenalizedEntry(t *testing.T) {
	mo := newTestOptimizer()
	mo.abortWeightFactor = 0.0
	// alpha 対 beta を2件に増やし、同じ組み合わせが重複した状態にする。
	mo.ScheduledMatches = append(mo.ScheduledMatches, model.MatchWeight{
		RoleIdxs: map[model.Role][]int{model.R_WEREWOLF: {0}, model.R_VILLAGER: {1}},
		Weight:   1.0,
	})
	match := map[model.Role][]string{model.R_WEREWOLF: {"alpha"}, model.R_VILLAGER: {"beta"}}

	// 1件目のゲームが異常終了し、重みが下がる。
	mo.PenalizeMatch(match)
	// 次のマッチング時に格納順が重みの降順へ揃う。
	mo.GetMatches()
	// 2件目のゲームが正常終了する。
	mo.SetMatchEnd(match)

	var remaining []float64
	for _, sm := range mo.ScheduledMatches {
		if sm.Equal(model.MatchWeight{RoleIdxs: map[model.Role][]int{model.R_WEREWOLF: {0}, model.R_VILLAGER: {1}}}) {
			remaining = append(remaining, sm.Weight)
		}
	}
	if len(remaining) != 1 {
		t.Fatalf("重複マッチが1件だけ消化されていません: %v", remaining)
	}
	if remaining[0] != 0.0 {
		t.Fatalf("重みを下げた方が消化され、ペナルティが失われました: %v", remaining[0])
	}
}

// infinite_loop で全マッチを消化したときに自己デッドロックしないこと。
func TestGetMatchesAppendsWhenExhausted(t *testing.T) {
	mo := newTestOptimizer()
	mo.InfiniteLoop = true
	mo.RoleNumMap = map[model.Role]int{model.R_WEREWOLF: 1, model.R_VILLAGER: 1}
	for i := range mo.ScheduledMatches {
		mo.ScheduledMatches[i].Weight = 0
	}

	done := make(chan int, 1)
	go func() { done <- len(mo.GetMatches()) }()

	select {
	case got := <-done:
		if got <= 2 {
			t.Fatalf("マッチが補充されていません: %v", got)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("GetMatches が返りませんでした（自己デッドロックの可能性）")
	}
}
