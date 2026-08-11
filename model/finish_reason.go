package model

// ゲームが終了した理由。勝敗が付いたかどうかは winSide で分かるが、winSide は
// max_day 到達による引き分けでもエラーによる打ち切りでも T_NONE になる。
// 参加チームの責任を問える終わり方かどうかを区別するためにこれを持つ。
//
// エージェント単位の Agent.HasError とは粒度が異なる。HasError は「そのエージェントが
// 以降のリクエストを受け付けられない」状態を表し、F_ERROR は「壊れたエージェントが
// max_continue_error_ratio を超えたのでゲームそのものを打ち切った」結果を表す。
type FinishReason string

const (
	F_WIN     FinishReason = "WIN"
	F_MAX_DAY FinishReason = "MAX_DAY"
	F_ERROR   FinishReason = "ERROR"
)

func (r FinishReason) String() string {
	return string(r)
}
