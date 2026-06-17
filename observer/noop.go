package observer

import (
	"encoding/json"

	"github.com/aiwolfdial/aiwolf-nlp-server/model"
)

// NoopObserver implements GameObserver with empty methods. Embed it in an
// adapter to inherit no-op defaults for the events it does not handle, and use
// it as the default observer so Game never holds a nil sink.
type NoopObserver struct{}

func (NoopObserver) OnGameStart(string, []model.AgentView)                         {}
func (NoopObserver) OnGameEnd(string, model.Team)                                  {}
func (NoopObserver) OnRequest(string, model.AgentView, json.RawMessage)            {}
func (NoopObserver) OnResponse(string, model.AgentView, string, error)             {}
func (NoopObserver) OnTalk(string, model.AgentView, model.Request, model.TalkView) {}
func (NoopObserver) OnPhase(string, model.Request)                                 {}
func (NoopObserver) OnLogLine(string, string)                                      {}
func (NoopObserver) OnBroadcast(model.BroadcastPacket)                             {}
func (NoopObserver) OnStreamCreate(string)                                         {}
func (NoopObserver) OnSpeak(string, string, int)                                   {}
