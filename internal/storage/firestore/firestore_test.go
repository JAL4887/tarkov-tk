package firestore

import (
	"testing"
	"time"

	"github.com/kyleshepherd/discord-tk-bot/internal/storage"
)

func TestSortKillsNewestFirst(t *testing.T) {
	oldest := time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC)
	middle := time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)
	newest := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)

	kills := []*storage.Kill{
		{ID: "middle", Date: middle},
		{ID: "oldest", Date: oldest},
		{ID: "newest", Date: newest},
	}

	sortKillsNewestFirst(kills)

	if kills[0].ID != "newest" || kills[1].ID != "middle" || kills[2].ID != "oldest" {
		t.Fatalf("unexpected order: %s, %s, %s", kills[0].ID, kills[1].ID, kills[2].ID)
	}
}
