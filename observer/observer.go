package observer

import (
	"encoding/json"

	"github.com/aiwolfdial/aiwolf-nlp-server/model"
)

// modelのみに依存する葉パッケージとし、logicとcoreの双方から循環なく参照できるようにする。
// 渡す値はすべて読み取り専用（値型・ビュー・マーシャル済みバイト）。
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
