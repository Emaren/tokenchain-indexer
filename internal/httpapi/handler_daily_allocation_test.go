package httpapi

import (
	"strings"
	"testing"
)

func TestComputeDailyAllocationsDeterministicRemainder(t *testing.T) {
	out, err := computeDailyAllocations(10, []adminDailyAllocationScoreItem{
		{Denom: "factory/z/token", ActivityScore: 1},
		{Denom: "factory/a/token", ActivityScore: 1},
		{Denom: "factory/m/token", ActivityScore: 1},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(out) != 3 {
		t.Fatalf("expected 3 items, got %d", len(out))
	}

	// Sorted by denom, then remainder goes to earliest denom.
	if out[0].Denom != "factory/a/token" || out[0].BucketCAmount != 4 {
		t.Fatalf("unexpected first allocation: %+v", out[0])
	}
	if out[1].Denom != "factory/m/token" || out[1].BucketCAmount != 3 {
		t.Fatalf("unexpected second allocation: %+v", out[1])
	}
	if out[2].Denom != "factory/z/token" || out[2].BucketCAmount != 3 {
		t.Fatalf("unexpected third allocation: %+v", out[2])
	}
}

func TestComputeDailyAllocationsDuplicateDenom(t *testing.T) {
	_, err := computeDailyAllocations(100, []adminDailyAllocationScoreItem{
		{Denom: "factory/a/token", ActivityScore: 10},
		{Denom: "factory/a/token", ActivityScore: 20},
	})
	if err == nil || !strings.Contains(err.Error(), "duplicate denom") {
		t.Fatalf("expected duplicate denom error, got %v", err)
	}
}

func TestComputeDailyAllocationsTrimsDenom(t *testing.T) {
	out, err := computeDailyAllocations(100, []adminDailyAllocationScoreItem{
		{Denom: "  factory/a/token  ", ActivityScore: 100},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("expected 1 item, got %d", len(out))
	}
	if out[0].Denom != "factory/a/token" || out[0].BucketCAmount != 100 {
		t.Fatalf("unexpected allocation: %+v", out[0])
	}
}

func TestNormalizeRunDateRejectsInvalid(t *testing.T) {
	_, err := normalizeRunDate("2026/02/26")
	if err == nil {
		t.Fatal("expected invalid date error")
	}
}

func TestDeriveAutoScoreItems(t *testing.T) {
	items, err := deriveAutoScoreItems([]merchantRoutingItem{
		{Denom: "factory/a/token", MintedSupply: "0"},
		{Denom: "factory/b/token", MintedSupply: "12"},
		{Denom: "factory/a/token", MintedSupply: "99"},
	}, 5)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 unique items, got %d", len(items))
	}
	if items[0].Denom != "factory/a/token" || items[0].ActivityScore != 5 {
		t.Fatalf("unexpected first item: %+v", items[0])
	}
	if items[1].Denom != "factory/b/token" || items[1].ActivityScore != 12 {
		t.Fatalf("unexpected second item: %+v", items[1])
	}
}

func TestResolveAutoMaxTokens(t *testing.T) {
	if got := resolveAutoMaxTokens(0); got != 200 {
		t.Fatalf("expected default max 200, got %d", got)
	}
	if got := resolveAutoMaxTokens(800); got != 500 {
		t.Fatalf("expected hard cap 500, got %d", got)
	}
}
