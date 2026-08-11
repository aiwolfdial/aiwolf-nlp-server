package model

import (
	"encoding/json"
	"time"

	"github.com/aiwolfdial/aiwolf-nlp-server/model/wire"
)

type Talk struct {
	Idx   int
	Day   int
	Turn  int
	Agent Agent
	Text  string
	Time  time.Time
}

// Unixミリ秒はint32に収まらないため、intが64bitでない環境ではこの式がゼロ除算となりビルドが止まる。
const _ = 1 / (^uint(0) >> 63)

// skip・overは本文から導出し、timeはUnixミリ秒へ変換して送る。
func (t Talk) wire() wire.Talk {
	return wire.Talk{
		Idx:   t.Idx,
		Day:   t.Day,
		Turn:  t.Turn,
		Agent: t.Agent.String(),
		Text:  t.Text,
		Skip:  t.Text == T_SKIP || t.Text == T_FORCE_SKIP,
		Over:  t.Text == T_OVER,
		Time:  int(t.Time.UnixMilli()),
	}
}

func (t Talk) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.wire())
}

const (
	T_OVER       = "Over"
	T_SKIP       = "Skip"
	T_FORCE_SKIP = "ForceSkip"
)
