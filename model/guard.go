package model

// 護衛はサーバ内部の状態としてのみ保持し、エージェントへは送信しない。
type Guard struct {
	Day    int
	Agent  Agent
	Target Agent
}
