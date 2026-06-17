package logic

import (
	"fmt"
	"log/slog"

	"github.com/aiwolfdial/aiwolf-nlp-server/model"
	"github.com/aiwolfdial/aiwolf-nlp-server/util"
)

func (g *Game) getVotedCandidates(votes []model.Vote) []model.Agent {
	return util.GetCandidates(votes, func(vote model.Vote) bool {
		return true
	})
}

func (g *Game) doExecution() {
	slog.Info("追放フェーズを開始します", "id", g.id, "day", g.currentDay)
	var executed *model.Agent
	candidates := make([]model.Agent, 0)
	for range g.setting.VoteMaxCount() {
		g.executeVote()
		candidates = g.getVotedCandidates(g.getCurrentGameStatus().Votes)
		if len(candidates) == 1 {
			executed = &candidates[0]
			break
		}
	}
	if executed == nil && len(candidates) > 0 {
		rand := util.SelectRandomAgent(candidates)
		executed = &rand
	}
	if executed != nil {
		g.getCurrentGameStatus().StatusMap[*executed] = model.S_DEAD
		g.getCurrentGameStatus().ExecutedAgent = executed
		g.obs.OnLogLine(g.id, fmt.Sprintf("%d,execute,%d,%s", g.currentDay, executed.Idx, executed.Role.Name))
		{
			packet := g.getRealtimeBroadcastPacket()
			packet.Event = "追放"
			packet.ToIdx = &executed.Idx
			g.obs.OnBroadcast(packet)
		}
		slog.Info("追放結果を設定しました", "id", g.id, "agent", executed.String())

		g.getCurrentGameStatus().MediumResult = &model.Judge{
			Day:    g.getCurrentGameStatus().Day,
			Agent:  *executed,
			Target: *executed,
			Result: executed.Role.Species,
		}
		slog.Info("霊能結果を設定しました", "id", g.id, "target", executed.String(), "result", executed.Role.Species)
	} else {
		{
			packet := g.getRealtimeBroadcastPacket()
			packet.Event = "追放"
			g.obs.OnBroadcast(packet)
		}
		slog.Warn("追放対象がいないため、追放結果を設定しません", "id", g.id)
	}
	slog.Info("追放フェーズを終了します", "id", g.id, "day", g.currentDay)
}
