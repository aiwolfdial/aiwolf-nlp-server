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

func (c *Composite) OnGameStart(id string, agents []model.AgentView) {
	for _, o := range c.observers {
		o.OnGameStart(id, agents)
	}
}

func (c *Composite) OnGameEnd(id string, winSide model.Team) {
	for _, o := range c.observers {
		o.OnGameEnd(id, winSide)
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

func (c *Composite) OnTalk(id string, agent model.AgentView, request model.Request, talk model.TalkView) {
	for _, o := range c.observers {
		o.OnTalk(id, agent, request, talk)
	}
}

func (c *Composite) OnPhase(id string, request model.Request) {
	for _, o := range c.observers {
		o.OnPhase(id, request)
	}
}

func (c *Composite) OnLogLine(id string, line string) {
	for _, o := range c.observers {
		o.OnLogLine(id, line)
	}
}

func (c *Composite) OnBroadcast(packet model.BroadcastPacket) {
	for _, o := range c.observers {
		o.OnBroadcast(packet)
	}
}

func (c *Composite) OnStreamCreate(id string) {
	for _, o := range c.observers {
		o.OnStreamCreate(id)
	}
}

func (c *Composite) OnSpeak(id string, text string, voiceID int) {
	for _, o := range c.observers {
		o.OnSpeak(id, text, voiceID)
	}
}
