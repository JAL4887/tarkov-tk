package main

import (
	"testing"
	"time"

	"github.com/kyleshepherd/discord-tk-bot/internal/storage"
)

func TestSortKillsHighestToLowest(t *testing.T) {
	older := time.Date(2026, 8, 10, 10, 0, 0, 0, time.UTC)
	newer := time.Date(2026, 8, 11, 10, 0, 0, 0, time.UTC)
	newest := time.Date(2026, 8, 12, 10, 0, 0, 0, time.UTC)
	kills := []*storage.Kill{
		{Killer: "bravo", Victim: "alpha", Date: newest},
		{Killer: "alpha", Victim: "bravo", Date: older},
		{Killer: "alpha", Victim: "charlie", Date: newer},
	}

	sortKillsHighestToLowest(kills)

	if kills[0].Killer != "alpha" || !kills[0].Date.Equal(newer) {
		t.Fatalf("expected alpha's newest TK first, got %s at %s", kills[0].Killer, kills[0].Date)
	}
	if kills[1].Killer != "alpha" || !kills[1].Date.Equal(older) {
		t.Fatalf("expected alpha's older TK second, got %s at %s", kills[1].Killer, kills[1].Date)
	}
	if kills[2].Killer != "bravo" {
		t.Fatalf("expected bravo's lower-count TK last, got %s", kills[2].Killer)
	}
}
