package service

import (
	"fmt"
	"math"
	"strings"
)

// Incoming Webhook のペイロード。attachments の color がメッセージ左端の色帯になり、
// 一覧の中でも重要度を色で判別できる。
type slackPayload struct {
	Username    string            `json:"username,omitempty"`
	IconEmoji   string            `json:"icon_emoji,omitempty"`
	Text        string            `json:"text"`
	Attachments []slackAttachment `json:"attachments,omitempty"`
}

type slackAttachment struct {
	Color  string       `json:"color,omitempty"`
	Blocks []slackBlock `json:"blocks,omitempty"`
}

type slackBlock struct {
	Type     string      `json:"type"`
	Text     *slackText  `json:"text,omitempty"`
	Fields   []slackText `json:"fields,omitempty"`
	Elements []slackText `json:"elements,omitempty"`
}

type slackText struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// Slack のブランドカラー。緑=正常、黄=注意、赤=異常。
const (
	colorGood    = "#2eb67d"
	colorWarning = "#ecb22e"
	colorDanger  = "#e01e5a"
	colorInfo    = "#36c5f0"
)

// 1メッセージあたりのブロック数上限は50。隔離が同時多発しても超えないよう余裕を持って切る。
const maxQuarantineBlocks = 12

func mrkdwn(format string) slackText {
	return slackText{Type: "mrkdwn", Text: format}
}

func sectionBlock(text string) slackBlock {
	t := mrkdwn(text)
	return slackBlock{Type: "section", Text: &t}
}

func fieldsBlock(fields ...string) slackBlock {
	texts := make([]slackText, 0, len(fields))
	for _, f := range fields {
		texts = append(texts, mrkdwn(f))
	}
	return slackBlock{Type: "section", Fields: texts}
}

func contextBlock(text string) slackBlock {
	return slackBlock{Type: "context", Elements: []slackText{mrkdwn(text)}}
}

// 進捗を塗りつぶしブロックで描く。Slack は本文がプロポーショナルフォントなので、
// 桁が揃うようインラインコードで囲んで等幅で表示させる。
func progressBar(ratio float64, cells int) string {
	if math.IsNaN(ratio) || ratio < 0 {
		ratio = 0
	}
	if ratio > 1 {
		ratio = 1
	}
	filled := int(math.Round(float64(cells) * ratio))
	return "`" + strings.Repeat("█", filled) + strings.Repeat("░", cells-filled) + "`"
}

// 1フィールドあたりの上限は2000文字。チーム数が多くても本文が壊れないよう頭から切る。
const maxTeamsInField = 20

// Slack のメンション記法・リンク記法として解釈される文字を潰す。
var slackEscaper = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")

// チーム名はクライアントが名乗った文字列そのままで、サーバは検証していない。
// `<!channel>` を名乗られると通知のたびにチャンネル全員が呼ばれてしまうため、
// 外部由来の文字列は本文へ入れる直前に必ずここを通す。
func escapeSlack(s string) string {
	return slackEscaper.Replace(s)
}

func escapeSlackAll(values []string) []string {
	escaped := make([]string, 0, len(values))
	for _, v := range values {
		escaped = append(escaped, escapeSlack(v))
	}
	return escaped
}

// チーム名を1行に並べる。空のときに欄が消えると「0件」なのか「取れていない」のか
// 区別できないため、必ず何か書く。
func teamList(teams []string) string {
	if len(teams) == 0 {
		return "_なし_"
	}
	escaped := escapeSlackAll(teams)
	if len(escaped) <= maxTeamsInField {
		return strings.Join(escaped, ", ")
	}
	return strings.Join(escaped[:maxTeamsInField], ", ") +
		fmt.Sprintf(" ほか %d チーム", len(escaped)-maxTeamsInField)
}

func ratioOf(done, total int) float64 {
	if total <= 0 {
		return 0
	}
	return float64(done) / float64(total)
}
