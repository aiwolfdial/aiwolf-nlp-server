package livestate

import (
	"testing"
	"time"

	"github.com/aiwolfdial/aiwolf-nlp-server/model"
)

func recvWithin(t *testing.T, ch <-chan model.BroadcastPacket, d time.Duration) (model.BroadcastPacket, bool) {
	t.Helper()
	select {
	case pkt, ok := <-ch:
		return pkt, ok
	case <-time.After(d):
		t.Fatal("チャネルからの受信がタイムアウトしました")
		return model.BroadcastPacket{}, false
	}
}

func TestSubscribeReceivesBroadcasts(t *testing.T) {
	ls := New()
	ls.OnGameStart("g1", []model.AgentView{{Idx: 1}}, model.GameState{})

	ch, cancel, ok := ls.Subscribe("g1")
	if !ok {
		t.Fatal("既存ゲームの購読に失敗しました")
	}
	defer cancel()

	// 購読時に直近（開始）パケットが届くので読み捨てる。
	recvWithin(t, ch, time.Second)

	ls.OnExecute("g1", 2, nil, model.GameState{Day: 2})
	pkt, _ := recvWithin(t, ch, time.Second)
	if pkt.Event != "追放" || pkt.Day != 2 {
		t.Fatalf("予期しないパケット: %+v", pkt)
	}
}

func TestSubscribeUnknownGame(t *testing.T) {
	ls := New()
	if _, _, ok := ls.Subscribe("missing"); ok {
		t.Fatal("存在しないゲームの購読は失敗するべき")
	}
}

func TestSnapshotReflectsLatestBroadcast(t *testing.T) {
	ls := New()
	ls.OnGameStart("g1", []model.AgentView{{Idx: 1, GameName: "A"}}, model.GameState{})

	state := model.GameState{Day: 3, Agents: []model.BroadcastAgent{{Idx: 1, IsAlive: false}}}
	ls.OnExecute("g1", 3, nil, state)

	snap, ok := ls.Snapshot("g1")
	if !ok {
		t.Fatal("既存ゲームのスナップショット取得に失敗しました")
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
	ls.OnGameStart("g1", nil, model.GameState{})
	ch, cancel, _ := ls.Subscribe("g1")
	defer cancel()

	ls.OnGameEnd("g1", model.T_WEREWOLF, model.GameState{})

	// 残った開始パケットを読み切ると、終了でcloseされてokがfalseになる。
	for {
		if _, ok := <-ch; !ok {
			break
		}
	}
	if _, exists := ls.Snapshot("g1"); exists {
		t.Fatal("ゲーム終了後はスナップショットが消えているべき")
	}
}
