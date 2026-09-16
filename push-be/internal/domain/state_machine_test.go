package domain

import "testing"

func TestCanPatchStage(t *testing.T) {
	cases := []struct {
		from, to ApplicationStage
		want     bool
	}{
		{StageDiscovered, StagePreparing, true},
		{StagePreparing, StageReady, true},
		{StageReady, StageApplied, false},
		{StageApplied, StageScreening, true},
		{StageScreening, StageInterview, true},
		{StageInterview, StageOffer, true},
		{StageOffer, StageAccepted, true},
		{StageDiscovered, StageDiscovered, true},
		{StageDiscovered, StageInterview, false},
		{StageDiscovered, StageApplied, false},
		{StagePreparing, StageDiscovered, false},
		{StageDiscovered, StageRejected, true},
		{StageInterview, StageWithdrawn, true},
		{StageAccepted, StageRejected, false},
		{StageRejected, StageWithdrawn, false},
		{StageWithdrawn, StageDiscovered, false},
	}
	for _, c := range cases {
		if got := CanPatchStage(c.from, c.to); got != c.want {
			t.Errorf("CanPatchStage(%s, %s) = %v, want %v", c.from, c.to, got, c.want)
		}
	}
}

func TestValidStage(t *testing.T) {
	for _, s := range []ApplicationStage{
		StageDiscovered, StagePreparing, StageReady, StageApplied, StageScreening,
		StageInterview, StageOffer, StageAccepted, StageRejected, StageWithdrawn,
	} {
		if !ValidStage(s) {
			t.Errorf("ValidStage(%s) = false", s)
		}
	}
	if ValidStage("BOGUS") {
		t.Error("ValidStage(BOGUS) = true")
	}
}

func TestTerminalStage(t *testing.T) {
	for _, s := range []ApplicationStage{StageAccepted, StageRejected, StageWithdrawn} {
		if !TerminalStage(s) {
			t.Errorf("TerminalStage(%s) = false", s)
		}
	}
	if TerminalStage(StageInterview) {
		t.Error("TerminalStage(INTERVIEW) = true")
	}
}

func TestNextStage(t *testing.T) {
	if next, ok := NextStage(StageOffer); !ok || next != StageAccepted {
		t.Errorf("NextStage(OFFER) = %s, %v", next, ok)
	}
	if _, ok := NextStage(StageAccepted); ok {
		t.Error("NextStage(ACCEPTED) should fail")
	}
	if _, ok := NextStage(StageRejected); ok {
		t.Error("NextStage(REJECTED) should fail")
	}
}

func TestCanTransitionLaunch(t *testing.T) {
	cases := []struct {
		from, to LaunchStatus
		want     bool
	}{
		{LaunchNotClaimed, LaunchClaimed, true},
		{LaunchNotClaimed, LaunchStarted, false},
		{LaunchClaimed, LaunchStarted, true},
		{LaunchClaimed, LaunchUnknown, true},
		{LaunchStarted, LaunchUnknown, true},
		{LaunchStarted, LaunchFinished, false},
		{LaunchUnknown, LaunchStarted, false},
		{LaunchFinished, LaunchFinished, true},
	}
	for _, c := range cases {
		if got := CanTransitionLaunch(c.from, c.to); got != c.want {
			t.Errorf("CanTransitionLaunch(%s, %s) = %v, want %v", c.from, c.to, got, c.want)
		}
	}
}

func TestValidReportedLaunchStatus(t *testing.T) {
	for _, s := range []LaunchStatus{LaunchClaimed, LaunchStarted, LaunchUnknown} {
		if !ValidReportedLaunchStatus(s) {
			t.Errorf("ValidReportedLaunchStatus(%s) = false", s)
		}
	}
	for _, s := range []LaunchStatus{LaunchNotClaimed, LaunchFinished} {
		if ValidReportedLaunchStatus(s) {
			t.Errorf("ValidReportedLaunchStatus(%s) = true", s)
		}
	}
}
