package model

// Gameはこのインターフェース型で設定を保持する。getterのみのため外部から変更できない。
// Snapshotはエージェント送信用のコピーを返す。
type SettingView interface {
	VoteVisibility() bool
	VoteMaxCount() int
	AttackVoteMaxCount() int
	AttackVoteAllowNoTarget() bool
	TalkSetting() TalkSetting
	WhisperSetting() TalkSetting
	Snapshot() *Setting
}

type frozenSetting struct {
	s *Setting
}

// 各ゲームが自前のコピーを持つよう複製を取り込む。
func NewSettingView(setting *Setting) SettingView {
	cp := *setting
	return frozenSetting{s: &cp}
}

func (f frozenSetting) VoteVisibility() bool          { return f.s.VoteVisibility }
func (f frozenSetting) VoteMaxCount() int             { return f.s.Vote.MaxCount }
func (f frozenSetting) AttackVoteMaxCount() int       { return f.s.AttackVote.MaxCount }
func (f frozenSetting) AttackVoteAllowNoTarget() bool { return f.s.AttackVote.AllowNoTarget }
func (f frozenSetting) TalkSetting() TalkSetting      { return f.s.Talk.TalkSetting }
func (f frozenSetting) WhisperSetting() TalkSetting   { return f.s.Whisper.TalkSetting }

func (f frozenSetting) Snapshot() *Setting {
	cp := *f.s
	return &cp
}
