package model

import "testing"

func TestRulesetViewFreezesAndCopies(t *testing.T) {
	cfg := Config{}
	cfg.Game.MaxDay = 5
	cfg.Server.MaxContinueErrorRatio = 0.5
	cfg.Game.Vote.AllowSelfVote = true
	cfg.Logic.DayPhases = []Phase{{Name: "talk", Actions: []string{"talk"}}}

	view := NewRulesetView(cfg)

	cfg.Game.MaxDay = 99
	cfg.Logic.DayPhases[0].Name = "mutated"
	if view.MaxDay() != 5 {
		t.Fatalf("MaxDay: expected frozen value 5, got %d", view.MaxDay())
	}
	if got := view.DayPhases()[0].Name; got != "talk" {
		t.Fatalf("DayPhases: expected frozen value 'talk', got %q", got)
	}

	phases := view.DayPhases()
	phases[0].Name = "tampered"
	if got := view.DayPhases()[0].Name; got != "talk" {
		t.Fatalf("DayPhases getter returned a shared slice: got %q", got)
	}
}

func TestSettingViewIsIndependent(t *testing.T) {
	s := &Setting{}
	s.VoteVisibility = true
	s.Vote.MaxCount = 3

	view := NewSettingView(s)

	s.Vote.MaxCount = 99
	s.VoteVisibility = false
	if view.VoteMaxCount() != 3 {
		t.Fatalf("VoteMaxCount: expected frozen value 3, got %d", view.VoteMaxCount())
	}
	if !view.VoteVisibility() {
		t.Fatalf("VoteVisibility: expected frozen value true, got false")
	}

	snap := view.Snapshot()
	snap.Vote.MaxCount = 7
	if view.VoteMaxCount() != 3 {
		t.Fatalf("Snapshot shared state with the view: got %d", view.VoteMaxCount())
	}
}

func TestAgentViewHasValueSemantics(t *testing.T) {
	a := &Agent{Idx: 2, TeamName: "team", OriginalName: "team1", GameName: "Agent[02]", Role: R_SEER}

	v := a.View()
	if v.Idx != 2 || v.GameName != "Agent[02]" || v.Role != R_SEER {
		t.Fatalf("View did not project fields: %+v", v)
	}

	v.Role = R_WEREWOLF
	if a.Role != R_SEER {
		t.Fatalf("mutating AgentView affected the source agent: %v", a.Role)
	}

	tv := Talk{Idx: 1, Agent: *a, Text: "hi"}.View()
	if tv.Agent.GameName != "Agent[02]" || tv.Text != "hi" {
		t.Fatalf("TalkView did not project fields: %+v", tv)
	}
}
