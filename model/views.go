package model

import "time"

// observerやAPIへ渡す唯一のエージェント表現。生のConnection/msgChanを含めない値型とし、
// 外部から内部状態へ到達・変更できないようにする。
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

// Aliveはtrue既定。生存状況を知る呼び出し側が必要に応じて設定する。
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

func ViewsOf(agents []*Agent) []AgentView {
	views := make([]AgentView, 0, len(agents))
	for _, a := range agents {
		views = append(views, a.View())
	}
	return views
}

// GameManager/APIが返す値型のスナップショット。内部のmap/sliceは複製して渡す。
type GameSnapshot struct {
	ID            string         `json:"id"`
	Day           int            `json:"day"`
	Finished      bool           `json:"finished"`
	WinSide       Team           `json:"win_side"`
	Agents        []AgentView    `json:"agents"`
	StatusByAgent map[int]string `json:"status_by_agent"`
}

// TalkはAgentを値で内包しConnection/msgChanを露出するため、observerへはこのビューを渡す。
type TalkView struct {
	Idx   int
	Day   int
	Turn  int
	Agent AgentView
	Text  string
	Time  time.Time
}

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
