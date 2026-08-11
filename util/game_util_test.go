package util

import "testing"

// 切り捨てが0になる設定でも、脱落0人でゲームが打ち切られないこと。
// 5人村の 0.2 未満、13人村の 0.077 未満が該当し、放置すると全ゲームが即座に
// 異常終了して全チームが隔離される。
func TestCalcErrorAbortThresholdIsAtLeastOne(t *testing.T) {
	cases := []struct {
		agents int
		ratio  float64
	}{
		{5, 0.0}, {5, 0.1}, {5, 0.19},
		{9, 0.1}, {13, 0.05}, {13, 0.0},
	}
	for _, c := range cases {
		if got := CalcErrorAbortThreshold(c.agents, c.ratio); got < 1 {
			t.Fatalf("agents=%d ratio=%v の閾値が %d です", c.agents, c.ratio, got)
		}
	}
}

// 実運用の設定では従来と同じ閾値になること。既存の大会結果を変えないため。
func TestCalcErrorAbortThresholdKeepsExistingValues(t *testing.T) {
	cases := []struct {
		agents int
		ratio  float64
		want   int
	}{
		{5, 0.2, 1},
		{9, 0.2, 1},
		{13, 0.2, 2},
		{13, 0.3, 3},
		{5, 1.0, 5},
	}
	for _, c := range cases {
		if got := CalcErrorAbortThreshold(c.agents, c.ratio); got != c.want {
			t.Fatalf("agents=%d ratio=%v の閾値が %d です (期待 %d)", c.agents, c.ratio, got, c.want)
		}
	}
}
