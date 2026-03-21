package logic

import (
	"fmt"
	"log/slog"
	"strings"
	"unicode/utf8"

	"github.com/aiwolfdial/aiwolf-nlp-server/model"
	"github.com/aiwolfdial/aiwolf-nlp-server/util"
)

func (g *Game) getTalkContext(request model.Request) (*model.TalkSetting, *[]model.Talk) {
	switch request {
	case model.R_TALK:
		return &g.setting.Talk.TalkSetting, &g.getCurrentGameStatus().Talks
	case model.R_WHISPER:
		return &g.setting.Whisper.TalkSetting, &g.getCurrentGameStatus().Whispers
	default:
		return nil, nil
	}
}

func (g *Game) initRemainMaps(agents []*model.Agent, talkSetting *model.TalkSetting) (map[model.Agent]int, map[model.Agent]int, map[model.Agent]int) {
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
	g.getCurrentGameStatus().RemainCountMap = &remainCountMap
	g.getCurrentGameStatus().RemainLengthMap = &remainLengthMap
	g.getCurrentGameStatus().RemainSkipMap = &remainSkipMap
	return remainCountMap, remainLengthMap, remainSkipMap
}

func (g *Game) clearRemainMaps() {
	g.getCurrentGameStatus().RemainCountMap = nil
	g.getCurrentGameStatus().RemainLengthMap = nil
	g.getCurrentGameStatus().RemainSkipMap = nil
}

func canAgentTalk(agent *model.Agent, remainCountMap *map[model.Agent]int, remainLengthMap *map[model.Agent]int) bool {
	if (*remainCountMap)[*agent] <= 0 {
		return false
	}
	if value, exists := (*remainLengthMap)[*agent]; exists {
		if value <= 0 {
			return false
		}
	}
	return true
}

func (g *Game) processSkipOver(agent *model.Agent, text string, talkSetting *model.TalkSetting, remainCountMap *map[model.Agent]int, remainSkipMap *map[model.Agent]int) string {
	switch text {
	case model.T_SKIP:
		if (*remainSkipMap)[*agent] <= 0 {
			text = model.T_OVER
			slog.Warn("スキップ回数が上限に達したため、発言をオーバーに置換しました", "id", g.id, "agent", agent.String())
		} else {
			(*remainSkipMap)[*agent]--
			slog.Info("発言をスキップしました", "id", g.id, "agent", agent.String())
		}
	case model.T_FORCE_SKIP:
		text = model.T_SKIP
		slog.Warn("強制スキップが指定されたため、発言をスキップに置換しました", "id", g.id, "agent", agent.String())
	}

	if text != model.T_OVER && text != model.T_SKIP && text != model.T_FORCE_SKIP {
		(*remainSkipMap)[*agent] = talkSetting.MaxSkip
		slog.Info("発言がオーバーもしくはスキップではないため、スキップ回数をリセットしました", "id", g.id, "agent", agent.String())
	}

	if text == model.T_OVER {
		(*remainCountMap)[*agent] = 0
		slog.Info("発言がオーバーであるため、残り発言回数を0にしました", "id", g.id, "agent", agent.String())
	}

	return text
}

func (g *Game) trimTextByLength(agent *model.Agent, text string, talkSetting *model.TalkSetting, remainLengthMap *map[model.Agent]int) string {
	if text == model.T_OVER || text == model.T_SKIP || text == model.T_FORCE_SKIP {
		return text
	}

	mention := ""
	commonText := ""
	mentionText := ""

	if talkSetting.MaxLength.PerAgent != nil || talkSetting.MaxLength.BaseLength != nil {
		baseLength := 0
		if talkSetting.MaxLength.BaseLength != nil {
			baseLength = *talkSetting.MaxLength.BaseLength
		}

		mentionIdx := -1
		if talkSetting.MaxLength.MentionLength != nil {
			for _, a := range g.agents {
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
			if value, exists := (*remainLengthMap)[*agent]; exists {
				remainLength += value
			}
			mentionBefore := text[:mentionIdx]
			mentionAfter := text[mentionIdx+len(mention):]

			mention = " " + mention + " "

			commonText = util.TrimLength(mentionBefore, remainLength, *talkSetting.MaxLength.CountInWord, *talkSetting.MaxLength.CountSpaces)
			cost := util.CountLength(mentionBefore, *talkSetting.MaxLength.CountInWord, *talkSetting.MaxLength.CountSpaces) - baseLength
			if cost > 0 {
				if _, exists := (*remainLengthMap)[*agent]; exists {
					(*remainLengthMap)[*agent] -= cost
				}
			}

			remainLength = *talkSetting.MaxLength.MentionLength
			if value, exists := (*remainLengthMap)[*agent]; exists {
				remainLength += value
			}
			mentionText = util.TrimLength(mentionAfter, remainLength, *talkSetting.MaxLength.CountInWord, *talkSetting.MaxLength.CountSpaces)
			mentionCost := util.CountLength(mentionText, *talkSetting.MaxLength.CountInWord, *talkSetting.MaxLength.CountSpaces) - *talkSetting.MaxLength.MentionLength
			if mentionCost > 0 {
				if _, exists := (*remainLengthMap)[*agent]; exists {
					(*remainLengthMap)[*agent] -= mentionCost
				}
			}
		} else {
			remainLength := baseLength
			if value, exists := (*remainLengthMap)[*agent]; exists {
				remainLength += value
			}
			commonText = util.TrimLength(text, remainLength, *talkSetting.MaxLength.CountInWord, *talkSetting.MaxLength.CountSpaces)
			cost := util.CountLength(text, *talkSetting.MaxLength.CountInWord, *talkSetting.MaxLength.CountSpaces) - baseLength
			if cost > 0 {
				if _, exists := (*remainLengthMap)[*agent]; exists {
					(*remainLengthMap)[*agent] -= cost
				}
			}
		}
	}

	if talkSetting.MaxLength.PerTalk != nil {
		commonLength := util.CountLength(commonText, *talkSetting.MaxLength.CountInWord, *talkSetting.MaxLength.CountSpaces)
		mentionLength := util.CountLength(mentionText, *talkSetting.MaxLength.CountInWord, *talkSetting.MaxLength.CountSpaces)
		totalLength := commonLength + mentionLength

		if totalLength > *talkSetting.MaxLength.PerTalk {
			if commonLength > *talkSetting.MaxLength.PerTalk {
				commonText = util.TrimLength(commonText, *talkSetting.MaxLength.PerTalk, *talkSetting.MaxLength.CountInWord, *talkSetting.MaxLength.CountSpaces)
				mention = ""
				mentionText = ""
			} else {
				mentionText = util.TrimLength(mentionText, *talkSetting.MaxLength.PerTalk-commonLength, *talkSetting.MaxLength.CountInWord, *talkSetting.MaxLength.CountSpaces)
			}
			slog.Warn("発言が最大文字数を超えたため、切り捨てました", "id", g.id, "agent", agent.String())
		}
	}

	text = commonText + mention + mentionText
	if utf8.RuneCountInString(text) == 0 {
		text = model.T_OVER
		slog.Warn("文字数が0のため、発言をオーバーに置換しました", "id", g.id, "agent", agent.String())
	}

	return text
}

func (g *Game) processAndCreateTalk(agent *model.Agent, text string, idx int, turn int, talkSetting *model.TalkSetting, remainCountMap *map[model.Agent]int, remainLengthMap *map[model.Agent]int, remainSkipMap *map[model.Agent]int) model.Talk {
	(*remainCountMap)[*agent]--

	text = g.processSkipOver(agent, text, talkSetting, remainCountMap, remainSkipMap)

	if text != model.T_OVER && text != model.T_SKIP && text != model.T_FORCE_SKIP {
		text = g.trimTextByLength(agent, text, talkSetting, remainLengthMap)
	}

	return model.Talk{
		Idx:   idx,
		Day:   g.getCurrentGameStatus().Day,
		Turn:  turn,
		Agent: *agent,
		Text:  text,
	}
}

func (g *Game) logTalk(talk model.Talk, request model.Request) {
	if g.gameLogger != nil {
		if request == model.R_TALK {
			g.gameLogger.AppendLog(g.id, fmt.Sprintf("%d,talk,%d,%d,%d,%s", g.currentDay, talk.Idx, talk.Turn, talk.Agent.Idx, talk.Text))
		} else {
			g.gameLogger.AppendLog(g.id, fmt.Sprintf("%d,whisper,%d,%d,%d,%s", g.currentDay, talk.Idx, talk.Turn, talk.Agent.Idx, talk.Text))
		}
	}
	if g.realtimeBroadcaster != nil {
		packet := g.getRealtimeBroadcastPacket()
		if request == model.R_TALK {
			packet.Event = "トーク"
		} else {
			packet.Event = "囁き"
		}
		packet.Message = &talk.Text
		packet.BubbleIdx = &talk.Agent.Idx
		g.realtimeBroadcaster.Broadcast(packet)
	}
	if g.ttsBroadcaster != nil && talk.Agent.Profile != nil {
		g.ttsBroadcaster.BroadcastText(g.id, talk.Text, talk.Agent.Profile.VoiceID)
	}
}
