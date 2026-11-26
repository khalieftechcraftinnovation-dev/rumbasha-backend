// Package dashboard defines API response types for the Admin dashboard.
package dashboard

import (
	dashboardmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/admin/dashboard"
)

type DailyQuizTrendQuery struct {
	Period string `form:"period" binding:"omitempty,oneof=12w 30d 7d"`
}

type SearchQuery struct {
	Query string `form:"q" binding:"required"`
}

type SummaryResponse dashboardmodel.Summary

func ToSummaryResponse(summary dashboardmodel.Summary) SummaryResponse {
	return SummaryResponse(summary)
}

type DailyQuizTrendResponse dashboardmodel.DailyQuizTrend

func ToDailyQuizTrendResponse(trend dashboardmodel.DailyQuizTrend) DailyQuizTrendResponse {
	return DailyQuizTrendResponse(trend)
}

type SearchResponse dashboardmodel.SearchResults

func ToSearchResponse(results dashboardmodel.SearchResults) SearchResponse {
	return SearchResponse(results)
}
