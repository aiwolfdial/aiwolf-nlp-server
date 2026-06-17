package observer

import (
	"encoding/json"

	"github.com/aiwolfdial/aiwolf-nlp-server/model"
)

// 必要なイベントだけ実装すればよいよう空実装を埋め込み用に提供する。
// 既定のobserverとしても使い、Gameが常に非nilのsinkを持てるようにする。
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
