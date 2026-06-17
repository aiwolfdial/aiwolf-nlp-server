package service

import (
	"github.com/aiwolfdial/aiwolf-nlp-server/model"
	"github.com/aiwolfdial/aiwolf-nlp-server/observer"
)

// gameLoggerObserver adapts GameLogger to the observer.GameObserver interface.
type gameLoggerObserver struct {
	observer.NoopObserver
	l *GameLogger
}

// AsObserver returns the GameLogger as a GameObserver.
func (g *GameLogger) AsObserver() observer.GameObserver {
	return gameLoggerObserver{l: g}
}

func (o gameLoggerObserver) OnGameStart(id string, agents []model.AgentView) {
	o.l.TrackStartGame(id, agents)
}

func (o gameLoggerObserver) OnGameEnd(id string, _ model.Team) {
	o.l.TrackEndGame(id)
}

func (o gameLoggerObserver) OnLogLine(id string, line string) {
	o.l.AppendLog(id, line)
}
