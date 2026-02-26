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
