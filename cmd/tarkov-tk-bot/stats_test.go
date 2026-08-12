package main

import (
	"testing"

	"github.com/kyleshepherd/discord-tk-bot/internal/storage"
)

func TestBuildCombinedPlayerStats(t *testing.T) {
	kills := []*storage.Kill{
		{Killer: "alpha", Victim: "bravo"},
		{Killer: "alpha", Victim: "charlie"},
		{Killer: "bravo", Victim: "alpha"},
	}
	disappointments := []*storage.Disappointment{
		{Responsible: "alpha", Victim: "bravo"},
		{Responsible: "charlie", Victim: "alpha"},
	}

	stats := buildCombinedPlayerStats(kills, disappointments)

	assertCombinedPlayerStats(t, stats["alpha"], 2, 1, 1, 1)
	assertCombinedPlayerStats(t, stats["bravo"], 1, 1, 0, 1)
	assertCombinedPlayerStats(t, stats["charlie"], 0, 1, 1, 0)
}

func assertCombinedPlayerStats(t *testing.T, stats *CombinedPlayerStats, teamKills int, teamDeaths int, disappointments int, disappointmentsReceived int) {
	t.Helper()
	if stats == nil {
		t.Fatal("expected player stats")
	}
	if stats.TeamKills != teamKills {
		t.Fatalf("expected %d team kills, got %d", teamKills, stats.TeamKills)
	}
	if stats.TeamDeaths != teamDeaths {
		t.Fatalf("expected %d team deaths, got %d", teamDeaths, stats.TeamDeaths)
	}
	if stats.Disappointments != disappointments {
		t.Fatalf("expected %d disappointments, got %d", disappointments, stats.Disappointments)
	}
	if stats.DisappointmentsReceived != disappointmentsReceived {
		t.Fatalf("expected %d disappointments received, got %d", disappointmentsReceived, stats.DisappointmentsReceived)
	}
}
