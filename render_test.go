package main

import (
	"strings"
	"testing"
	"time"
)

func TestViewRenders(t *testing.T) {
	m := initialModel()
	m.width = 100
	m.height = 30
	v := m.View()
	if v.Content == "" {
		t.Fatal("empty view content")
	}
	if !strings.Contains(v.Content, "FOCUS") {
		t.Error("missing FOCUS phase label")
	}
	if !strings.Contains(v.Content, "Campbell") {
		t.Error("missing soup-can label")
	}
}

func TestPhaseAdvance(t *testing.T) {
	m := initialModel()
	for i := 0; i < m.cfg.PomodorosBeforeLong; i++ {
		if m.phase != phaseWork {
			t.Fatalf("expected work phase, got %v", m.phase)
		}
		m.advancePhase(true)
		if i < m.cfg.PomodorosBeforeLong-1 {
			if m.phase != phaseShortBreak {
				t.Fatalf("expected short break, got %v", m.phase)
			}
		} else {
			if m.phase != phaseLongBreak {
				t.Fatalf("expected long break after %d pomodoros, got %v", m.cfg.PomodorosBeforeLong, m.phase)
			}
		}
		m.advancePhase(false)
	}
}

func TestFormatDuration(t *testing.T) {
	cases := map[time.Duration]string{
		25 * time.Minute: "25:00",
		61 * time.Second: "01:01",
		0:                "00:00",
	}
	for d, want := range cases {
		if got := formatDuration(d); got != want {
			t.Errorf("formatDuration(%v) = %q, want %q", d, got, want)
		}
	}
}
