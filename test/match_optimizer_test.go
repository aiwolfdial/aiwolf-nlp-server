package test

import (
	"log/slog"
	"testing"

	"github.com/aiwolfdial/aiwolf-nlp-server/matchmaking"
	"github.com/aiwolfdial/aiwolf-nlp-server/model"
)

func TestInitializeMatchOptimizer(t *testing.T) {
	config, err := model.LoadFromPath("./config/optimize.yml")
	if err != nil {
		t.Fatalf("設定ファイルの読み込みに失敗しました: %v", err)
	}

	mo, err := matchmaking.NewMatchOptimizerFromConfig(*config)
	if err != nil {
		t.Fatalf("マッチオプティマイザの初期化に失敗しました: %v", err)
	}

	roleCounts := make(map[int]map[model.Role]int)
	for i := range mo.TeamCount {
		roleCounts[i] = make(map[model.Role]int)
	}
	for _, match := range mo.ScheduledMatches {
		for role, idxs := range match.RoleIdxs {
			for _, idx := range idxs {
				roleCounts[idx][role]++
			}
		}
	}
	t.Log(roleCounts)

	for i := range mo.TeamCount {
		slog.Info("team", "idx", i, "roles", roleCounts[i])
	}
}
