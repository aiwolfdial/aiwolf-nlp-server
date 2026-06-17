package model

import (
	"encoding/json"
	"errors"
)

type Role struct {
	Name    string
	Team    Team
	Species Species
}

var (
	R_WEREWOLF  = Role{Name: "WEREWOLF", Team: T_WEREWOLF, Species: S_WEREWOLF}
	R_POSSESSED = Role{Name: "POSSESSED", Team: T_WEREWOLF, Species: S_HUMAN}
	R_SEER      = Role{Name: "SEER", Team: T_VILLAGER, Species: S_HUMAN}
	R_BODYGUARD = Role{Name: "BODYGUARD", Team: T_VILLAGER, Species: S_HUMAN}
	R_VILLAGER  = Role{Name: "VILLAGER", Team: T_VILLAGER, Species: S_HUMAN}
	R_MEDIUM    = Role{Name: "MEDIUM", Team: T_VILLAGER, Species: S_HUMAN}
	R_NONE      = Role{Name: "NONE", Team: T_NONE, Species: S_NONE}
)

type Team string

const (
	T_VILLAGER Team = "VILLAGER"
	T_WEREWOLF Team = "WEREWOLF"
	T_NONE     Team = "NONE"
)

func TeamFromString(s string) Team {
	switch s {
	case "VILLAGER":
		return T_VILLAGER
	case "WEREWOLF":
		return T_WEREWOLF
	}
	return T_NONE
}

type Species string

const (
	S_HUMAN    Species = "HUMAN"
	S_WEREWOLF Species = "WEREWOLF"
	S_NONE     Species = "NONE"
)

func (r Role) String() string {
	return r.Name
}

func (r Role) MarshalJSON() ([]byte, error) {
	return json.Marshal(r.String())
}

func RoleFromString(s string) Role {
	switch s {
	case "WEREWOLF":
		return R_WEREWOLF
	case "POSSESSED":
		return R_POSSESSED
	case "SEER":
		return R_SEER
	case "BODYGUARD":
		return R_BODYGUARD
	case "VILLAGER":
		return R_VILLAGER
	case "MEDIUM":
		return R_MEDIUM
	}
	return R_NONE
}

func RolesFromConfig(config Config) (map[Role]int, error) {
	roleNumMap := make(map[Role]int)
	if roles, ok := config.Logic.Roles[config.Game.AgentCount]; ok {
		for roleName, num := range roles {
			role := RoleFromString(roleName)
			if role == R_NONE {
				return nil, errors.New("不明な役職名があります")
			}
			roleNumMap[role] = num
		}
	} else {
		return nil, errors.New("対応する役職の人数がありません")
	}
	return roleNumMap, nil
}
