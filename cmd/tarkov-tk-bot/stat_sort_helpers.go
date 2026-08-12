package main

import (
	"sort"

	"github.com/kyleshepherd/discord-tk-bot/internal/storage"
)

func sortKillsHighestToLowest(kills []*storage.Kill) {
	counts := make(map[string]int)
	for _, kill := range kills {
		counts[kill.Killer]++
	}

	sort.SliceStable(kills, func(i, j int) bool {
		leftCount := counts[kills[i].Killer]
		rightCount := counts[kills[j].Killer]
		if leftCount != rightCount {
			return leftCount > rightCount
		}
		if !kills[i].Date.Equal(kills[j].Date) {
			return kills[i].Date.After(kills[j].Date)
		}
		if kills[i].Killer != kills[j].Killer {
			return kills[i].Killer < kills[j].Killer
		}
		return kills[i].Victim < kills[j].Victim
	})
}
