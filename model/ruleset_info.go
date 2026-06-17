package model

// このプロセスが実行中のルール。サーバは1プロセス1設定で動くため一覧ではなく単一を表す。
type RulesetInfo struct {
	AgentCount     int            `json:"agent_count"`
	MaxDay         int            `json:"max_day"`
	VoteVisibility bool           `json:"vote_visibility"`
	IsOptimize     bool           `json:"is_optimize"`
	SelfMatch      bool           `json:"self_match"`
	Roles          map[string]int `json:"roles"`
}

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
