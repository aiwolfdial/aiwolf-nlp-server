package model

import (
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/aiwolfdial/aiwolf-nlp-server/model/wire"
)

// スキーマへ項目を足したのにドメイン型からwire型への変換を書き忘れると、その項目は
// 常にゼロ値のまま送られてしまう。全項目を埋めたパケットを変換し、ゼロ値が残らないことで検出する。
func TestPacketWireConversionCoversAllFields(t *testing.T) {
	packet := fullPacket()

	covered := map[string]bool{}
	markNonZero(reflect.ValueOf(packet.wire()), covered)

	expected := map[string]bool{}
	collectFields(reflect.TypeOf(wire.Packet{}), expected, map[reflect.Type]bool{})

	var missing []string
	for field := range expected {
		if !covered[field] {
			missing = append(missing, field)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		t.Fatalf("wire型の以下の項目が変換されていません:\n  %s", strings.Join(missing, "\n  "))
	}
}

func fullPacket() Packet {
	agent := &Agent{Idx: 1, GameName: "Agent[01]", Role: R_SEER}
	target := Agent{Idx: 2, GameName: "Agent[02]", Role: R_WEREWOLF}
	at := time.UnixMilli(1700000000000)

	judge := &Judge{Day: 1, Agent: *agent, Target: target, Result: S_WEREWOLF}
	votes := []Vote{{Day: 1, Agent: *agent, Target: target}}
	// skipとoverは本文から決まるため、両方が真になる履歴を用意する。
	talks := []Talk{
		{Idx: 1, Day: 1, Turn: 1, Agent: *agent, Text: "こんにちは", Time: at},
		{Idx: 2, Day: 1, Turn: 1, Agent: *agent, Text: T_SKIP, Time: at},
		{Idx: 3, Day: 1, Turn: 1, Agent: *agent, Text: T_OVER, Time: at},
	}

	info := &Info{
		GameID:         "game",
		Day:            1,
		Agent:          agent,
		Profile:        ptrOf("プロフィール"),
		MediumResult:   judge,
		DivineResult:   judge,
		ExecutedAgent:  &target,
		AttackedAgent:  &target,
		VoteList:       votes,
		AttackVoteList: votes,
		StatusMap:      map[Agent]Status{*agent: S_ALIVE},
		RoleMap:        map[Agent]Role{*agent: R_SEER},
		RemainCount:    ptrOf(1),
		RemainLength:   ptrOf(1),
		RemainSkip:     ptrOf(1),
	}

	talkSetting := TalkSetting{
		MaxCount:  wire.SettingTalkMaxCount{PerAgent: 1, PerDay: 1},
		MaxLength: wire.SettingTalkMaxLength{CountInWord: ptrOf(true), CountSpaces: ptrOf(true), PerTalk: ptrOf(1), MentionLength: ptrOf(1), PerAgent: ptrOf(1), BaseLength: ptrOf(1)},
		Duration:  ptrOf(1),
		MaxSkip:   1,
	}
	setting := &Setting{
		AgentCount:     5,
		MaxDay:         ptrOf(1),
		RoleNumMap:     map[Role]int{R_SEER: 1},
		VoteVisibility: true,
		Talk:           talkSetting,
		Whisper:        talkSetting,
		Vote:           wire.SettingVote{MaxCount: 1, AllowSelfVote: true},
		AttackVote:     wire.SettingAttackVote{MaxCount: 1, AllowSelfVote: true, AllowNoTarget: true},
		Timeout:        wire.SettingTimeout{Action: 1, Response: 1},
	}

	return Packet{
		Request:        &R_TALK,
		Info:           info,
		Setting:        setting,
		TalkHistory:    &talks,
		WhisperHistory: &talks,
		NewTalk:        &talks[0],
		NewWhisper:     &talks[0],
	}
}

func ptrOf[T any](v T) *T { return &v }

// 値をたどり、ゼロ値でない項目を「型名.フィールド名」で記録する。
func markNonZero(v reflect.Value, covered map[string]bool) {
	switch v.Kind() {
	case reflect.Pointer, reflect.Interface:
		if v.IsNil() {
			return
		}
		markNonZero(v.Elem(), covered)
	case reflect.Slice, reflect.Array:
		for i := range v.Len() {
			markNonZero(v.Index(i), covered)
		}
	case reflect.Map:
		for _, key := range v.MapKeys() {
			markNonZero(v.MapIndex(key), covered)
		}
	case reflect.Struct:
		typ := v.Type()
		for i := range typ.NumField() {
			if !v.Field(i).IsZero() {
				covered[typ.Name()+"."+typ.Field(i).Name] = true
			}
			markNonZero(v.Field(i), covered)
		}
	}
}

func collectFields(t reflect.Type, out map[string]bool, seen map[reflect.Type]bool) {
	for t.Kind() == reflect.Pointer || t.Kind() == reflect.Slice || t.Kind() == reflect.Array || t.Kind() == reflect.Map {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct || seen[t] {
		return
	}
	seen[t] = true
	for i := range t.NumField() {
		out[t.Name()+"."+t.Field(i).Name] = true
		collectFields(t.Field(i).Type, out, seen)
	}
}
