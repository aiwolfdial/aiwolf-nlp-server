package model

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/aiwolfdial/aiwolf-nlp-server/model/wire"
)

// 役職の陣営・種族やリクエストの応答要否はGoとPythonの双方で手書きしているため、
// schema/enum_attributes.json を唯一の出所として突き合わせる。
func TestEnumAttributesMatchSchema(t *testing.T) {
	var attrs struct {
		Role struct {
			Values map[string]struct {
				Team    string `json:"team"`
				Species string `json:"species"`
			} `json:"values"`
		} `json:"Role"`
		Request struct {
			Values map[string]struct {
				RequireResponse bool `json:"require_response"`
			} `json:"values"`
		} `json:"Request"`
	}
	readJSON(t, "../schema/enum_attributes.json", &attrs)

	for name, want := range attrs.Role.Values {
		role := RoleFromString(name)
		if role == R_NONE {
			t.Errorf("役職 %s がGo側に定義されていません", name)
			continue
		}
		if string(role.Team) != want.Team {
			t.Errorf("役職 %s の陣営が一致しません: Go=%s スキーマ=%s", name, role.Team, want.Team)
		}
		if string(role.Species) != want.Species {
			t.Errorf("役職 %s の種族が一致しません: Go=%s スキーマ=%s", name, role.Species, want.Species)
		}
	}
	for name, want := range attrs.Request.Values {
		request := RequestFromString(name)
		if request.Type != name {
			t.Errorf("リクエスト %s がGo側に定義されていません", name)
			continue
		}
		if request.RequireResponse != want.RequireResponse {
			t.Errorf("リクエスト %s の応答要否が一致しません: Go=%t スキーマ=%t",
				name, request.RequireResponse, want.RequireResponse)
		}
	}
}

// 手書きの列挙が、スキーマが許す値と過不足なく一致していることを確かめる。
func TestEnumValuesMatchSchema(t *testing.T) {
	var schema struct {
		Defs map[string]struct {
			Enum []string `json:"enum"`
		} `json:"$defs"`
	}
	readJSON(t, "../schema/protocol.schema.json", &schema)

	cases := []struct {
		def  string
		has  func(string) bool
		want []string
	}{
		{"Role", func(s string) bool { return RoleFromString(s) != R_NONE }, nil},
		{"Team", func(s string) bool { return TeamFromString(s) != T_NONE }, nil},
		{"Status", func(s string) bool { return s == string(S_ALIVE) || s == string(S_DEAD) }, nil},
		{"Species", func(s string) bool { return s == string(S_HUMAN) || s == string(S_WEREWOLF) }, nil},
		{"Request", func(s string) bool { return RequestFromString(s).Type == s }, nil},
	}
	for _, c := range cases {
		def, ok := schema.Defs[c.def]
		if !ok {
			t.Errorf("スキーマに %s の定義がありません", c.def)
			continue
		}
		if len(def.Enum) == 0 {
			t.Errorf("スキーマの %s に列挙値がありません", c.def)
		}
		for _, value := range def.Enum {
			if !c.has(value) {
				t.Errorf("%s の値 %s がGo側に定義されていません", c.def, value)
			}
		}
	}
}

// wireパッケージが生成物であることの確認も兼ねて、送信される文字列型が列挙と同じ値を取れることを見る。
func TestWireEnumTypesAreStrings(t *testing.T) {
	if got := string(wire.Role(R_SEER.Name)); got != "SEER" {
		t.Errorf("wire.Roleへの変換が壊れています: %s", got)
	}
	if got := string(wire.Status(S_ALIVE)); got != "ALIVE" {
		t.Errorf("wire.Statusへの変換が壊れています: %s", got)
	}
}

func readJSON(t *testing.T, path string, v any) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%s を読み込めません: %v", path, err)
	}
	if err := json.Unmarshal(data, v); err != nil {
		t.Fatalf("%s を解析できません: %v", path, err)
	}
}
