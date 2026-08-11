package model

import (
	"encoding/json"
	"errors"

	"github.com/aiwolfdial/aiwolf-nlp-server/model/wire"
)

// トークと囁きは同じ設定項目を持つため、Go側では同じ型を使う。
type TalkSetting = wire.SettingTalk

// エージェントへ送るゲーム設定。JSONの形は wire.Setting が唯一の定義。
type Setting struct {
	AgentCount     int
	MaxDay         *int
	RoleNumMap     map[Role]int
	VoteVisibility bool
	Talk           TalkSetting
	Whisper        TalkSetting
	Vote           wire.SettingVote
	AttackVote     wire.SettingAttackVote
	Timeout        wire.SettingTimeout
}

func NewSetting(config Config) (*Setting, error) {
	roles, err := RolesFromConfig(config)
	if err != nil {
		return nil, err
	}
	if config.CustomProfile.Enable {
		if config.CustomProfile.DynamicProfile.Enable {
			if len(config.CustomProfile.DynamicProfile.Avatars) < config.Game.AgentCount {
				return nil, errors.New("カスタムプロフィールのアバターがエージェント数より少ないです")
			}
		} else {
			if len(config.CustomProfile.Profiles) < config.Game.AgentCount {
				return nil, errors.New("カスタムプロフィールの人数がエージェント数より少ないです")
			}
		}
	}
	if config.Game.Talk.MaxLength.CountInWord && config.Game.Talk.MaxLength.CountSpaces {
		return nil, errors.New("TalkのCountInWordとCountSpacesを両方有効にすることはできません")
	}
	if config.Game.Whisper.MaxLength.CountInWord && config.Game.Whisper.MaxLength.CountSpaces {
		return nil, errors.New("WhisperのCountInWordとCountSpacesを両方有効にすることはできません")
	}

	setting := Setting{
		AgentCount:     config.Game.AgentCount,
		RoleNumMap:     roles,
		VoteVisibility: config.Game.VoteVisibility,
		Talk:           newTalkSetting(config.Game.Talk),
		Whisper:        newTalkSetting(config.Game.Whisper),
		Vote: wire.SettingVote{
			MaxCount:      config.Game.Vote.MaxCount,
			AllowSelfVote: config.Game.Vote.AllowSelfVote,
		},
		AttackVote: wire.SettingAttackVote{
			MaxCount:      config.Game.AttackVote.MaxCount,
			AllowSelfVote: config.Game.AttackVote.AllowSelfVote,
			AllowNoTarget: config.Game.AttackVote.AllowNoTarget,
		},
		Timeout: wire.SettingTimeout{
			Action:   int(config.Server.Timeout.Action.Milliseconds()),
			Response: int(config.Server.Timeout.Response.Milliseconds()),
		},
	}
	if config.Game.MaxDay != -1 {
		setting.MaxDay = &config.Game.MaxDay
	}
	return &setting, nil
}

// -1は「制限なし」を表すため、その項目自体を送らない。
func newTalkSetting(config TalkConfig) TalkSetting {
	setting := TalkSetting{
		MaxCount: wire.SettingTalkMaxCount{
			PerAgent: config.MaxCount.PerAgent,
			PerDay:   config.MaxCount.PerDay,
		},
		MaxSkip: config.MaxSkip,
	}
	if config.Duration != nil {
		duration := int(config.Duration.Milliseconds())
		setting.Duration = &duration
	}
	countInWord := config.MaxLength.CountInWord
	countSpaces := config.MaxLength.CountSpaces
	mentionLength := config.MaxLength.MentionLength
	if config.MaxLength.PerTalk != -1 {
		setting.MaxLength.CountInWord = &countInWord
		setting.MaxLength.CountSpaces = &countSpaces
		setting.MaxLength.PerTalk = &config.MaxLength.PerTalk
	}
	if config.MaxLength.PerAgent != -1 {
		setting.MaxLength.CountInWord = &countInWord
		setting.MaxLength.CountSpaces = &countSpaces
		setting.MaxLength.PerAgent = &config.MaxLength.PerAgent
		setting.MaxLength.MentionLength = &mentionLength
	}
	if config.MaxLength.BaseLength != -1 {
		setting.MaxLength.CountInWord = &countInWord
		setting.MaxLength.CountSpaces = &countSpaces
		setting.MaxLength.BaseLength = &config.MaxLength.BaseLength
		setting.MaxLength.MentionLength = &mentionLength
	}
	return setting
}

func (s Setting) wire() wire.Setting {
	roleNumMap := make(map[string]int, len(s.RoleNumMap))
	for role, num := range s.RoleNumMap {
		roleNumMap[role.String()] = num
	}
	return wire.Setting{
		AgentCount:     s.AgentCount,
		MaxDay:         s.MaxDay,
		RoleNumMap:     roleNumMap,
		VoteVisibility: s.VoteVisibility,
		Talk:           s.Talk,
		Whisper:        s.Whisper,
		Vote:           s.Vote,
		AttackVote:     s.AttackVote,
		Timeout:        s.Timeout,
	}
}

func (s Setting) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.wire())
}
