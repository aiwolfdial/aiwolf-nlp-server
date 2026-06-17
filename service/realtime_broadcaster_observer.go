package service

import (
	"github.com/aiwolfdial/aiwolf-nlp-server/model"
	"github.com/aiwolfdial/aiwolf-nlp-server/observer"
)

// realtimeBroadcasterObserver adapts RealtimeBroadcaster to observer.GameObserver.
type realtimeBroadcasterObserver struct {
	observer.NoopObserver
	b *RealtimeBroadcaster
}

// AsObserver returns the RealtimeBroadcaster as a GameObserver.
func (rb *RealtimeBroadcaster) AsObserver() observer.GameObserver {
	return realtimeBroadcasterObserver{b: rb}
}

func (o realtimeBroadcasterObserver) OnGameStart(id string, agents []model.AgentView) {
	o.b.TrackStartGame(id, agents)
}

func (o realtimeBroadcasterObserver) OnGameEnd(id string, _ model.Team) {
	o.b.TrackEndGame(id)
}

func (o realtimeBroadcasterObserver) OnBroadcast(packet model.BroadcastPacket) {
	o.b.Broadcast(packet)
}
