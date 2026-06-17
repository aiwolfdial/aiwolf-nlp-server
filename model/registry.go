package model

import (
	"errors"
	"path/filepath"
	"strings"
)

// RulesetSummary is a read-only summary of an available game configuration, for
// the management API / Web UI to enumerate selectable rulesets.
type RulesetSummary struct {
	Name       string `json:"name"`
	AgentCount int    `json:"agent_count"`
	MaxDay     int    `json:"max_day"`
	IsOptimize bool   `json:"is_optimize"`
	SelfMatch  bool   `json:"self_match"`
}

// RulesetRegistry enumerates the YAML game configurations under a directory. It
// is lazy: nothing is loaded until List or Get is called, so it never changes
// startup behaviour.
type RulesetRegistry struct {
	dir string
}

// NewRulesetRegistry returns a registry scanning the given directory.
func NewRulesetRegistry(dir string) *RulesetRegistry {
	return &RulesetRegistry{dir: dir}
}

// List returns a summary of every parseable *.yml config in the directory.
// Unparseable files are skipped rather than failing the whole listing.
func (r *RulesetRegistry) List() ([]RulesetSummary, error) {
	paths, err := filepath.Glob(filepath.Join(r.dir, "*.yml"))
	if err != nil {
		return nil, err
	}
	summaries := make([]RulesetSummary, 0, len(paths))
	for _, path := range paths {
		cfg, err := LoadFromPath(path)
		if err != nil {
			continue
		}
		name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		summaries = append(summaries, RulesetSummary{
			Name:       name,
			AgentCount: cfg.Game.AgentCount,
			MaxDay:     cfg.Game.MaxDay,
			IsOptimize: cfg.Matching.IsOptimize,
			SelfMatch:  cfg.Matching.SelfMatch,
		})
	}
	return summaries, nil
}

// Get loads the full config for a named ruleset. The name must be a bare file
// name (no path separators) to prevent directory traversal.
func (r *RulesetRegistry) Get(name string) (*Config, error) {
	if name == "" || name != filepath.Base(name) || strings.Contains(name, "..") {
		return nil, errors.New("不正なルールセット名です")
	}
	return LoadFromPath(filepath.Join(r.dir, name+".yml"))
}
