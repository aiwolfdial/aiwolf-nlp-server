package model

import (
	"slices"
	"time"
)

// RulesetView is a read-only projection of the game rules that the engine
// consumes. The engine holds the game configuration through this interface, so
// no code — not even within the logic package — can mutate or reassign the
// rules of an in-flight game through it.
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

// frozenRuleset captures, at construction time, exactly the config fields the
// engine reads. It is a value type with no setters; slice getters return copies
// so callers cannot reach the captured data.
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

// NewRulesetView freezes the rules the engine needs from config into a
// read-only view. Each game gets its own frozen copy, so later edits to the
// source config cannot affect a running game.
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
func (r frozenRuleset) DayPhases() []Phase             { return slices.Clone(r.dayPhases) }
func (r frozenRuleset) NightPhases() []Phase           { return slices.Clone(r.nightPhases) }
func (r frozenRuleset) ActionTimeout() time.Duration   { return r.actionTimeout }
func (r frozenRuleset) ResponseTimeout() time.Duration { return r.responseTimeout }
func (r frozenRuleset) AcceptableTimeout() time.Duration {
	return r.acceptableTimeout
}
