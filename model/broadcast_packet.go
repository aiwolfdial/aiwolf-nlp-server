package model

type BroadcastPacket struct {
	Id     string `json:"id"`
	Idx    int    `json:"idx"`
	Day    int    `json:"day"`
	IsDay  bool   `json:"is_day"`
	Agents []struct {
		Idx     int     `json:"idx"`
		Team    string  `json:"team"`
		Name    string  `json:"name"`
		Profile *string `json:"profile,omitempty"`
		Avatar  *string `json:"avatar,omitempty"`
		Role    string  `json:"role"`
		IsAlive bool    `json:"is_alive"`
	} `json:"agents"`
	Event     string  `json:"event"`
	Message   *string `json:"message,omitempty"`
	FromIdx   *int    `json:"from_idx,omitempty"`
	ToIdx     *int    `json:"to_idx,omitempty"`
	BubbleIdx *int    `json:"bubble_idx,omitempty"`
	Timestamp int64   `json:"timestamp"`
}
