package dashboard

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/clock"
)

func TestDailyQuizTrendWindow(t *testing.T) {
	now := time.Date(2026, time.November, 26, 16, 42, 0, 0, clock.Jakarta)

	tests := []struct {
		name     string
		period   string
		count    int
		wantFrom string
		wantLast string
		wantEnd  string
	}{
		{name: "7 days", period: TrendPeriod7Days, count: 7, wantFrom: "2026-11-20", wantLast: "2026-11-26", wantEnd: "2026-11-27"},
		{name: "30 days", period: TrendPeriod30Days, count: 30, wantFrom: "2026-10-28", wantLast: "2026-11-26", wantEnd: "2026-11-27"},
		{name: "12 weeks", period: TrendPeriod12Weeks, count: 12, wantFrom: "2026-09-07", wantLast: "2026-11-23", wantEnd: "2026-11-27"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			window, err := dailyQuizTrendWindow(tt.period, now)
			if err != nil {
				t.Fatal(err)
			}
			if len(window.buckets) != tt.count {
				t.Fatalf("got %d buckets, want %d", len(window.buckets), tt.count)
			}
			if got := window.from.Format(time.DateOnly); got != tt.wantFrom {
				t.Errorf("from = %s, want %s", got, tt.wantFrom)
			}
			if got := window.lastBucket.Format(time.DateOnly); got != tt.wantLast {
				t.Errorf("last bucket = %s, want %s", got, tt.wantLast)
			}
			if got := window.to.Format(time.DateOnly); got != tt.wantEnd {
				t.Errorf("exclusive end = %s, want %s", got, tt.wantEnd)
			}
		})
	}
}

func TestDailyQuizTrendWindowRejectsUnknownPeriod(t *testing.T) {
	_, err := dailyQuizTrendWindow("90d", time.Now())
	if !errors.Is(err, ErrInvalidTrendPeriod) {
		t.Fatalf("got error %v, want %v", err, ErrInvalidTrendPeriod)
	}
}

func TestDashboardSearchRejectsQueriesShorterThanThreeCharacters(t *testing.T) {
	service := NewDashboardService(nil)
	for _, query := range []string{"", "  ", "ab", " аб ", strings.Repeat("a", 101)} {
		t.Run(query, func(t *testing.T) {
			_, err := service.Search(context.Background(), query)
			if !errors.Is(err, ErrInvalidSearchQuery) {
				t.Fatalf("got error %v, want %v", err, ErrInvalidSearchQuery)
			}
		})
	}
}
