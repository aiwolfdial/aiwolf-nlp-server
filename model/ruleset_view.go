package model

import (
	"slices"
	"time"
)

// Gameはこのインターフェース型で設定を保持する。getterのみのため、logic内ですら
// ルールの書き換え・再代入ができない。
type RulesetView interface {
	MaxDay() int
	MaxContinueErrorRatio() float64
	VoteAllowSelfVote() bool
	AttackVoteAllowSelfVote() bool
	DayPhases() []Phase
	NightPhases() []Phase
	ActionTimeout() time.Duration
	ResponseTimeout() time.Duration
	AcceptableTimeout() time.Duration
}

// 構築時にエンジンが読む設定値だけを取り込む。各ゲームが自前のコピーを持つため、
// 後から元configを編集しても進行中のゲームに影響しない。
type frozenRuleset struct {
	maxDay                  int
	maxContinueErrorRatio   float64
	voteAllowSelfVote       bool
	attackVoteAllowSelfVote bool
	dayPhases               []Phase
	nightPhases             []Phase
	actionTimeout           time.Duration
	responseTimeout         time.Duration
	acceptableTimeout       time.Duration
}

func NewRulesetView(config Config) RulesetView {
	return frozenRuleset{
		maxDay:                  config.Game.MaxDay,
		maxContinueErrorRatio:   config.Server.MaxContinueErrorRatio,
		voteAllowSelfVote:       config.Game.Vote.AllowSelfVote,
		attackVoteAllowSelfVote: config.Game.AttackVote.AllowSelfVote,
		dayPhases:               slices.Clone(config.Logic.DayPhases),
		nightPhases:             slices.Clone(config.Logic.NightPhases),
		actionTimeout:           config.Server.Timeout.Action,
		responseTimeout:         config.Server.Timeout.Response,
		acceptableTimeout:       config.Server.Timeout.Acceptable,
	}
}

func (r frozenRuleset) MaxDay() int                    { return r.maxDay }
func (r frozenRuleset) MaxContinueErrorRatio() float64 { return r.maxContinueErrorRatio }
func (r frozenRuleset) VoteAllowSelfVote() bool        { return r.voteAllowSelfVote }
func (r frozenRuleset) AttackVoteAllowSelfVote() bool  { return r.attackVoteAllowSelfVote }
func (r frozenRuleset) ActionTimeout() time.Duration   { return r.actionTimeout }
func (r frozenRuleset) ResponseTimeout() time.Duration { return r.responseTimeout }
func (r frozenRuleset) AcceptableTimeout() time.Duration {
	return r.acceptableTimeout
}

// 内部スライスを渡さないよう複製を返す。
func (r frozenRuleset) DayPhases() []Phase   { return slices.Clone(r.dayPhases) }
func (r frozenRuleset) NightPhases() []Phase { return slices.Clone(r.nightPhases) }
