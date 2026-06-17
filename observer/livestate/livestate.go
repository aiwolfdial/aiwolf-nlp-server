// Package livestate provides an in-memory GameObserver that tracks the current
// state of each running game and lets HTTP clients subscribe to a live stream of
// broadcast events (e.g. over SSE), without polling files.
package livestate

import (
	"sync"

	"github.com/aiwolfdial/aiwolf-nlp-server/model"
	"github.com/aiwolfdial/aiwolf-nlp-server/observer"
)

// subscriberBuffer bounds how many pending events a slow subscriber may hold
// before the oldest is dropped. Dropping (rather than blocking) guarantees the
// game goroutine is never stalled by a slow HTTP client.
const subscriberBuffer = 64

// LiveState is a GameObserver that keeps the latest snapshot of each game and
// fans broadcast events out to subscribers. It embeds NoopObserver so it only
// implements the events it needs.
type LiveState struct {
	observer.NoopObserver
	mu    sync.Mutex
	games map[string]*gameState
}

type gameState struct {
	agents      []model.AgentView
	finished    bool
	winSide     model.Team
	lastPacket  *model.BroadcastPacket
	subscribers map[int]chan model.BroadcastPacket
	nextSubID   int
}

// New returns an empty LiveState.
func New() *LiveState {
	return &LiveState{games: make(map[string]*gameState)}
}

func (l *LiveState) OnGameStart(id string, agents []model.AgentView) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.games[id] = &gameState{
		agents:      append([]model.AgentView(nil), agents...),
		subscribers: make(map[int]chan model.BroadcastPacket),
	}
}

func (l *LiveState) OnBroadcast(packet model.BroadcastPacket) {
	l.mu.Lock()
	defer l.mu.Unlock()
	gs := l.games[packet.Id]
	if gs == nil {
		return
	}
	p := packet
	gs.lastPacket = &p
	for _, ch := range gs.subscribers {
		select {
		case ch <- packet:
		default:
			// Subscriber is behind: drop its oldest event and try again so we
			// never block the game goroutine.
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

func (l *LiveState) OnGameEnd(id string, winSide model.Team) {
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

// Subscribe registers a subscriber for the game's broadcast stream. It returns a
// receive-only channel, a cancel func to unsubscribe, and whether the game
// exists. The channel is closed when the game ends. The most recent packet (if
// any) is delivered immediately so a late subscriber sees current state.
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

// Snapshot returns the current state of a game, derived from the latest
// broadcast. All collections are copies.
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
