package model

import (
	"encoding/json"

	"github.com/aiwolfdial/aiwolf-nlp-server/model/wire"
)

// エージェントへ送るパケット。JSONの形は wire.Packet が唯一の定義なので、
// ここへフィールドを足しても schema/protocol.schema.json を更新しない限り送信されない。
type Packet struct {
	Request        *Request
	Info           *Info
	Setting        *Setting
	TalkHistory    *[]Talk
	WhisperHistory *[]Talk
	NewTalk        *Talk
	NewWhisper     *Talk
}

func (p Packet) wire() wire.Packet {
	packet := wire.Packet{
		Info:           wireInfo(p.Info),
		Setting:        wireSetting(p.Setting),
		TalkHistory:    wireTalks[wire.PacketTalkHistory](p.TalkHistory),
		WhisperHistory: wireTalks[wire.PacketWhisperHistory](p.WhisperHistory),
		NewTalk:        wireTalk(p.NewTalk),
		NewWhisper:     wireTalk(p.NewWhisper),
	}
	if p.Request != nil {
		packet.Request = wire.Request(p.Request.Type)
	}
	return packet
}

func (p Packet) MarshalJSON() ([]byte, error) {
	return json.Marshal(p.wire())
}

// 履歴が空でも「空配列を送った」ことを伝える必要があるため、nilと空スライスを取り違えないようポインタで返す。
// トークと囁きで生成される型が別なので型引数で受ける。
func wireTalks[T ~[]wire.Talk](talks *[]Talk) *T {
	if talks == nil {
		return nil
	}
	out := make(T, 0, len(*talks))
	for _, t := range *talks {
		out = append(out, t.wire())
	}
	return &out
}

func wireTalk(talk *Talk) *wire.Talk {
	if talk == nil {
		return nil
	}
	t := talk.wire()
	return &t
}

func wireInfo(info *Info) *wire.Info {
	if info == nil {
		return nil
	}
	i := info.wire()
	return &i
}

func wireSetting(setting *Setting) *wire.Setting {
	if setting == nil {
		return nil
	}
	s := setting.wire()
	return &s
}
