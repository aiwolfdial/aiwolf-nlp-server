package logic

import (
	"fmt"
	"log/slog"
	"strings"
	"unicode/utf8"

	"github.com/aiwolfdial/aiwolf-nlp-server/model"
	"github.com/aiwolfdial/aiwolf-nlp-server/util"
)

type CommunicationSession struct {
	game            *Game
	request         model.Request
	agents          []*model.Agent
	talkSetting     *model.TalkSetting
	talkList        *[]model.Talk
	remainCountMap  map[model.Agent]int
	remainLengthMap map[model.Agent]int
	remainSkipMap   map[model.Agent]int
	idx             int
}

func newCommunicationSession(game *Game, request model.Request, agents []*model.Agent) *CommunicationSession {
	var talkSetting *model.TalkSetting
	var talkList *[]model.Talk

	switch request {
	case model.R_TALK:
		talkSetting = &game.setting.Talk.TalkSetting
		talkList = &game.getCurrentGameStatus().Talks
	case model.R_WHISPER:
		talkSetting = &game.setting.Whisper.TalkSetting
		talkList = &game.getCurrentGameStatus().Whispers
	default:
		return nil
	}

	remainCountMap := make(map[model.Agent]int)
	remainLengthMap := make(map[model.Agent]int)
	remainSkipMap := make(map[model.Agent]int)
	for _, agent := range agents {
		remainCountMap[*agent] = talkSetting.MaxCount.PerAgent
		if talkSetting.MaxLength.PerAgent != nil {
			remainLengthMap[*agent] = *talkSetting.MaxLength.PerAgent
		}
		remainSkipMap[*agent] = talkSetting.MaxSkip
	}

	gs := game.getCurrentGameStatus()
	gs.RemainCountMap = &remainCountMap
	gs.RemainLengthMap = &remainLengthMap
	gs.RemainSkipMap = &remainSkipMap

	return &CommunicationSession{
		game:            game,
		request:         request,
		agents:          agents,
		talkSetting:     talkSetting,
		talkList:        talkList,
		remainCountMap:  remainCountMap,
		remainLengthMap: remainLengthMap,
		remainSkipMap:   remainSkipMap,
		idx:             len(*talkList),
	}
}

func (s *CommunicationSession) cleanup() {
	gs := s.game.getCurrentGameStatus()
	gs.RemainCountMap = nil
	gs.RemainLengthMap = nil
	gs.RemainSkipMap = nil
}

func (s *CommunicationSession) allAgentsDone() bool {
	for _, agent := range s.agents {
		if s.remainCountMap[*agent] > 0 {
			return false
		}
	}
	return true
}

func (s *CommunicationSession) canAgentTalk(agent *model.Agent) bool {
	if s.remainCountMap[*agent] <= 0 {
		return false
	}
	if value, exists := s.remainLengthMap[*agent]; exists {
		if value <= 0 {
			return false
		}
	}
	return true
}

func (s *CommunicationSession) processSkipOver(agent *model.Agent, text string) string {
	switch text {
	case model.T_SKIP:
		if s.remainSkipMap[*agent] <= 0 {
			text = model.T_OVER
			slog.Warn("スキップ回数が上限に達したため、発言をオーバーに置換しました", "id", s.game.id, "agent", agent.String())
		} else {
			s.remainSkipMap[*agent]--
			slog.Info("発言をスキップしました", "id", s.game.id, "agent", agent.String())
		}
	case model.T_FORCE_SKIP:
		text = model.T_SKIP
		slog.Warn("強制スキップが指定されたため、発言をスキップに置換しました", "id", s.game.id, "agent", agent.String())
	}

	if text != model.T_OVER && text != model.T_SKIP && text != model.T_FORCE_SKIP {
		s.remainSkipMap[*agent] = s.talkSetting.MaxSkip
		slog.Info("発言がオーバーもしくはスキップではないため、スキップ回数をリセットしました", "id", s.game.id, "agent", agent.String())
	}

	if text == model.T_OVER {
		s.remainCountMap[*agent] = 0
		slog.Info("発言がオーバーであるため、残り発言回数を0にしました", "id", s.game.id, "agent", agent.String())
	}

	return text
}

func (s *CommunicationSession) processText(agent *model.Agent, text string) string {
	if text == model.T_OVER || text == model.T_SKIP || text == model.T_FORCE_SKIP {
		return text
	}

	mention := ""
	commonText := ""
	mentionText := ""

	if s.talkSetting.MaxLength.PerAgent != nil || s.talkSetting.MaxLength.BaseLength != nil {
		baseLength := 0
		if s.talkSetting.MaxLength.BaseLength != nil {
			baseLength = *s.talkSetting.MaxLength.BaseLength
		}

		mentionIdx := -1
		if s.talkSetting.MaxLength.MentionLength != nil {
			for _, a := range s.game.agents {
				if a != agent {
					if strings.Contains(text, "@"+a.String()) {
						if mentionIdx == -1 {
							mention = "@" + a.String()
							mentionIdx = strings.Index(text, mention)
						}
						if strings.Index(text, "@"+a.String()) < mentionIdx {
							mention = "@" + a.String()
							mentionIdx = strings.Index(text, mention)
						}
					}
				}
			}
		}

		if mentionIdx != -1 {
			remainLength := baseLength
			if value, exists := s.remainLengthMap[*agent]; exists {
				remainLength += value
			}
			mentionBefore := text[:mentionIdx]
			mentionAfter := text[mentionIdx+len(mention):]

			mention = " " + mention + " "

			commonText = util.TrimLength(mentionBefore, remainLength, *s.talkSetting.MaxLength.CountInWord, *s.talkSetting.MaxLength.CountSpaces)
			cost := util.CountLength(mentionBefore, *s.talkSetting.MaxLength.CountInWord, *s.talkSetting.MaxLength.CountSpaces) - baseLength
			if cost > 0 {
				if _, exists := s.remainLengthMap[*agent]; exists {
					s.remainLengthMap[*agent] -= cost
				}
			}

			remainLength = *s.talkSetting.MaxLength.MentionLength
			if value, exists := s.remainLengthMap[*agent]; exists {
				remainLength += value
			}
			mentionText = util.TrimLength(mentionAfter, remainLength, *s.talkSetting.MaxLength.CountInWord, *s.talkSetting.MaxLength.CountSpaces)
			mentionCost := util.CountLength(mentionText, *s.talkSetting.MaxLength.CountInWord, *s.talkSetting.MaxLength.CountSpaces) - *s.talkSetting.MaxLength.MentionLength
			if mentionCost > 0 {
				if _, exists := s.remainLengthMap[*agent]; exists {
					s.remainLengthMap[*agent] -= mentionCost
				}
			}
		} else {
			remainLength := baseLength
			if value, exists := s.remainLengthMap[*agent]; exists {
				remainLength += value
			}
			commonText = util.TrimLength(text, remainLength, *s.talkSetting.MaxLength.CountInWord, *s.talkSetting.MaxLength.CountSpaces)
			cost := util.CountLength(text, *s.talkSetting.MaxLength.CountInWord, *s.talkSetting.MaxLength.CountSpaces) - baseLength
			if cost > 0 {
				if _, exists := s.remainLengthMap[*agent]; exists {
					s.remainLengthMap[*agent] -= cost
				}
			}
		}
	}

	if s.talkSetting.MaxLength.PerTalk != nil {
		commonLength := util.CountLength(commonText, *s.talkSetting.MaxLength.CountInWord, *s.talkSetting.MaxLength.CountSpaces)
		mentionLength := util.CountLength(mentionText, *s.talkSetting.MaxLength.CountInWord, *s.talkSetting.MaxLength.CountSpaces)
		totalLength := commonLength + mentionLength

		if totalLength > *s.talkSetting.MaxLength.PerTalk {
			if commonLength > *s.talkSetting.MaxLength.PerTalk {
				commonText = util.TrimLength(commonText, *s.talkSetting.MaxLength.PerTalk, *s.talkSetting.MaxLength.CountInWord, *s.talkSetting.MaxLength.CountSpaces)
				mention = ""
				mentionText = ""
			} else {
				mentionText = util.TrimLength(mentionText, *s.talkSetting.MaxLength.PerTalk-commonLength, *s.talkSetting.MaxLength.CountInWord, *s.talkSetting.MaxLength.CountSpaces)
			}
			slog.Warn("発言が最大文字数を超えたため、切り捨てました", "id", s.game.id, "agent", agent.String())
		}
	}

	text = commonText + mention + mentionText
	if utf8.RuneCountInString(text) == 0 {
		text = model.T_OVER
		slog.Warn("文字数が0のため、発言をオーバーに置換しました", "id", s.game.id, "agent", agent.String())
	}

	return text
}

func (s *CommunicationSession) buildTalk(agent *model.Agent, text string, turn int) model.Talk {
	s.remainCountMap[*agent]--

	text = s.processSkipOver(agent, text)

	if text != model.T_OVER && text != model.T_SKIP && text != model.T_FORCE_SKIP {
		text = s.processText(agent, text)
	}

	talk := model.Talk{
		Idx:   s.idx,
		Day:   s.game.getCurrentGameStatus().Day,
		Turn:  turn,
		Agent: *agent,
		Text:  text,
	}
	s.idx++

	return talk
}

func (s *CommunicationSession) appendTalk(talk model.Talk) {
	*s.talkList = append(*s.talkList, talk)
}

func (s *CommunicationSession) logTalk(talk model.Talk) {
	if s.game.gameLogger != nil {
		if s.request == model.R_TALK {
			s.game.gameLogger.AppendLog(s.game.id, fmt.Sprintf("%d,talk,%d,%d,%d,%s", s.game.currentDay, talk.Idx, talk.Turn, talk.Agent.Idx, talk.Text))
		} else {
			s.game.gameLogger.AppendLog(s.game.id, fmt.Sprintf("%d,whisper,%d,%d,%d,%s", s.game.currentDay, talk.Idx, talk.Turn, talk.Agent.Idx, talk.Text))
		}
	}
	if s.game.realtimeBroadcaster != nil {
		packet := s.game.getRealtimeBroadcastPacket()
		if s.request == model.R_TALK {
			packet.Event = "トーク"
		} else {
			packet.Event = "囁き"
		}
		packet.Message = &talk.Text
		packet.BubbleIdx = &talk.Agent.Idx
		s.game.realtimeBroadcaster.Broadcast(packet)
	}
	if s.game.ttsBroadcaster != nil && talk.Agent.Profile != nil {
		s.game.ttsBroadcaster.BroadcastText(s.game.id, talk.Text, talk.Agent.Profile.VoiceID)
	}
}
