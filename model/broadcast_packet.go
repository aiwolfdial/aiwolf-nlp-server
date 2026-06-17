package model

type BroadcastAgent struct {
	Idx     int     `json:"idx"`
	Team    string  `json:"team"`
	Name    string  `json:"name"`
	Profile *string `json:"profile,omitempty"`
	Avatar  *string `json:"avatar,omitempty"`
	Role    string  `json:"role"`
	IsAlive bool    `json:"is_alive"`
}

type BroadcastPacket struct {
	Id        string           `json:"id"`
	Idx       int              `json:"idx"`
	Day       int              `json:"day"`
	IsDay     bool             `json:"is_day"`
	Agents    []BroadcastAgent `json:"agents"`
	Event     string           `json:"event"`
	Message   *string          `json:"message,omitempty"`
	FromIdx   *int             `json:"from_idx,omitempty"`
	ToIdx     *int             `json:"to_idx,omitempty"`
	BubbleIdx *int             `json:"bubble_idx,omitempty"`
	Timestamp int64            `json:"timestamp"`
}

// GameState は1イベント時点のゲーム状態のスナップショット。realtime sink が
// これからブロードキャストパケットを組み立てる（名簿・生存状況・日付など）。
type GameState struct {
	Day       int
	IsDaytime bool
	Agents    []BroadcastAgent
}

// AgentStatus は .log の status 行に必要なエージェント単位の情報。
type AgentStatus struct {
	Idx          int
	Role         string
	Status       string
	OriginalName string
	GameName     string
}
