package service

import (
	"github.com/aiwolfdial/aiwolf-nlp-server/model"
	"github.com/aiwolfdial/aiwolf-nlp-server/observer"
)

type realtimeBroadcasterObserver struct {
	observer.Broadcaster
	b *RealtimeBroadcaster
}

func (rb *RealtimeBroadcaster) AsObserver() observer.GameObserver {
	o := realtimeBroadcasterObserver{b: rb}
	o.Broadcaster.Emit = rb.Emit
	return o
}

func (o realtimeBroadcasterObserver) OnGameStart(id string, agents []model.AgentView, state model.GameState) {
	o.b.TrackStartGame(id, agents)
	o.Broadcaster.OnGameStart(id, agents, state)
}

func (o realtimeBroadcasterObserver) OnGameEnd(id string, winSide model.Team, state model.GameState) {
	o.Broadcaster.OnGameEnd(id, winSide, state)
	o.b.TrackEndGame(id)
}
