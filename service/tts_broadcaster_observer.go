package service

import (
	"github.com/aiwolfdial/aiwolf-nlp-server/model"
	"github.com/aiwolfdial/aiwolf-nlp-server/observer"
)

type ttsBroadcasterObserver struct {
	observer.NoopObserver
	b *TTSBroadcaster
}

func (tb *TTSBroadcaster) AsObserver() observer.GameObserver {
	return ttsBroadcasterObserver{b: tb}
}

func (o ttsBroadcasterObserver) OnGameStart(id string, _ []model.AgentView, _ model.GameState) {
	o.b.CreateStream(id)
	o.b.BroadcastText(id, "ゲームが開始されました", 23)
}

func (o ttsBroadcasterObserver) OnGameEnd(id string, _ model.Team, _ model.GameState) {
	o.b.BroadcastText(id, "ゲームが終了しました", 23)
}

func (o ttsBroadcasterObserver) OnTalk(id string, _ int, _ model.Request, talk model.TalkView, voiceID *int, _ model.GameState) {
	if voiceID != nil {
		o.b.BroadcastText(id, talk.Text, *voiceID)
	}
}
