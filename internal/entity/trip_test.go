package entity

import (
	"testing"
	"time"
)

func TestTripDaysTotal(t *testing.T) {
	tests := []struct {
		name  string
		start time.Time
		end   time.Time
		want  int
	}{
		{
			name:  "single day trip",
			start: time.Date(2026, time.July, 10, 0, 0, 0, 0, time.UTC),
			end:   time.Date(2026, time.July, 10, 0, 0, 0, 0, time.UTC),
			want:  1,
		},
		{
			name:  "five day trip",
			start: time.Date(2026, time.July, 10, 0, 0, 0, 0, time.UTC),
			end:   time.Date(2026, time.July, 14, 0, 0, 0, 0, time.UTC),
			want:  5,
		},
		{
			name:  "end before start returns one",
			start: time.Date(2026, time.July, 14, 0, 0, 0, 0, time.UTC),
			end:   time.Date(2026, time.July, 10, 0, 0, 0, 0, time.UTC),
			want:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trip := &Trip{StartDate: tt.start, EndDate: tt.end}
			if got := trip.DaysTotal(); got != tt.want {
				t.Fatalf("DaysTotal() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestTripProgressAndRemaining(t *testing.T) {
	trip := &Trip{EstimatedTotal: 1000, SpentTotal: 250}

	if got := trip.ProgressPercentage(); got != 25 {
		t.Fatalf("ProgressPercentage() = %.2f, want 25", got)
	}
	if got := trip.RemainingAmount(); got != 750 {
		t.Fatalf("RemainingAmount() = %.2f, want 750", got)
	}
	if trip.IsOverBudget() {
		t.Fatal("IsOverBudget() = true, want false")
	}

	trip.SpentTotal = 1500
	if !trip.IsOverBudget() {
		t.Fatal("IsOverBudget() = false, want true")
	}
}

func TestTripCanTransitions(t *testing.T) {
	planning := &Trip{Status: TripStatusPlanning}
	active := &Trip{Status: TripStatusActive}
	completed := &Trip{Status: TripStatusCompleted}
	cancelled := &Trip{Status: TripStatusCancelled}

	if !planning.CanBeStarted() {
		t.Fatal("planning trip should be startable")
	}
	if active.CanBeStarted() {
		t.Fatal("active trip should not be startable again")
	}

	if !active.CanBeCompleted() {
		t.Fatal("active trip should be completable")
	}
	if completed.CanBeCompleted() {
		t.Fatal("completed trip should not be completable again")
	}

	if !planning.CanBeCancelled() {
		t.Fatal("planning trip should be cancellable")
	}
	if cancelled.CanBeCancelled() {
		t.Fatal("cancelled trip should not be cancellable")
	}
}

func TestTripIsActiveNow(t *testing.T) {
	now := time.Now()
	cases := []struct {
		name    string
		trip    *Trip
		isAlive bool
	}{
		{
			name: "active in range",
			trip: &Trip{
				Status:    TripStatusActive,
				StartDate: now.Add(-24 * time.Hour),
				EndDate:   now.Add(24 * time.Hour),
			},
			isAlive: true,
		},
		{
			name: "active before start",
			trip: &Trip{
				Status:    TripStatusActive,
				StartDate: now.Add(48 * time.Hour),
				EndDate:   now.Add(72 * time.Hour),
			},
			isAlive: false,
		},
		{
			name: "planning",
			trip: &Trip{
				Status:    TripStatusPlanning,
				StartDate: now.Add(-24 * time.Hour),
				EndDate:   now.Add(24 * time.Hour),
			},
			isAlive: false,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.trip.IsActiveNow(); got != tt.isAlive {
				t.Fatalf("IsActiveNow() = %v, want %v", got, tt.isAlive)
			}
		})
	}
}
