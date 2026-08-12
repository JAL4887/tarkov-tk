package main

import (
	"testing"
	"time"

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

func TestSortCombinedPlayerStatsHighestToLowest(t *testing.T) {
	stats := []*CombinedPlayerStats{
		{PlayerID: "charlie", TeamKills: 1, TeamDeaths: 2, Disappointments: 0, DisappointmentsReceived: 0},
		{PlayerID: "bravo", TeamKills: 3, TeamDeaths: 1, Disappointments: 0, DisappointmentsReceived: 0},
		{PlayerID: "delta", TeamKills: 1, TeamDeaths: 1, Disappointments: 2, DisappointmentsReceived: 0},
		{PlayerID: "alpha", TeamKills: 3, TeamDeaths: 1, Disappointments: 1, DisappointmentsReceived: 0},
	}
	displayNames := map[string]string{
		"alpha":   "Alpha",
		"bravo":   "Bravo",
		"charlie": "Charlie",
		"delta":   "Delta",
	}

	sortCombinedPlayerStatsHighestToLowest(stats, displayNames)

	expected := []string{"alpha", "bravo", "delta", "charlie"}
	for i, playerID := range expected {
		if stats[i].PlayerID != playerID {
			t.Fatalf("expected player %s at position %d, got %s", playerID, i, stats[i].PlayerID)
		}
	}
}

func TestSortDisappointmentsHighestToLowest(t *testing.T) {
	older := time.Date(2026, 8, 10, 10, 0, 0, 0, time.UTC)
	newer := time.Date(2026, 8, 11, 10, 0, 0, 0, time.UTC)
	newest := time.Date(2026, 8, 12, 10, 0, 0, 0, time.UTC)
	disappointments := []*storage.Disappointment{
		{Responsible: "bravo", Victim: "alpha", Date: newest},
		{Responsible: "alpha", Victim: "bravo", Date: older},
		{Responsible: "alpha", Victim: "charlie", Date: newer},
	}

	sortDisappointmentsHighestToLowest(disappointments)

	if disappointments[0].Responsible != "alpha" || !disappointments[0].Date.Equal(newer) {
		t.Fatalf("expected alpha's newest disappointment first, got %s at %s", disappointments[0].Responsible, disappointments[0].Date)
	}
	if disappointments[1].Responsible != "alpha" || !disappointments[1].Date.Equal(older) {
		t.Fatalf("expected alpha's older disappointment second, got %s at %s", disappointments[1].Responsible, disappointments[1].Date)
	}
	if disappointments[2].Responsible != "bravo" {
		t.Fatalf("expected bravo's lower-count disappointment last, got %s", disappointments[2].Responsible)
	}
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
