package observer

import "github.com/aiwolfdial/aiwolf-nlp-server/model"

// EmitFunc は組み立て済みの配信情報（イベント名・状態・各インデックス）を受け取る。
type EmitFunc func(id string, state model.GameState, event string, message *string, fromIdx *int, toIdx *int, bubbleIdx *int)

// Broadcaster はブロードキャスト対象のイベントを EmitFunc 呼び出しへ変換する共有実装。
// realtime ブロードキャスタと LiveState が埋め込んで使い、イベント→配信情報の対応を一元化する。
type Broadcaster struct {
	NoopObserver
	Emit EmitFunc
}

func (b Broadcaster) OnGameStart(id string, _ []model.AgentView, state model.GameState) {
	message := "ゲームが開始されました"
	b.Emit(id, state, "開始", &message, nil, nil, nil)
}

func (b Broadcaster) OnGameEnd(id string, winSide model.Team, state model.GameState) {
	message := string(winSide)
	b.Emit(id, state, "終了", &message, nil, nil, nil)
}

func (b Broadcaster) OnTalk(id string, _ int, request model.Request, talk model.TalkView, _ *int, state model.GameState) {
	event := "トーク"
	if request != model.R_TALK {
		event = "囁き"
	}
	b.Emit(id, state, event, &talk.Text, nil, nil, &talk.Agent.Idx)
}

func (b Broadcaster) OnVote(id string, _ int, agent model.AgentView, target model.AgentView, state model.GameState) {
	b.Emit(id, state, "投票", nil, &agent.Idx, &target.Idx, nil)
}

func (b Broadcaster) OnAttackVote(id string, _ int, agent model.AgentView, target model.AgentView, state model.GameState) {
	b.Emit(id, state, "襲撃投票", nil, &agent.Idx, &target.Idx, nil)
}

func (b Broadcaster) OnExecute(id string, _ int, executed *model.AgentView, state model.GameState) {
	var toIdx *int
	if executed != nil {
		toIdx = &executed.Idx
	}
	b.Emit(id, state, "追放", nil, nil, toIdx, nil)
}

func (b Broadcaster) OnDivine(id string, _ int, agent model.AgentView, target model.AgentView, state model.GameState) {
	b.Emit(id, state, "占い", nil, &agent.Idx, &target.Idx, nil)
}

func (b Broadcaster) OnGuard(id string, _ int, agent model.AgentView, target model.AgentView, state model.GameState) {
	b.Emit(id, state, "護衛", nil, &agent.Idx, &target.Idx, nil)
}

func (b Broadcaster) OnAttack(id string, _ int, attacked *model.AgentView, guarded bool, state model.GameState) {
	var fromIdx *int
	var toIdx *int
	if attacked != nil {
		toIdx = &attacked.Idx
		if guarded {
			idx := -1
			fromIdx = &idx
		}
	}
	b.Emit(id, state, "襲撃", nil, fromIdx, toIdx, nil)
}
