package model

// SettingView is a read-only projection of a game Setting. The engine holds the
// setting through this interface so no code can mutate or reassign the settings
// of an in-flight game. Snapshot returns a copy suitable for sending to agents.
type SettingView interface {
	VoteVisibility() bool
	VoteMaxCount() int
	AttackVoteMaxCount() int
	AttackVoteAllowNoTarget() bool
	TalkSetting() TalkSetting
	WhisperSetting() TalkSetting
	Snapshot() *Setting
}

// frozenSetting wraps a per-game copy of a Setting and exposes only read-only
// accessors. The wrapped Setting is never mutated after construction.
type frozenSetting struct {
	s *Setting
}

// NewSettingView freezes a per-game copy of the setting into a read-only view.
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

// Snapshot returns a copy of the setting for marshaling into an agent packet,
// so the engine never hands out a pointer to the setting it holds.
func (f frozenSetting) Snapshot() *Setting {
	cp := *f.s
	return &cp
}
