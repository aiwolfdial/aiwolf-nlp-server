package model

// RulesetInfo summarises the single ruleset this server process is running.
// The server runs exactly one config per process, so this reports the active
// rules rather than enumerating configs on disk.
type RulesetInfo struct {
	AgentCount     int            `json:"agent_count"`
	MaxDay         int            `json:"max_day"`
	VoteVisibility bool           `json:"vote_visibility"`
	IsOptimize     bool           `json:"is_optimize"`
	SelfMatch      bool           `json:"self_match"`
	Roles          map[string]int `json:"roles"`
}

// RulesetInfo builds a read-only summary of the active ruleset from the config.
func (c Config) RulesetInfo() RulesetInfo {
	info := RulesetInfo{
		AgentCount:     c.Game.AgentCount,
		MaxDay:         c.Game.MaxDay,
		VoteVisibility: c.Game.VoteVisibility,
		IsOptimize:     c.Matching.IsOptimize,
		SelfMatch:      c.Matching.SelfMatch,
		Roles:          make(map[string]int),
	}
	if roles, err := RolesFromConfig(c); err == nil {
		for role, num := range roles {
			info.Roles[role.String()] = num
		}
	}
	return info
}
