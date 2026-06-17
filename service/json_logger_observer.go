package service

import (
	"encoding/json"

	"github.com/aiwolfdial/aiwolf-nlp-server/model"
	"github.com/aiwolfdial/aiwolf-nlp-server/observer"
)

// jsonLoggerObserver adapts JSONLogger to the observer.GameObserver interface.
// It embeds NoopObserver so it only has to implement the events it cares about.
type jsonLoggerObserver struct {
	observer.NoopObserver
	l *JSONLogger
}

// AsObserver returns the JSONLogger as a GameObserver.
func (j *JSONLogger) AsObserver() observer.GameObserver {
	return jsonLoggerObserver{l: j}
}

func (o jsonLoggerObserver) OnGameStart(id string, agents []model.AgentView) {
	o.l.TrackStartGame(id, agents)
}

func (o jsonLoggerObserver) OnGameEnd(id string, winSide model.Team) {
	o.l.TrackEndGame(id, winSide)
}

func (o jsonLoggerObserver) OnRequest(id string, agent model.AgentView, request json.RawMessage) {
	o.l.TrackStartRequest(id, agent, request)
}

func (o jsonLoggerObserver) OnResponse(id string, agent model.AgentView, response string, err error) {
	o.l.TrackEndRequest(id, agent, response, err)
}

func (o jsonLoggerObserver) OnTalk(id string, agent model.AgentView, request model.Request, talk model.TalkView) {
	o.l.TrackTalk(id, agent, request, talk)
}

func (o jsonLoggerObserver) OnPhase(id string, request model.Request) {
	o.l.TrackPhase(id, request)
}
