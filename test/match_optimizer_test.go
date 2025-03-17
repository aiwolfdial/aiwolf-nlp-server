package test

import (
	"log/slog"
	"testing"

	"github.com/aiwolfdial/aiwolf-nlp-server/core"
	"github.com/aiwolfdial/aiwolf-nlp-server/model"
)

func TestInitializeMatchOptimizer(t *testing.T) {
	config, err := model.LoadFromPath("../config/debug.yml")
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	mo, err := core.NewMatchOptimizerFromConfig(*config)
	if err != nil {
		t.Fatalf("Failed to create MatchOptimizer: %v", err)
	}

	roleCounts := make(map[int]map[model.Role]int)
	for i := 0; i < mo.TeamCount; i++ {
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

	for i := 0; i < mo.TeamCount; i++ {
		slog.Info("team", "idx", i, "roles", roleCounts[i])
	}
}

func TestLoadMatchOptimizer(t *testing.T) {
	config, err := model.LoadFromPath("../config/debug.yml")
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	mo, err := core.NewMatchOptimizer(*config)
	if err != nil {
		t.Fatalf("Failed to create MatchOptimizer: %v", err)
	}
	t.Log(mo)
}
