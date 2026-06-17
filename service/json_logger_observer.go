package service

import (
	"encoding/json"

	"github.com/aiwolfdial/aiwolf-nlp-server/model"
	"github.com/aiwolfdial/aiwolf-nlp-server/observer"
)

type jsonLoggerObserver struct {
	observer.NoopObserver
	l *JSONLogger
}

func (j *JSONLogger) AsObserver() observer.GameObserver {
	return jsonLoggerObserver{l: j}
}

func (o jsonLoggerObserver) OnGameStart(id string, agents []model.AgentView, _ model.GameState) {
	o.l.TrackStartGame(id, agents)
}

func (o jsonLoggerObserver) OnGameEnd(id string, winSide model.Team, _ model.GameState) {
	o.l.TrackEndGame(id, winSide)
}

func (o jsonLoggerObserver) OnRequest(id string, agent model.AgentView, request json.RawMessage) {
	o.l.TrackStartRequest(id, agent, request)
}

func (o jsonLoggerObserver) OnResponse(id string, agent model.AgentView, response string, err error) {
	o.l.TrackEndRequest(id, agent, response, err)
}

func (o jsonLoggerObserver) OnFreeformTalk(id string, agent model.AgentView, request model.Request, talk model.TalkView) {
	o.l.TrackTalk(id, agent, request, talk)
}

func (o jsonLoggerObserver) OnPhase(id string, request model.Request) {
	o.l.TrackPhase(id, request)
}
