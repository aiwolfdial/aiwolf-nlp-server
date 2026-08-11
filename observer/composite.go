package observer

import (
	"encoding/json"

	"github.com/aiwolfdial/aiwolf-nlp-server/model"
)

// 登録順に各observerへイベントを配信する。構築後はスライスを変更しないため並行安全。
type Composite struct {
	observers []GameObserver
}

func NewComposite(observers ...GameObserver) *Composite {
	filtered := make([]GameObserver, 0, len(observers))
	for _, o := range observers {
		if o != nil {
			filtered = append(filtered, o)
		}
	}
	return &Composite{observers: filtered}
}

func (c *Composite) OnGameStart(id string, agents []model.AgentView, state model.GameState) {
	for _, o := range c.observers {
		o.OnGameStart(id, agents, state)
	}
}

func (c *Composite) OnGameEnd(id string, winSide model.Team, state model.GameState) {
	for _, o := range c.observers {
		o.OnGameEnd(id, winSide, state)
	}
}

func (c *Composite) OnDayStatus(id string, day int, statuses []model.AgentStatus) {
	for _, o := range c.observers {
		o.OnDayStatus(id, day, statuses)
	}
}

func (c *Composite) OnResult(id string, day int, villagers int, werewolves int, winSide model.Team) {
	for _, o := range c.observers {
		o.OnResult(id, day, villagers, werewolves, winSide)
	}
}

func (c *Composite) OnTalk(id string, day int, request model.Request, talk model.TalkView, voiceID *int, state model.GameState) {
	for _, o := range c.observers {
		o.OnTalk(id, day, request, talk, voiceID, state)
	}
}

func (c *Composite) OnFreeformTalk(id string, agent model.AgentView, request model.Request, talk model.TalkView) {
	for _, o := range c.observers {
		o.OnFreeformTalk(id, agent, request, talk)
	}
}

func (c *Composite) OnPhase(id string, request model.Request) {
	for _, o := range c.observers {
		o.OnPhase(id, request)
	}
}

func (c *Composite) OnRequest(id string, agent model.AgentView, request json.RawMessage) {
	for _, o := range c.observers {
		o.OnRequest(id, agent, request)
	}
}

func (c *Composite) OnResponse(id string, agent model.AgentView, response string, err error) {
	for _, o := range c.observers {
		o.OnResponse(id, agent, response, err)
	}
}

func (c *Composite) OnAgentFatal(id string, agent model.AgentView, err error) {
	for _, o := range c.observers {
		o.OnAgentFatal(id, agent, err)
	}
}

func (c *Composite) OnVote(id string, day int, agent model.AgentView, target model.AgentView, state model.GameState) {
	for _, o := range c.observers {
		o.OnVote(id, day, agent, target, state)
	}
}

func (c *Composite) OnAttackVote(id string, day int, agent model.AgentView, target model.AgentView, state model.GameState) {
	for _, o := range c.observers {
		o.OnAttackVote(id, day, agent, target, state)
	}
}

func (c *Composite) OnExecute(id string, day int, executed *model.AgentView, state model.GameState) {
	for _, o := range c.observers {
		o.OnExecute(id, day, executed, state)
	}
}

func (c *Composite) OnDivine(id string, day int, agent model.AgentView, target model.AgentView, state model.GameState) {
	for _, o := range c.observers {
		o.OnDivine(id, day, agent, target, state)
	}
}

func (c *Composite) OnGuard(id string, day int, agent model.AgentView, target model.AgentView, state model.GameState) {
	for _, o := range c.observers {
		o.OnGuard(id, day, agent, target, state)
	}
}

func (c *Composite) OnAttack(id string, day int, attacked *model.AgentView, guarded bool, state model.GameState) {
	for _, o := range c.observers {
		o.OnAttack(id, day, attacked, guarded, state)
	}
}
