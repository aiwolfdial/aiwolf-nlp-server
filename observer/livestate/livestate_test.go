package livestate

import (
	"testing"
	"time"

	"github.com/aiwolfdial/aiwolf-nlp-server/model"
)

func TestSubscribeReceivesBroadcasts(t *testing.T) {
	ls := New()
	ls.OnGameStart("g1", []model.AgentView{{Idx: 1}})

	ch, cancel, ok := ls.Subscribe("g1")
	if !ok {
		t.Fatal("subscribe to existing game failed")
	}
	defer cancel()

	ls.OnBroadcast(model.BroadcastPacket{Id: "g1", Day: 2, Event: "talk"})

	select {
	case pkt := <-ch:
		if pkt.Event != "talk" || pkt.Day != 2 {
			t.Fatalf("unexpected packet: %+v", pkt)
		}
	case <-time.After(time.Second):
		t.Fatal("no packet received within timeout")
	}
}

func TestSubscribeUnknownGame(t *testing.T) {
	ls := New()
	if _, _, ok := ls.Subscribe("missing"); ok {
		t.Fatal("expected subscribe to fail for unknown game")
	}
}

func TestSnapshotReflectsLatestBroadcast(t *testing.T) {
	ls := New()
	ls.OnGameStart("g1", []model.AgentView{{Idx: 1, GameName: "A"}})

	packet := model.BroadcastPacket{Id: "g1", Day: 3}
	packet.Agents = append(packet.Agents, struct {
		Idx     int     `json:"idx"`
		Team    string  `json:"team"`
		Name    string  `json:"name"`
		Profile *string `json:"profile,omitempty"`
		Avatar  *string `json:"avatar,omitempty"`
		Role    string  `json:"role"`
		IsAlive bool    `json:"is_alive"`
	}{Idx: 1, IsAlive: false})
	ls.OnBroadcast(packet)

	snap, ok := ls.Snapshot("g1")
	if !ok {
		t.Fatal("snapshot of existing game failed")
	}
	if snap.Day != 3 {
		t.Fatalf("day: got %d", snap.Day)
	}
	if snap.StatusByAgent[1] != "DEAD" {
		t.Fatalf("status: got %q", snap.StatusByAgent[1])
	}
}

func TestChannelClosesOnGameEnd(t *testing.T) {
	ls := New()
	ls.OnGameStart("g1", nil)
	ch, cancel, _ := ls.Subscribe("g1")
	defer cancel()

	ls.OnGameEnd("g1", model.T_WEREWOLF)

	if _, ok := <-ch; ok {
		t.Fatal("expected channel to be closed after game end")
	}
	if _, exists := ls.Snapshot("g1"); exists {
		t.Fatal("expected game to be removed after game end")
	}
}
