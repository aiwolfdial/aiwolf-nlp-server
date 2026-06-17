package service

import (
	"fmt"
	"strings"

	"github.com/aiwolfdial/aiwolf-nlp-server/model"
	"github.com/aiwolfdial/aiwolf-nlp-server/observer"
)

type gameLoggerObserver struct {
	observer.NoopObserver
	l *GameLogger
}

func (g *GameLogger) AsObserver() observer.GameObserver {
	return gameLoggerObserver{l: g}
}

// 各フィールドをコンマ区切りの1行に整形する。
func csvLine(fields ...any) string {
	parts := make([]string, len(fields))
	for i, f := range fields {
		parts[i] = fmt.Sprint(f)
	}
	return strings.Join(parts, ",")
}

func (o gameLoggerObserver) OnGameStart(id string, agents []model.AgentView, _ model.GameState) {
	o.l.TrackStartGame(id, agents)
}

func (o gameLoggerObserver) OnGameEnd(id string, _ model.Team, _ model.GameState) {
	o.l.TrackEndGame(id)
}

func (o gameLoggerObserver) OnDayStatus(id string, day int, statuses []model.AgentStatus) {
	for _, s := range statuses {
		o.l.AppendLog(id, csvLine(day, "status", s.Idx, s.Role, s.Status, s.OriginalName, s.GameName))
	}
}

func (o gameLoggerObserver) OnResult(id string, day int, villagers int, werewolves int, winSide model.Team) {
	o.l.AppendLog(id, csvLine(day, "result", villagers, werewolves, winSide))
}

func (o gameLoggerObserver) OnTalk(id string, day int, request model.Request, talk model.TalkView, _ *int, _ model.GameState) {
	kind := "talk"
	if request != model.R_TALK {
		kind = "whisper"
	}
	o.l.AppendLog(id, csvLine(day, kind, talk.Idx, talk.Turn, talk.Agent.Idx, talk.Text, talk.Time.Unix()))
}

func (o gameLoggerObserver) OnVote(id string, day int, agent model.AgentView, target model.AgentView, _ model.GameState) {
	o.l.AppendLog(id, csvLine(day, "vote", agent.Idx, target.Idx))
}

func (o gameLoggerObserver) OnAttackVote(id string, day int, agent model.AgentView, target model.AgentView, _ model.GameState) {
	o.l.AppendLog(id, csvLine(day, "attackVote", agent.Idx, target.Idx))
}

func (o gameLoggerObserver) OnExecute(id string, day int, executed *model.AgentView, _ model.GameState) {
	if executed == nil {
		return
	}
	o.l.AppendLog(id, csvLine(day, "execute", executed.Idx, executed.Role.Name))
}

func (o gameLoggerObserver) OnDivine(id string, day int, agent model.AgentView, target model.AgentView, _ model.GameState) {
	o.l.AppendLog(id, csvLine(day, "divine", agent.Idx, target.Idx, target.Role.Species))
}

func (o gameLoggerObserver) OnGuard(id string, day int, agent model.AgentView, target model.AgentView, _ model.GameState) {
	o.l.AppendLog(id, csvLine(day, "guard", agent.Idx, target.Idx, target.Role.Name))
}

func (o gameLoggerObserver) OnAttack(id string, day int, attacked *model.AgentView, guarded bool, _ model.GameState) {
	if attacked == nil {
		o.l.AppendLog(id, csvLine(day, "attack", -1, true))
		return
	}
	o.l.AppendLog(id, csvLine(day, "attack", attacked.Idx, !guarded))
}
