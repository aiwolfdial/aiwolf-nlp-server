package observer

import (
	"encoding/json"

	"github.com/aiwolfdial/aiwolf-nlp-server/model"
)

// 必要なイベントだけ実装すればよいよう空実装を埋め込み用に提供する。
// 既定のobserverとしても使い、Gameが常に非nilのsinkを持てるようにする。
type NoopObserver struct{}

func (NoopObserver) OnGameStart(string, []model.AgentView, model.GameState)                      {}
func (NoopObserver) OnGameEnd(string, model.Team, model.GameState)                               {}
func (NoopObserver) OnDayStatus(string, int, []model.AgentStatus)                                {}
func (NoopObserver) OnResult(string, int, int, int, model.Team)                                  {}
func (NoopObserver) OnTalk(string, int, model.Request, model.TalkView, *int, model.GameState)    {}
func (NoopObserver) OnFreeformTalk(string, model.AgentView, model.Request, model.TalkView)       {}
func (NoopObserver) OnPhase(string, model.Request)                                               {}
func (NoopObserver) OnRequest(string, model.AgentView, json.RawMessage)                          {}
func (NoopObserver) OnResponse(string, model.AgentView, string, error)                           {}
func (NoopObserver) OnAgentFatal(string, model.AgentView, error)                                 {}
func (NoopObserver) OnVote(string, int, model.AgentView, model.AgentView, model.GameState)       {}
func (NoopObserver) OnAttackVote(string, int, model.AgentView, model.AgentView, model.GameState) {}
func (NoopObserver) OnExecute(string, int, *model.AgentView, model.GameState)                    {}
func (NoopObserver) OnDivine(string, int, model.AgentView, model.AgentView, model.GameState)     {}
func (NoopObserver) OnGuard(string, int, model.AgentView, model.AgentView, model.GameState)      {}
func (NoopObserver) OnAttack(string, int, *model.AgentView, bool, model.GameState)               {}
