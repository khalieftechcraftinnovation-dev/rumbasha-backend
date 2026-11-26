// Package dashboard prepares aggregated data for the Admin dashboard.
package dashboard

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/clock"
	dashboardmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/admin/dashboard"
	dashboardrepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/admin/dashboard"
)

const (
	TrendPeriod12Weeks = "12w"
	TrendPeriod30Days  = "30d"
	TrendPeriod7Days   = "7d"
)

var ErrInvalidTrendPeriod = errors.New("invalid Daily Quiz trend period")
var ErrInvalidSearchQuery = errors.New("dashboard search query must contain at least 3 characters")

type DashboardService struct {
	repo *dashboardrepo.DashboardRepository
}

func NewDashboardService(repo *dashboardrepo.DashboardRepository) *DashboardService {
	return &DashboardService{repo: repo}
}

func (s *DashboardService) Summary(ctx context.Context) (dashboardmodel.Summary, error) {
	now := time.Now().In(clock.Jakarta)
	today := clock.StartOfDayInJakarta(now)
	monthStart := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, clock.Jakarta)
	weekStart := mondayOf(today)
	periods := dashboardmodel.SummaryPeriods{
		MonthStart:         monthStart,
		MonthEnd:           monthStart.AddDate(0, 1, 0),
		PreviousMonthStart: monthStart.AddDate(0, -1, 0),
		WeekStart:          weekStart,
		QuizWindowStart:    today.AddDate(0, 0, -29),
		TodayStart:         today,
		TomorrowStart:      today.AddDate(0, 0, 1),
	}

	summary, err := s.repo.Summary(ctx, periods)
	if err != nil {
		return dashboardmodel.Summary{}, err
	}
	summary.GeneratedAt = now
	summary.QuizCompletion.PeriodStart = periods.QuizWindowStart.Format(time.DateOnly)
	summary.QuizCompletion.PeriodEnd = today.Format(time.DateOnly)
	return summary, nil
}

func (s *DashboardService) Search(ctx context.Context, query string) (dashboardmodel.SearchResults, error) {
	query = strings.TrimSpace(query)
	queryLength := utf8.RuneCountInString(query)
	if queryLength < 3 || queryLength > 100 {
		return dashboardmodel.SearchResults{}, ErrInvalidSearchQuery
	}
	return s.repo.Search(ctx, query)
}

func (s *DashboardService) DailyQuizTrend(ctx context.Context, period string) (dashboardmodel.DailyQuizTrend, error) {
	now := time.Now().In(clock.Jakarta)
	window, err := dailyQuizTrendWindow(period, now)
	if err != nil {
		return dashboardmodel.DailyQuizTrend{}, err
	}

	rows, err := s.repo.DailyQuizTrendRows(ctx, window.from, window.lastBucket, window.to, period)
	if err != nil {
		return dashboardmodel.DailyQuizTrend{}, err
	}

	trend := dashboardmodel.DailyQuizTrend{
		Period:      period,
		PeriodStart: window.from.Format(time.DateOnly),
		PeriodEnd:   window.today.Format(time.DateOnly),
		Buckets:     make([]string, 0, len(window.buckets)),
		Series:      make([]dashboardmodel.TrendSeries, 0, 2),
	}
	for _, bucket := range window.buckets {
		trend.Buckets = append(trend.Buckets, bucket.Format(time.DateOnly))
	}

	seriesIndex := make(map[string]int, 2)
	for _, row := range rows {
		index, ok := seriesIndex[row.EducationLevel.ID]
		if !ok {
			index = len(trend.Series)
			seriesIndex[row.EducationLevel.ID] = index
			trend.Series = append(trend.Series, dashboardmodel.TrendSeries{
				EducationLevel: row.EducationLevel,
				Points:         make([]dashboardmodel.TrendPoint, 0, len(window.buckets)),
			})
		}
		trend.Series[index].Points = append(trend.Series[index].Points, dashboardmodel.TrendPoint{
			PeriodStart: row.BucketStart.Format(time.DateOnly),
			Sessions:    row.Sessions,
			Average:     row.Average,
		})
	}
	return trend, nil
}

type trendWindow struct {
	from       time.Time
	lastBucket time.Time
	to         time.Time
	today      time.Time
	buckets    []time.Time
}

func dailyQuizTrendWindow(period string, now time.Time) (trendWindow, error) {
	today := clock.StartOfDayInJakarta(now)
	window := trendWindow{today: today, to: today.AddDate(0, 0, 1)}
	stepDays := 1

	switch period {
	case TrendPeriod7Days:
		window.from = today.AddDate(0, 0, -6)
		window.lastBucket = today
	case TrendPeriod30Days:
		window.from = today.AddDate(0, 0, -29)
		window.lastBucket = today
	case TrendPeriod12Weeks:
		window.lastBucket = mondayOf(today)
		window.from = window.lastBucket.AddDate(0, 0, -7*11)
		stepDays = 7
	default:
		return trendWindow{}, ErrInvalidTrendPeriod
	}

	for bucket := window.from; !bucket.After(window.lastBucket); bucket = bucket.AddDate(0, 0, stepDays) {
		window.buckets = append(window.buckets, bucket)
	}
	return window, nil
}

func mondayOf(date time.Time) time.Time {
	offset := (int(date.Weekday()) + 6) % 7
	return date.AddDate(0, 0, -offset)
}
