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

// 空欄を消してしまうと「0件」なのか「取得できていない」のか読み手が区別できない。
func TestTeamListIsNeverEmpty(t *testing.T) {
	if got := teamList(nil); got == "" {
		t.Fatal("チームが0件のときに空文字になりました")
	}
}

func TestTeamListTruncatesLongLists(t *testing.T) {
	teams := make([]string, maxTeamsInField+5)
	for i := range teams {
		teams[i] = "team"
	}
	got := teamList(teams)
	if !strings.Contains(got, "ほか 5 チーム") {
		t.Fatalf("打ち切りの表示がありません: %s", got)
	}
}

// チーム名はクライアントが名乗った文字列なので、Slack のメンション記法を通してはいけない。
// `<!channel>` を名乗られると通知のたびにチャンネル全員が呼ばれる。
func TestEscapeSlackNeutralizesMentions(t *testing.T) {
	for _, raw := range []string{"<!channel>", "<!here>", "<http://evil|click>", "a&b"} {
		got := escapeSlack(raw)
		if strings.ContainsAny(got, "<>") {
			t.Fatalf("%q がエスケープされていません: %q", raw, got)
		}
	}
	if got := escapeSlack("a&b"); got != "a&amp;b" {
		t.Fatalf("アンパサンドのエスケープが想定と異なります: %q", got)
	}
}

// 一覧に並べる経路でもエスケープが漏れないこと。
func TestTeamListEscapes(t *testing.T) {
	got := teamList([]string{"<!channel>", "normal"})
	if strings.ContainsAny(got, "<>") {
		t.Fatalf("一覧でエスケープが漏れています: %q", got)
	}
}

func TestRatioOfHandlesZeroTotal(t *testing.T) {
	if got := ratioOf(3, 0); got != 0 {
		t.Fatalf("総数0のとき比率が %v になりました", got)
	}
}
