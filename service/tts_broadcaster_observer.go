package service

import (
	"github.com/aiwolfdial/aiwolf-nlp-server/observer"
)

type ttsBroadcasterObserver struct {
	observer.NoopObserver
	b *TTSBroadcaster
}

func (tb *TTSBroadcaster) AsObserver() observer.GameObserver {
	return ttsBroadcasterObserver{b: tb}
}

func (o ttsBroadcasterObserver) OnStreamCreate(id string) {
	o.b.CreateStream(id)
}

func (o ttsBroadcasterObserver) OnSpeak(id string, text string, voiceID int) {
	o.b.BroadcastText(id, text, voiceID)
}
