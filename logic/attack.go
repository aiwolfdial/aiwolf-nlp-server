package logic

import (
	"fmt"
	"log/slog"

	"github.com/aiwolfdial/aiwolf-nlp-server/model"
	"github.com/aiwolfdial/aiwolf-nlp-server/util"
)

func (g *Game) getAttackVotedCandidates(votes []model.Vote) []model.Agent {
	return util.GetCandidates(votes, func(vote model.Vote) bool {
		return vote.Target.Role != model.R_WEREWOLF
	})
}

func (g *Game) doAttack() {
	slog.Info("襲撃フェーズを開始します", "id", g.id, "day", g.currentDay)
	var attacked *model.Agent
	werewolfs := g.getAliveWerewolves()
	if len(werewolfs) > 0 {
		candidates := make([]model.Agent, 0)
		for range g.setting.AttackVoteMaxCount() {
			g.executeAttackVote()
			candidates = g.getAttackVotedCandidates(g.getCurrentGameStatus().AttackVotes)
			if len(candidates) == 1 {
				attacked = &candidates[0]
				break
			}
		}
		if attacked == nil && !g.setting.AttackVoteAllowNoTarget() && len(candidates) > 0 {
			rand := util.SelectRandomAgent(candidates)
			attacked = &rand
		}

		if attacked != nil && !g.isGuarded(attacked) {
			g.getCurrentGameStatus().StatusMap[*attacked] = model.S_DEAD
			g.getCurrentGameStatus().AttackedAgent = attacked
			g.obs.OnLogLine(g.id, fmt.Sprintf("%d,attack,%d,true", g.currentDay, attacked.Idx))
			{
				packet := g.getRealtimeBroadcastPacket()
				packet.Event = "襲撃"
				packet.ToIdx = &attacked.Idx
				g.obs.OnBroadcast(packet)
			}
			slog.Info("襲撃結果を設定しました", "id", g.id, "agent", attacked.String())
		} else if attacked != nil {
			g.obs.OnLogLine(g.id, fmt.Sprintf("%d,attack,%d,false", g.currentDay, attacked.Idx))
			{
				packet := g.getRealtimeBroadcastPacket()
				packet.Event = "襲撃"
				idx := -1
				packet.FromIdx = &idx
				packet.ToIdx = &attacked.Idx
				g.obs.OnBroadcast(packet)
			}
			slog.Info("護衛されたため、襲撃結果を設定しません", "id", g.id, "agent", attacked.String())
		} else {
			g.obs.OnLogLine(g.id, fmt.Sprintf("%d,attack,-1,true", g.currentDay))
			{
				packet := g.getRealtimeBroadcastPacket()
				packet.Event = "襲撃"
				g.obs.OnBroadcast(packet)
			}
			slog.Info("襲撃対象がいないため、襲撃結果を設定しません", "id", g.id)
		}
	}
	slog.Info("襲撃フェーズを終了します", "id", g.id, "day", g.currentDay)
}

func (g *Game) isGuarded(attacked *model.Agent) bool {
	if g.getCurrentGameStatus().Guard == nil {
		return false
	}
	return g.getCurrentGameStatus().Guard.Target == *attacked && g.isAlive(&g.getCurrentGameStatus().Guard.Agent)
}
