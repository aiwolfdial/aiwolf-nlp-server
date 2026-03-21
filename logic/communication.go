package logic

import (
	"log/slog"
	"math/rand"

	"github.com/aiwolfdial/aiwolf-nlp-server/model"
)

func (g *Game) doWhisper() {
	slog.Info("囁きフェーズを開始します", "id", g.id, "day", g.currentDay)
	g.conductCommunication(model.R_WHISPER)
}

func (g *Game) doTalk() {
	slog.Info("トークフェーズを開始します", "id", g.id, "day", g.currentDay)
	g.conductCommunication(model.R_TALK)
}

func (g *Game) conductCommunication(request model.Request) {
	var agents []*model.Agent
	var talkSetting *model.TalkSetting

	switch request {
	case model.R_TALK:
		agents = g.getAliveAgents()
		talkSetting = &g.setting.Talk.TalkSetting
	case model.R_WHISPER:
		agents = g.getAliveWerewolves()
		talkSetting = &g.setting.Whisper.TalkSetting
	default:
		return
	}
	if len(agents) < 2 {
		slog.Warn("エージェント数が2未満のため、通信を行いません", "id", g.id, "agentNum", len(agents))
		return
	}

	if talkSetting.Duration != nil {
		g.conductFreeformCommunication(request, agents)
	} else {
		g.conductTurnBasedCommunication(request, agents)
	}
}

func (g *Game) conductTurnBasedCommunication(request model.Request, agents []*model.Agent) {
	talkSetting, talkList := g.getTalkContext(request)
	if talkSetting == nil {
		return
	}

	remainCountMap, remainLengthMap, remainSkipMap := g.initRemainMaps(agents, talkSetting)
	defer g.clearRemainMaps()

	rand.Shuffle(len(agents), func(i, j int) {
		agents[i], agents[j] = agents[j], agents[i]
	})

	idx := 0
	for i := range talkSetting.MaxCount.PerDay {
		cnt := false
		for _, agent := range agents {
			if !canAgentTalk(agent, &remainCountMap, &remainLengthMap) {
				continue
			}
			text := g.getTalkWhisperText(agent, request)

			talk := g.buildTalk(agent, text, idx, i, talkSetting, &remainCountMap, &remainLengthMap, &remainSkipMap)
			idx++
			*talkList = append(*talkList, talk)
			if talk.Text != model.T_OVER {
				cnt = true
			}
			g.logTalk(talk, request)
			slog.Info("発言を受信しました", "id", g.id, "agent", agent.String(), "text", talk.Text, "count", remainCountMap[*agent], "length", remainLengthMap[*agent], "skip", remainSkipMap[*agent])
		}
		if !cnt {
			break
		}
	}
}

func (g *Game) getTalkWhisperText(agent *model.Agent, request model.Request) string {
	text, err := g.requestToAgent(agent, request)
	if text == model.T_FORCE_SKIP {
		text = model.T_SKIP
		slog.Warn("クライアントから強制スキップが指定されたため、発言をスキップに置換しました", "id", g.id, "agent", agent.String())
	}
	if err != nil {
		text = model.T_FORCE_SKIP
		slog.Warn("リクエストの送受信に失敗したため、発言をスキップに置換しました", "id", g.id, "agent", agent.String())
	}
	return text
}
