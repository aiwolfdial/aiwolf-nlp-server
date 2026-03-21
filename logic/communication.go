package logic

import (
	"log/slog"

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

	s := newCommunicationSession(g, request, agents)
	if s == nil {
		return
	}
	defer s.cleanup()

	if talkSetting.Duration != nil {
		s.runFreeform()
	} else {
		s.runTurnBased()
	}
}
