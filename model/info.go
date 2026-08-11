package model

import (
	"encoding/json"

	"github.com/aiwolfdial/aiwolf-nlp-server/model/wire"
)

// TalkList・WhisperListはサーバ内部の集計用で、エージェントへは talk_history として別途送る。
type Info struct {
	GameID         string
	Day            int
	Agent          *Agent
	Profile        *string
	MediumResult   *Judge
	DivineResult   *Judge
	ExecutedAgent  *Agent
	AttackedAgent  *Agent
	VoteList       []Vote
	AttackVoteList []Vote
	TalkList       []Talk
	WhisperList    []Talk
	StatusMap      map[Agent]Status
	RoleMap        map[Agent]Role
	RemainCount    *int
	RemainLength   *int
	RemainSkip     *int
}

func (i Info) wire() wire.Info {
	statusMap := make(map[string]wire.Status, len(i.StatusMap))
	for agent, status := range i.StatusMap {
		statusMap[agent.String()] = wire.Status(status)
	}
	roleMap := make(map[string]wire.Role, len(i.RoleMap))
	for agent, role := range i.RoleMap {
		roleMap[agent.String()] = wire.Role(role.Name)
	}
	agent := ""
	if i.Agent != nil {
		agent = i.Agent.String()
	}
	return wire.Info{
		GameID:         i.GameID,
		Day:            i.Day,
		Agent:          agent,
		Profile:        i.Profile,
		MediumResult:   wireJudge(i.MediumResult),
		DivineResult:   wireJudge(i.DivineResult),
		ExecutedAgent:  wireAgentName(i.ExecutedAgent),
		AttackedAgent:  wireAgentName(i.AttackedAgent),
		VoteList:       wireVotes(i.VoteList),
		AttackVoteList: wireVotes(i.AttackVoteList),
		StatusMap:      statusMap,
		RoleMap:        roleMap,
		RemainCount:    i.RemainCount,
		RemainLength:   i.RemainLength,
		RemainSkip:     i.RemainSkip,
	}
}

func (i Info) MarshalJSON() ([]byte, error) {
	return json.Marshal(i.wire())
}

func wireAgentName(agent *Agent) *string {
	if agent == nil {
		return nil
	}
	name := agent.String()
	return &name
}
