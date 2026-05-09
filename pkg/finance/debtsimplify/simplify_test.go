package debtsimplify

import (
	"math"
	"testing"
)

func TestSimplify(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		balances  map[uint]float64
		wantCount int
	}{
		{
			name:      "empty balances",
			balances:  map[uint]float64{},
			wantCount: 0,
		},
		{
			name:      "all zero balances",
			balances:  map[uint]float64{1: 0, 2: 0, 3: 0},
			wantCount: 0,
		},
		{
			name:      "two members owe each other",
			balances:  map[uint]float64{1: 50, 2: -50},
			wantCount: 1,
		},
		{
			name:      "three members one creditor two debtors",
			balances:  map[uint]float64{1: 100, 2: -40, 3: -60},
			wantCount: 2,
		},
		{
			name:      "split chain three members",
			balances:  map[uint]float64{1: 100, 2: -50, 3: -50},
			wantCount: 2,
		},
		{
			name:      "no transfers when single zeroed member",
			balances:  map[uint]float64{1: 0},
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			transfers := Simplify(tt.balances)
			if len(transfers) != tt.wantCount {
				t.Fatalf("expected %d transfers, got %d (%+v)", tt.wantCount, len(transfers), transfers)
			}
			assertBalanced(t, tt.balances, transfers)
		})
	}
}

func TestSimplifyMinimizesTransfers(t *testing.T) {
	t.Parallel()
	balances := map[uint]float64{
		1: 100,
		2: -50,
		3: 50,
		4: -100,
	}
	transfers := Simplify(balances)
	// Greedy debe acotar a N-1 transferencias para 4 miembros con desequilibrio
	if len(transfers) > 3 {
		t.Fatalf("expected at most 3 transfers, got %d (%+v)", len(transfers), transfers)
	}
	assertBalanced(t, balances, transfers)
}

func TestSimplifyHandlesRounding(t *testing.T) {
	t.Parallel()
	balances := map[uint]float64{
		1: 33.33,
		2: 33.33,
		3: 33.34,
		4: -100,
	}
	transfers := Simplify(balances)
	if len(transfers) == 0 {
		t.Fatal("expected transfers when there are non-zero balances")
	}
	assertBalanced(t, balances, transfers)
}

func assertBalanced(t *testing.T, original map[uint]float64, transfers []Transfer) {
	t.Helper()

	resulting := make(map[uint]float64, len(original))
	for k, v := range original {
		resulting[k] = v
	}
	for _, tr := range transfers {
		if tr.Amount <= 0 {
			t.Fatalf("transfer amount must be positive, got %v", tr)
		}
		resulting[tr.From] += tr.Amount
		resulting[tr.To] -= tr.Amount
	}
	for member, balance := range resulting {
		if math.Abs(balance) > 0.05 {
			t.Fatalf("member %d not settled, residual %v", member, balance)
		}
	}
}
