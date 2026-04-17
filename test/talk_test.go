package test

import (
	"sync"
	"testing"

	"github.com/aiwolfdial/aiwolf-nlp-server/model"
	"github.com/stretchr/testify/assert"
)

func TestTalkPhase1(t *testing.T) {
	t.Log("トークフェーズ")
	config, err := model.LoadFromPath("./config/talk.yml")
	if err != nil {
		t.Fatalf("設定ファイルの読み込みに失敗しました: %v", err)
	}

	sendMessagesMap := map[string][]string{
		"WEREWOLF":   {"Hello World!"},
		"POSSESSED":  {"Hello World!"},
		"SEER":       {"Hello World!"},
		"VILLAGER-A": {"Hello World!"},
		"VILLAGER-B": {"Hello World!"},
	}
	executeTalkPhase(t, sendMessagesMap, config)
}

func executeTalkPhase(t *testing.T, sendMessagesMap map[string][]string, config *model.Config) {
	var nameMu sync.Mutex
	var talkMu sync.Mutex

	nameMap := make(map[string]string)

	messageIdxMap := make(map[string]int)

	var idx float64 = 0
	expectTalks := []any{}

	handlers := map[model.Request]func(tc TestClient) (string, error){
		model.R_INITIALIZE: func(tc TestClient) (string, error) {
			nameMu.Lock()
			defer nameMu.Unlock()
			nameMap[tc.originalName] = tc.gameName
			return "", nil
		},
		model.R_TALK: func(tc TestClient) (string, error) {
			talkMu.Lock()
			defer talkMu.Unlock()
			assert.Equal(t, len(expectTalks), len(tc.talkHistory))
			if len(expectTalks) > 0 {
				stripped := make([]any, len(tc.talkHistory))
				for i, entry := range tc.talkHistory {
					if m, ok := entry.(map[string]any); ok {
						copied := make(map[string]any, len(m))
						for k, v := range m {
							if k == "time" {
								continue
							}
							copied[k] = v
						}
						stripped[i] = copied
					} else {
						stripped[i] = entry
					}
				}
				assert.Equal(t, expectTalks, stripped)
			}

			messageIdx := messageIdxMap[tc.originalName]
			messageIdxMap[tc.originalName]++
			message := model.T_OVER
			if messageIdx < len(sendMessagesMap[tc.originalName]) {
				message = sendMessagesMap[tc.originalName][messageIdx]
			}
			tc.t.Logf("トーク: %s < %s", tc.gameName, message)

			expectTalks = append(expectTalks, map[string]any{
				"idx":   idx,
				"day":   tc.info["day"].(float64),
				"turn":  float64(messageIdx),
				"agent": tc.gameName,
				"text":  message,
				"skip":  message == model.T_SKIP || message == model.T_FORCE_SKIP,
				"over":  message == model.T_OVER,
			})
			idx++
			return message, nil
		},
		model.R_DAILY_FINISH: func(tc TestClient) (string, error) {
			return "", nil
		},
	}
	executeGame(t, []string{"WEREWOLF", "POSSESSED", "SEER", "VILLAGER-A", "VILLAGER-B"}, config, handlers)
}
