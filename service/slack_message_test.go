package service

import (
	"strings"
	"testing"
)

// バーの長さは常に一定でないと、通知が並んだときに桁がずれて読みにくくなる。
func TestProgressBarWidthIsStable(t *testing.T) {
	for _, ratio := range []float64{0, 0.001, 0.25, 0.536, 0.999, 1} {
		bar := strings.Trim(progressBar(ratio, 20), "`")
		if got := len([]rune(bar)); got != 20 {
			t.Fatalf("ratio=%v のバー長が %d です", ratio, got)
		}
	}
}

func TestProgressBarFill(t *testing.T) {
	cases := []struct {
		ratio  float64
		filled int
	}{
		{0, 0},
		{0.5, 10},
		{1, 20},
		// 範囲外の値でも潰れないこと。進捗が game_count を超えることが実際にある。
		{-1, 0},
		{2, 20},
	}
	for _, c := range cases {
		bar := strings.Trim(progressBar(c.ratio, 20), "`")
		if got := strings.Count(bar, "█"); got != c.filled {
			t.Fatalf("ratio=%v の塗りつぶしが %d です (期待 %d)", c.ratio, got, c.filled)
		}
	}
}

func TestRatioOfHandlesZeroTotal(t *testing.T) {
	if got := ratioOf(3, 0); got != 0 {
		t.Fatalf("総数0のとき比率が %v になりました", got)
	}
}
