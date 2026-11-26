package router

import (
	"net/http"
	"testing"

	dashboardmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/admin/dashboard"
	loginmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/auth/login"
)

func TestAdminDashboardSummaryAndDailyQuizTrend(t *testing.T) {
	r, db := setupRouter(t)
	adminID := createUser(t, db, loginmodel.RoleAdmin, "admin.dashboard", loginmodel.UserStatusActive)
	adminToken := adminTaskToken(t, adminID, loginmodel.RoleAdmin)

	res := sendRequest(r, http.MethodGet, "/api/v1/admin/dashboard/summary", "", adminToken)
	if res.Code != http.StatusOK {
		t.Fatalf("dashboard summary: status %d, body: %s", res.Code, res.Body)
	}
	var summary dashboardmodel.Summary
	decodeData(t, res, &summary)
	if summary.Users.Total != 1 || summary.Users.ByRole.Admins != 1 {
		t.Errorf("unexpected user summary: %+v", summary.Users)
	}
	if summary.QuizCompletion.PeriodStart == "" || summary.QuizCompletion.PeriodEnd == "" {
		t.Errorf("quiz completion period is missing: %+v", summary.QuizCompletion)
	}
	if summary.Tasks.Items == nil || summary.Tasks.RecentActivity == nil {
		t.Errorf("task arrays must be present: %+v", summary.Tasks)
	}

	res = sendRequest(r, http.MethodGet, "/api/v1/admin/dashboard/daily-quiz-trend?period=7d", "", adminToken)
	if res.Code != http.StatusOK {
		t.Fatalf("daily quiz trend: status %d, body: %s", res.Code, res.Body)
	}
	var trend dashboardmodel.DailyQuizTrend
	decodeData(t, res, &trend)
	if trend.Period != "7d" || len(trend.Buckets) != 7 {
		t.Errorf("unexpected 7-day trend: %+v", trend)
	}

	res = sendRequest(r, http.MethodGet, "/api/v1/admin/dashboard/daily-quiz-trend?period=90d", "", adminToken)
	if res.Code != http.StatusUnprocessableEntity {
		t.Errorf("unsupported trend period: status %d, body: %s", res.Code, res.Body)
	}
}

func TestAdminDashboardRejectsTutor(t *testing.T) {
	r, _ := setupRouter(t)
	res := sendRequest(r, http.MethodGet, "/api/v1/admin/dashboard/summary", "", tokenFor(t, loginmodel.RoleTutor))
	if res.Code != http.StatusForbidden {
		t.Fatalf("tutor dashboard access: status %d, body: %s", res.Code, res.Body)
	}
}
