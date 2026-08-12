package main

import (
	"strings"
	"testing"
)

func TestLeaderboardRank(t *testing.T) {
	tests := []struct {
		index    int
		expected string
	}{
		{index: 0, expected: "🥇"},
		{index: 1, expected: "🥈"},
		{index: 2, expected: "🥉"},
		{index: 3, expected: "`4.`"},
	}

	for _, test := range tests {
		if actual := leaderboardRank(test.index); actual != test.expected {
			t.Fatalf("expected rank %q for index %d, got %q", test.expected, test.index, actual)
		}
	}
}

func TestBuildStatEmbedPagesKeepsShortLinesTogether(t *testing.T) {
	pages := buildStatEmbedPages([]string{"first", "second"})

	if len(pages) != 1 {
		t.Fatalf("expected one page, got %d", len(pages))
	}
	if pages[0] != "first\n\nsecond" {
		t.Fatalf("unexpected page content: %q", pages[0])
	}
}

func TestBuildStatEmbedPagesSplitsLongContent(t *testing.T) {
	longLine := strings.Repeat("x", statEmbedDescriptionLimit-10)
	secondLine := strings.Repeat("y", 20)
	pages := buildStatEmbedPages([]string{longLine, secondLine})

	if len(pages) != 2 {
		t.Fatalf("expected two pages, got %d", len(pages))
	}
	if pages[0] != longLine {
		t.Fatal("expected the long line to remain on the first page")
	}
	if pages[1] != secondLine {
		t.Fatalf("unexpected second page content: %q", pages[1])
	}
}
