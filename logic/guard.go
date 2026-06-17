package logic

import (
	"log/slog"

	"github.com/aiwolfdial/aiwolf-nlp-server/model"
)

func (g *Game) doGuard() {
	slog.Info("護衛フェーズを開始します", "id", g.id, "day", g.currentDay)
	for _, agent := range g.getAliveAgents() {
		if agent.Role == model.R_BODYGUARD {
			g.conductGuard(agent)
			break
		}
	}
}

func (g *Game) conductGuard(agent *model.Agent) {
	slog.Info("護衛アクションを実行します", "id", g.id, "agent", agent.String())
	target, err := g.findTargetByRequest(agent, model.R_GUARD)
	if err != nil {
		slog.Warn("護衛対象が見つからなかったため、護衛対象を設定しません", "id", g.id)
		return
	}
	if !g.isAlive(target) {
		slog.Warn("護衛対象が死亡しているため、護衛対象を設定しません", "id", g.id, "target", target.String())
		return
	}
	if agent == target {
		slog.Warn("護衛対象が自分自身であるため、護衛対象を設定しません", "id", g.id, "target", target.String())
		return
	}
	g.getCurrentGameStatus().Guard = &model.Guard{
		Day:    g.getCurrentGameStatus().Day,
		Agent:  *agent,
		Target: *target,
	}
	g.obs.OnGuard(g.id, g.currentDay, agent.View(), target.View(), g.gameState())
	slog.Info("護衛対象を設定しました", "id", g.id, "target", target.String())
}
