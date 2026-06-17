package model

import "time"

// AgentView is a read-only, value-type projection of an Agent. It exposes only
// safe scalar fields and never the live websocket Connection or the message
// channel, so observers and API consumers cannot reach or mutate game state
// through it.
type AgentView struct {
	Idx          int    `json:"idx"`
	TeamName     string `json:"team_name"`
	OriginalName string `json:"original_name"`
	GameName     string `json:"game_name"`
	Role         Role   `json:"role"`
	Alive        bool   `json:"alive"`
}

func (a AgentView) String() string {
	return a.GameName
}

// View returns a read-only projection of the agent. Alive defaults to true;
// callers that know the agent's status set it explicitly.
func (a *Agent) View() AgentView {
	return AgentView{
		Idx:          a.Idx,
		TeamName:     a.TeamName,
		OriginalName: a.OriginalName,
		GameName:     a.GameName,
		Role:         a.Role,
		Alive:        true,
	}
}

// ViewsOf converts a slice of agents into read-only views.
func ViewsOf(agents []*Agent) []AgentView {
	views := make([]AgentView, 0, len(agents))
	for _, a := range agents {
		views = append(views, a.View())
	}
	return views
}

// GameSnapshot is a read-only, value-type summary of a game's state, handed out
// by the orchestration/API layer. All collections are copies, so consumers
// cannot reach or mutate live game state.
type GameSnapshot struct {
	ID            string         `json:"id"`
	Day           int            `json:"day"`
	Finished      bool           `json:"finished"`
	WinSide       Team           `json:"win_side"`
	Agents        []AgentView    `json:"agents"`
	StatusByAgent map[int]string `json:"status_by_agent"` // agent idx -> status (ALIVE/DEAD)
}

// TalkView is a read-only projection of a Talk. Unlike Talk, it does not embed a
// full Agent (which would leak the live Connection and message channel); it
// carries an AgentView instead.
type TalkView struct {
	Idx   int
	Day   int
	Turn  int
	Agent AgentView
	Text  string
	Time  time.Time
}

// View returns a read-only projection of the talk.
func (t Talk) View() TalkView {
	return TalkView{
		Idx:   t.Idx,
		Day:   t.Day,
		Turn:  t.Turn,
		Agent: t.Agent.View(),
		Text:  t.Text,
		Time:  t.Time,
	}
}
