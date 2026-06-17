package livestate

import (
	"sync"
	"time"

	"github.com/aiwolfdial/aiwolf-nlp-server/model"
	"github.com/aiwolfdial/aiwolf-nlp-server/observer"
)

const subscriberBuffer = 64

// 各ゲームの現在状態を保持し、ブロードキャストを購読者へ配信するobserver。
// イベント→パケットの変換は observer.Broadcaster を埋め込んで共有する。
type LiveState struct {
	observer.Broadcaster
	mu    sync.Mutex
	games map[string]*gameState
}

type gameState struct {
	agents      []model.AgentView
	finished    bool
	winSide     model.Team
	packetIdx   int
	lastPacket  *model.BroadcastPacket
	subscribers map[int]chan model.BroadcastPacket
	nextSubID   int
}

func New() *LiveState {
	l := &LiveState{games: make(map[string]*gameState)}
	l.Broadcaster.Emit = l.emit
	return l
}

func (l *LiveState) OnGameStart(id string, agents []model.AgentView, state model.GameState) {
	l.mu.Lock()
	l.games[id] = &gameState{
		agents:      append([]model.AgentView(nil), agents...),
		subscribers: make(map[int]chan model.BroadcastPacket),
	}
	l.mu.Unlock()
	l.Broadcaster.OnGameStart(id, agents, state)
}

func (l *LiveState) OnGameEnd(id string, winSide model.Team, state model.GameState) {
	l.Broadcaster.OnGameEnd(id, winSide, state)
	l.mu.Lock()
	defer l.mu.Unlock()
	gs := l.games[id]
	if gs == nil {
		return
	}
	gs.finished = true
	gs.winSide = winSide
	for _, ch := range gs.subscribers {
		close(ch)
	}
	gs.subscribers = nil
	delete(l.games, id)
}

func (l *LiveState) emit(id string, state model.GameState, event string, message *string, fromIdx *int, toIdx *int, bubbleIdx *int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	gs := l.games[id]
	if gs == nil {
		return
	}
	gs.packetIdx++
	packet := model.BroadcastPacket{
		Id:        id,
		Idx:       gs.packetIdx,
		Day:       state.Day,
		IsDay:     state.IsDaytime,
		Agents:    state.Agents,
		Event:     event,
		Message:   message,
		FromIdx:   fromIdx,
		ToIdx:     toIdx,
		BubbleIdx: bubbleIdx,
		Timestamp: time.Now().Unix(),
	}
	gs.lastPacket = &packet
	for _, ch := range gs.subscribers {
		select {
		case ch <- packet:
		default:
			// 満杯なら最古を捨てて再投入する。ゲームのgoroutineを遅い購読者でブロックさせない。
			select {
			case <-ch:
			default:
			}
			select {
			case ch <- packet:
			default:
			}
		}
	}
}

// 購読チャネルとその解除関数を返す。チャネルはゲーム終了時にcloseされる。
// 直近のパケットがあれば即座に配信し、遅れて参加した購読者にも現在状態を見せる。
func (l *LiveState) Subscribe(id string) (<-chan model.BroadcastPacket, func(), bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	gs := l.games[id]
	if gs == nil {
		return nil, nil, false
	}
	ch := make(chan model.BroadcastPacket, subscriberBuffer)
	subID := gs.nextSubID
	gs.nextSubID++
	gs.subscribers[subID] = ch
	if gs.lastPacket != nil {
		select {
		case ch <- *gs.lastPacket:
		default:
		}
	}
	cancel := func() {
		l.mu.Lock()
		defer l.mu.Unlock()
		if cur := l.games[id]; cur != nil {
			if _, ok := cur.subscribers[subID]; ok {
				delete(cur.subscribers, subID)
				close(ch)
			}
		}
	}
	return ch, cancel, true
}

// 直近のブロードキャストから現在状態のコピーを組み立てて返す。
func (l *LiveState) Snapshot(id string) (model.GameSnapshot, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	gs := l.games[id]
	if gs == nil {
		return model.GameSnapshot{}, false
	}
	snap := model.GameSnapshot{
		ID:       id,
		Finished: gs.finished,
		WinSide:  gs.winSide,
		Agents:   append([]model.AgentView(nil), gs.agents...),
	}
	if gs.lastPacket != nil {
		snap.Day = gs.lastPacket.Day
		snap.StatusByAgent = make(map[int]string, len(gs.lastPacket.Agents))
		for _, a := range gs.lastPacket.Agents {
			status := "ALIVE"
			if !a.IsAlive {
				status = "DEAD"
			}
			snap.StatusByAgent[a.Idx] = status
		}
	}
	return snap, true
}
