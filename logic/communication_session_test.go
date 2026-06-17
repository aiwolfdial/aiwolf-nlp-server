package logic

import (
	"testing"

	"github.com/aiwolfdial/aiwolf-nlp-server/model"
	"github.com/stretchr/testify/assert"
)

func ptr[T any](v T) *T { return &v }

func newTestSession(ts *model.TalkSetting) (*CommunicationSession, *model.Agent) {
	agent := &model.Agent{GameName: "Agent[01]"}
	return &CommunicationSession{
		game:            &Game{id: "test", agents: []*model.Agent{agent}},
		talkSetting:     ts,
		remainLengthMap: map[model.Agent]int{},
	}, agent
}

// per_agent と base_length が両方無効（-1 → nil）かつ per_talk も無効の場合、
// 文字数制限が一切ないので本文はそのまま通る必要がある。
// 修正前は本文が空文字に組み立て直されて OVER に置換されていた。
func TestProcessTextBothLengthLimitsDisabled(t *testing.T) {
	ts := &model.TalkSetting{}
	s, agent := newTestSession(ts)

	got := s.processText(agent, "Hello World!")
	assert.Equal(t, "Hello World!", got)
}

// per_agent と base_length は無効だが per_talk のみ有効な場合、
// 本文が消えずに per_talk の範囲で切り詰められる必要がある。
func TestProcessTextBothLengthLimitsDisabledWithPerTalk(t *testing.T) {
	ts := &model.TalkSetting{}
	ts.MaxLength.CountInWord = ptr(false)
	ts.MaxLength.CountSpaces = ptr(true)
	ts.MaxLength.PerTalk = ptr(5)
	s, agent := newTestSession(ts)

	got := s.processText(agent, "abcdefgh")
	assert.Equal(t, "abcde", got)
}

// base_length が有効な場合の既存挙動が変わっていないことを確認する（回帰防止）。
// この経路では commonText が必ず TrimLength で上書きされるため、
// 初期値変更による副作用がないことを保証する。
func TestProcessTextBaseLengthUnaffectedByFix(t *testing.T) {
	ts := &model.TalkSetting{}
	ts.MaxLength.CountInWord = ptr(false)
	ts.MaxLength.CountSpaces = ptr(true)
	ts.MaxLength.BaseLength = ptr(5)
	s, agent := newTestSession(ts)

	got := s.processText(agent, "abcdefgh")
	assert.Equal(t, "abcde", got)
}
