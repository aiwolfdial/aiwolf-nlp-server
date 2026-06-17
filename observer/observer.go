// Package observer defines the event interface through which the game engine
// reports lifecycle and turn events to sinks (loggers, broadcasters, live-state
// trackers). It is a leaf package depending only on model, so both logic and
// core can import it without creating a dependency cycle.
//
// Every value passed to an observer is read-only — value types, read-only views
// (model.AgentView, model.TalkView) or marshaled bytes — so a sink can never
// mutate live game state or touch a live websocket connection.
package observer

import (
	"encoding/json"

	"github.com/aiwolfdial/aiwolf-nlp-server/model"
)

// GameObserver receives semantic game events. Implementations must be safe for
// concurrent calls across different game ids.
type GameObserver interface {
	OnGameStart(id string, agents []model.AgentView)
	OnGameEnd(id string, winSide model.Team)
	OnRequest(id string, agent model.AgentView, request json.RawMessage)
	OnResponse(id string, agent model.AgentView, response string, err error)
	OnTalk(id string, agent model.AgentView, request model.Request, talk model.TalkView)
	OnPhase(id string, request model.Request)
	OnLogLine(id string, line string)
	OnBroadcast(packet model.BroadcastPacket)
	OnStreamCreate(id string)
	OnSpeak(id string, text string, voiceID int)
}
