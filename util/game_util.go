package util

import (
	"maps"
	"math/rand/v2"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/aiwolfdial/aiwolf-nlp-server/model"
)

func CountAliveTeams(statusMap map[model.Agent]model.Status) (int, int) {
	var humans, werewolfs int
	for agent, status := range statusMap {
		if status == model.S_ALIVE {
			switch agent.Role.Species {
			case model.S_HUMAN:
				humans++
			case model.S_WEREWOLF:
				werewolfs++
			}
		}
	}
	return humans, werewolfs
}

func CalcWinSideTeam(statusMap map[model.Agent]model.Status) model.Team {
	humans, werewolfs := CountAliveTeams(statusMap)
	if humans <= werewolfs {
		return model.T_WEREWOLF
	}
	if werewolfs == 0 {
		return model.T_VILLAGER
	}
	return model.T_NONE
}

func CalcHasErrorAgents(agents []*model.Agent) int {
	var count int
	for _, a := range agents {
		if a.HasError {
			count++
		}
	}
	return count
}

func GetRoleMap(agents []*model.Agent) map[model.Agent]model.Role {
	roleMap := make(map[model.Agent]model.Role)
	for _, a := range agents {
		roleMap[*a] = a.Role
	}
	return roleMap
}

func CreateAgents(conns []model.Connection, roles map[model.Role]int) []*model.Agent {
	rolesCopy := make(map[model.Role]int)
	maps.Copy(rolesCopy, roles)
	agents := make([]*model.Agent, 0)
	for i, conn := range conns {
		role := assignRole(rolesCopy)
		agent := model.NewAgent(i+1, role, conn)
		agents = append(agents, agent)
	}
	return agents
}

func CreateAgentsWithProfiles(conns []model.Connection, roles map[model.Role]int, profiles []model.Profile, encoding map[string]string) []*model.Agent {
	rolesCopy := make(map[model.Role]int)
	maps.Copy(rolesCopy, roles)
	agents := make([]*model.Agent, 0)

	// Shuffle a copy so we never mutate the caller's (shared) profiles slice;
	// concurrent games would otherwise corrupt each other's profile assignment.
	profilesCopy := slices.Clone(profiles)
	rand.Shuffle(len(profilesCopy), func(i, j int) { profilesCopy[i], profilesCopy[j] = profilesCopy[j], profilesCopy[i] })

	for i, conn := range conns {
		role := assignRole(rolesCopy)
		agent := model.NewAgentWithProfile(i+1, role, conn, profilesCopy[i], encoding)
		agents = append(agents, agent)
	}
	return agents
}

func CreateAgentsWithRole(roleMapConns map[model.Role][]model.Connection) []*model.Agent {
	agents := make([]*model.Agent, 0)
	i := 0
	for role, conns := range roleMapConns {
		for _, conn := range conns {
			agent := model.NewAgent(i+1, role, conn)
			i++
			agents = append(agents, agent)
		}
	}
	return agents
}

func CreateAgentsWithRoleAndProfile(roleMapConns map[model.Role][]model.Connection, profiles []model.Profile, encoding map[string]string) []*model.Agent {
	agents := make([]*model.Agent, 0)

	// Shuffle a copy so we never mutate the caller's (shared) profiles slice.
	profilesCopy := slices.Clone(profiles)
	rand.Shuffle(len(profilesCopy), func(i, j int) { profilesCopy[i], profilesCopy[j] = profilesCopy[j], profilesCopy[i] })

	i := 0
	for role, conns := range roleMapConns {
		for _, conn := range conns {
			profile := profilesCopy[i]
			agent := model.NewAgentWithProfile(i+1, role, conn, profile, encoding)
			i++
			agents = append(agents, agent)
		}
	}
	return agents
}

func assignRole(roles map[model.Role]int) model.Role {
	for r, n := range roles {
		if n > 0 {
			roles[r]--
			return r
		}
	}
	return model.R_VILLAGER
}

func GetCandidates(votes []model.Vote, condition func(model.Vote) bool) []model.Agent {
	counter := make(map[model.Agent]int)
	for _, vote := range votes {
		if condition(vote) {
			counter[vote.Target]++
		}
	}
	return getMaxCountCandidates(counter)
}

func getMaxCountCandidates(counter map[model.Agent]int) []model.Agent {
	var max int
	for _, count := range counter {
		if count > max {
			max = count
		}
	}
	candidates := make([]model.Agent, 0)
	for agent, count := range counter {
		if count == max {
			candidates = append(candidates, agent)
		}
	}
	return candidates
}

func GetRoleTeamNamesMap(agents []*model.Agent) map[model.Role][]string {
	roleTeamNamesMap := make(map[model.Role][]string)
	for _, a := range agents {
		roleTeamNamesMap[a.Role] = append(roleTeamNamesMap[a.Role], a.TeamName)
	}
	return roleTeamNamesMap
}

func CountLength(text string, inWord bool, countSpaces bool) int {
	if inWord {
		return len(strings.Fields(text))
	}
	if countSpaces {
		return utf8.RuneCountInString(text)
	}

	words := strings.Fields(text)
	return utf8.RuneCountInString(strings.Join(words, ""))
}

func TrimLength(text string, length int, inWord bool, countSpaces bool) string {
	if inWord {
		return trimByWords(text, length)
	}

	currentLength := CountLength(text, inWord, countSpaces)
	if currentLength <= length {
		return text
	}

	if countSpaces {
		return trimByRunes(text, length)
	}

	return trimByNonSpaceCount(text, length)
}

func trimByWords(text string, maxWords int) string {
	words := strings.Fields(text)
	if len(words) <= maxWords {
		return text
	}
	return strings.Join(words[:maxWords], " ")
}

func trimByRunes(text string, maxRunes int) string {
	runes := []rune(text)
	if len(runes) <= maxRunes {
		return text
	}
	return string(runes[:maxRunes])
}

func trimByNonSpaceCount(text string, maxNonSpaceChars int) string {
	runes := []rune(text)
	nonSpaceCount := 0

	for i, r := range runes {
		if !unicode.IsSpace(r) {
			nonSpaceCount++
			if nonSpaceCount == maxNonSpaceChars {
				return string(runes[:i+1])
			}
		}
	}

	return text
}
