package model

import (
	"encoding/json"
	"errors"
)

type Setting struct {
	AgentCount     int          `json:"agent_count"`
	RoleNumMap     map[Role]int `json:"role_num_map"`
	VoteVisibility bool         `json:"vote_visibility"`
	TalkOnFirstDay bool         `json:"talk_on_first_day"`
	Talk           struct {
		TalkSetting `json:",inline"`
	} `json:"talk"`
	Whisper struct {
		TalkSetting `json:",inline"`
	} `json:"whisper"`
	Vote struct {
		MaxCount int `json:"max_count"`
	} `json:"vote"`
	AttackVote struct {
		MaxCount      int  `json:"max_count"`
		AllowNoTarget bool `json:"allow_no_target"`
	} `json:"attack_vote"`
	Timeout struct {
		Action   int `json:"action"`
		Response int `json:"response"`
	} `json:"timeout"`
}

type TalkSetting struct {
	MaxCount struct {
		PerAgent int `json:"per_agent"`
		PerDay   int `json:"per_day"`
	} `json:"max_count"`
	MaxLength struct {
		CountInWord   *bool `json:"count_in_word,omitempty"`
		PerTalk       *int  `json:"per_talk,omitempty"`
		MentionLength *int  `json:"mention_length,omitempty"`
		PerAgent      *int  `json:"per_agent,omitempty"`
		BaseLength    *int  `json:"base_length,omitempty"`
	} `json:"max_length"`
	MaxSkip int `json:"max_skip"`
}

func NewSetting(config Config) (*Setting, error) {
	roleNumMap := Roles(config.Game.AgentCount)
	if roleNumMap == nil {
		return nil, errors.New("対応する役職の人数がありません")
	}
	if config.Game.CustomProfile.Enable && !config.Game.CustomProfile.DynamicProfile.Enable {
		if len(config.Game.CustomProfile.Profiles) < config.Game.AgentCount {
			return nil, errors.New("カスタムプロフィールの人数がエージェント数より少ないです")
		}
	}
	setting := Setting{
		AgentCount:     config.Game.AgentCount,
		RoleNumMap:     roleNumMap,
		VoteVisibility: config.Game.VoteVisibility,
		TalkOnFirstDay: config.Game.TalkOnFirstDay,
		Talk: struct {
			TalkSetting `json:",inline"`
		}{
			TalkSetting: TalkSetting{
				MaxCount: struct {
					PerAgent int `json:"per_agent"`
					PerDay   int `json:"per_day"`
				}{
					PerAgent: config.Game.Talk.MaxCount.PerAgent,
					PerDay:   config.Game.Talk.MaxCount.PerDay,
				},
				MaxLength: struct {
					CountInWord   *bool `json:"count_in_word,omitempty"`
					PerTalk       *int  `json:"per_talk,omitempty"`
					MentionLength *int  `json:"mention_length,omitempty"`
					PerAgent      *int  `json:"per_agent,omitempty"`
					BaseLength    *int  `json:"base_length,omitempty"`
				}{},
				MaxSkip: config.Game.Talk.MaxSkip,
			},
		},
		Whisper: struct {
			TalkSetting `json:",inline"`
		}{
			TalkSetting: TalkSetting{
				MaxCount: struct {
					PerAgent int `json:"per_agent"`
					PerDay   int `json:"per_day"`
				}{
					PerAgent: config.Game.Whisper.MaxCount.PerAgent,
					PerDay:   config.Game.Whisper.MaxCount.PerDay,
				},
				MaxLength: struct {
					CountInWord   *bool `json:"count_in_word,omitempty"`
					PerTalk       *int  `json:"per_talk,omitempty"`
					MentionLength *int  `json:"mention_length,omitempty"`
					PerAgent      *int  `json:"per_agent,omitempty"`
					BaseLength    *int  `json:"base_length,omitempty"`
				}{},
				MaxSkip: config.Game.Whisper.MaxSkip,
			},
		},
		Vote: struct {
			MaxCount int `json:"max_count"`
		}{
			MaxCount: config.Game.Vote.MaxCount,
		},
		AttackVote: struct {
			MaxCount      int  `json:"max_count"`
			AllowNoTarget bool `json:"allow_no_target"`
		}{
			MaxCount:      config.Game.AttackVote.MaxCount,
			AllowNoTarget: config.Game.AttackVote.AllowNoTarget,
		},
		Timeout: struct {
			Action   int `json:"action"`
			Response int `json:"response"`
		}{
			Action:   int(config.Game.Timeout.Action.Milliseconds()),
			Response: int(config.Game.Timeout.Response.Milliseconds()),
		},
	}
	if config.Game.Talk.MaxLength.PerTalk != -1 {
		setting.Talk.MaxLength.CountInWord = &config.Game.Talk.MaxLength.CountInWord
		setting.Talk.MaxLength.PerTalk = &config.Game.Talk.MaxLength.PerTalk
	}
	if config.Game.Talk.MaxLength.PerAgent != -1 {
		setting.Talk.MaxLength.CountInWord = &config.Game.Talk.MaxLength.CountInWord
		setting.Talk.MaxLength.PerAgent = &config.Game.Talk.MaxLength.PerAgent
		setting.Talk.MaxLength.MentionLength = &config.Game.Talk.MaxLength.MentionLength
	}
	if config.Game.Talk.MaxLength.BaseLength != -1 {
		setting.Talk.MaxLength.CountInWord = &config.Game.Talk.MaxLength.CountInWord
		setting.Talk.MaxLength.BaseLength = &config.Game.Talk.MaxLength.BaseLength
		setting.Talk.MaxLength.MentionLength = &config.Game.Talk.MaxLength.MentionLength
	}
	if config.Game.Whisper.MaxLength.PerTalk != -1 {
		setting.Whisper.MaxLength.CountInWord = &config.Game.Whisper.MaxLength.CountInWord
		setting.Whisper.MaxLength.PerTalk = &config.Game.Whisper.MaxLength.PerTalk
	}
	if config.Game.Whisper.MaxLength.PerAgent != -1 {
		setting.Whisper.MaxLength.CountInWord = &config.Game.Whisper.MaxLength.CountInWord
		setting.Whisper.MaxLength.PerAgent = &config.Game.Whisper.MaxLength.PerAgent
		setting.Whisper.MaxLength.MentionLength = &config.Game.Whisper.MaxLength.MentionLength
	}
	if config.Game.Whisper.MaxLength.BaseLength != -1 {
		setting.Whisper.MaxLength.CountInWord = &config.Game.Whisper.MaxLength.CountInWord
		setting.Whisper.MaxLength.BaseLength = &config.Game.Whisper.MaxLength.BaseLength
		setting.Whisper.MaxLength.MentionLength = &config.Game.Whisper.MaxLength.MentionLength
	}
	return &setting, nil
}

func (s Setting) MarshalJSON() ([]byte, error) {
	roleNumMap := make(map[string]int)
	for k, v := range s.RoleNumMap {
		roleNumMap[k.String()] = v
	}
	type Alias Setting
	return json.Marshal(&struct {
		*Alias
		RoleNumMap map[string]int `json:"role_num_map"`
	}{
		Alias:      (*Alias)(&s),
		RoleNumMap: roleNumMap,
	})
}
