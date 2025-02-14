package model

import (
	"encoding/json"
	"errors"
)

type Setting struct {
	PlayerNum        int          `json:"playerNum"`
	RoleNumMap       map[Role]int `json:"roleNumMap"`
	MaxTalk          int          `json:"maxTalk"`
	MaxTalkTurn      int          `json:"maxTalkTurn"`
	MaxWhisper       int          `json:"maxWhisper"`
	MaxWhisperTurn   int          `json:"maxWhisperTurn"`
	MaxSkip          int          `json:"maxSkip"`
	IsEnableNoAttack bool         `json:"isEnableNoAttack"`
	IsVoteVisible    bool         `json:"isVoteVisible"`
	IsTalkOnFirstDay bool         `json:"isTalkOnFirstDay"`
	ResponseTimeout  int          `json:"responseTimeout"`
	ActionTimeout    int          `json:"actionTimeout"`
	MaxRevote        int          `json:"maxRevote"`
	MaxAttackRevote  int          `json:"maxAttackRevote"`
}

func NewSetting(config Config) (*Setting, error) {
	roleNumMap := Roles(config.Game.AgentCount)
	if roleNumMap == nil {
		return nil, errors.New("対応する役職の人数がありません")
	}
	return &Setting{
		PlayerNum:        config.Game.AgentCount,
		RoleNumMap:       roleNumMap,
		MaxTalk:          config.Game.Talk.MaxCount.PerAgent,
		MaxTalkTurn:      config.Game.Talk.MaxCount.PerDay,
		MaxWhisper:       config.Game.Whisper.MaxCount.PerAgent,
		MaxWhisperTurn:   config.Game.Whisper.MaxCount.PerDay,
		MaxSkip:          config.Game.Skip.MaxCount,
		IsEnableNoAttack: config.Game.Attack.AllowNoTarget,
		IsVoteVisible:    config.Game.VoteVisibility,
		IsTalkOnFirstDay: config.Game.TalkOnFirstDay,
		ResponseTimeout:  int(config.Game.Timeout.Response.Milliseconds()),
		ActionTimeout:    int(config.Game.Timeout.Action.Milliseconds()),
		MaxRevote:        config.Game.Vote.MaxCount,
		MaxAttackRevote:  config.Game.Attack.MaxCount,
	}, nil
}

func (s Setting) MarshalJSON() ([]byte, error) {
	roleNumMap := make(map[string]int)
	for k, v := range s.RoleNumMap {
		roleNumMap[k.String()] = v
	}
	type Alias Setting
	return json.Marshal(&struct {
		*Alias
		RoleNumMap map[string]int `json:"roleNumMap"`
	}{
		Alias:      (*Alias)(&s),
		RoleNumMap: roleNumMap,
	})
}
