package logic

import (
	"log/slog"
	"math/rand"

	"github.com/aiwolfdial/aiwolf-nlp-server/model"
)

func (s *CommunicationSession) runTurnBased() {
	rand.Shuffle(len(s.agents), func(i, j int) {
		s.agents[i], s.agents[j] = s.agents[j], s.agents[i]
	})

	for i := range s.talkSetting.MaxCount.PerDay {
		cnt := false
		for _, agent := range s.agents {
			if !s.canAgentTalk(agent) {
				continue
			}
			text := s.game.getTalkWhisperText(agent, s.request)

			talk := s.buildTalk(agent, text, i)
			s.appendTalk(talk)
			if talk.Text != model.T_OVER {
				cnt = true
			}
			s.logTalk(talk)
			slog.Info("発言を受信しました", "id", s.game.id, "agent", agent.String(), "text", talk.Text, "count", s.remainCountMap[*agent], "length", s.remainLengthMap[*agent], "skip", s.remainSkipMap[*agent])
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
