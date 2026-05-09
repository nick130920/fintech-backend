package entity

import (
	"math"
	"testing"
)

func TestRecalculateSharesEqual(t *testing.T) {
	splits := []*ExpenseSplit{
		{ShareType: ExpenseSplitShareTypeEqual},
		{ShareType: ExpenseSplitShareTypeEqual},
		{ShareType: ExpenseSplitShareTypeEqual},
	}

	if err := RecalculateShares(splits, 30); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, s := range splits {
		if s.ShareAmount != 10 {
			t.Fatalf("expected 10 per split, got %.2f", s.ShareAmount)
		}
	}
}

func TestRecalculateSharesPercentage(t *testing.T) {
	splits := []*ExpenseSplit{
		{ShareType: ExpenseSplitShareTypePercentage, ShareValue: 50},
		{ShareType: ExpenseSplitShareTypePercentage, ShareValue: 30},
		{ShareType: ExpenseSplitShareTypePercentage, ShareValue: 20},
	}

	if err := RecalculateShares(splits, 200); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []float64{100, 60, 40}
	for i, s := range splits {
		if math.Abs(s.ShareAmount-expected[i]) > 0.01 {
			t.Fatalf("split %d: expected %.2f, got %.2f", i, expected[i], s.ShareAmount)
		}
	}
}

func TestRecalculateSharesPercentageInvalid(t *testing.T) {
	splits := []*ExpenseSplit{
		{ShareType: ExpenseSplitShareTypePercentage, ShareValue: 50},
		{ShareType: ExpenseSplitShareTypePercentage, ShareValue: 30},
	}

	if err := RecalculateShares(splits, 100); err == nil {
		t.Fatal("expected error when percentages dont sum to 100")
	}
}

func TestRecalculateSharesShares(t *testing.T) {
	splits := []*ExpenseSplit{
		{ShareType: ExpenseSplitShareTypeShares, ShareValue: 1},
		{ShareType: ExpenseSplitShareTypeShares, ShareValue: 2},
		{ShareType: ExpenseSplitShareTypeShares, ShareValue: 1},
	}

	if err := RecalculateShares(splits, 100); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []float64{25, 50, 25}
	for i, s := range splits {
		if math.Abs(s.ShareAmount-expected[i]) > 0.01 {
			t.Fatalf("split %d: expected %.2f, got %.2f", i, expected[i], s.ShareAmount)
		}
	}
}

func TestRecalculateSharesExact(t *testing.T) {
	splits := []*ExpenseSplit{
		{ShareType: ExpenseSplitShareTypeExact, ShareValue: 30},
		{ShareType: ExpenseSplitShareTypeExact, ShareValue: 70},
	}

	if err := RecalculateShares(splits, 100); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if splits[0].ShareAmount != 30 {
		t.Fatalf("expected 30, got %.2f", splits[0].ShareAmount)
	}
	if splits[1].ShareAmount != 70 {
		t.Fatalf("expected 70, got %.2f", splits[1].ShareAmount)
	}
}

func TestRecalculateSharesExactMismatch(t *testing.T) {
	splits := []*ExpenseSplit{
		{ShareType: ExpenseSplitShareTypeExact, ShareValue: 30},
		{ShareType: ExpenseSplitShareTypeExact, ShareValue: 50},
	}

	if err := RecalculateShares(splits, 100); err == nil {
		t.Fatal("expected error when exact sums dont match total")
	}
}

func TestRecalculateSharesMixedTypesFails(t *testing.T) {
	splits := []*ExpenseSplit{
		{ShareType: ExpenseSplitShareTypeEqual},
		{ShareType: ExpenseSplitShareTypePercentage, ShareValue: 100},
	}
	if err := RecalculateShares(splits, 50); err == nil {
		t.Fatal("expected error for mixed share types")
	}
}

func TestExpenseSplitMarkPaid(t *testing.T) {
	split := &ExpenseSplit{}
	split.MarkPaid()
	if !split.IsPaid {
		t.Fatal("expected IsPaid true")
	}
	if split.PaidAt == nil {
		t.Fatal("expected PaidAt set")
	}

	split.MarkUnpaid()
	if split.IsPaid {
		t.Fatal("expected IsPaid false")
	}
	if split.PaidAt != nil {
		t.Fatal("expected PaidAt cleared")
	}
}
