package router

import (
	"net/http"
	"testing"
	"time"

	loginmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/auth/login"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/token"
)

type adminTaskResponse struct {
	ID              string          `json:"id"`
	Text            string          `json:"text"`
	Status          string          `json:"status"`
	CreatedBy       adminTaskActor  `json:"created_by"`
	CreatedAt       time.Time       `json:"created_at"`
	StatusUpdatedBy *adminTaskActor `json:"status_updated_by"`
	StatusUpdatedAt *time.Time      `json:"status_updated_at"`
}

type adminTaskActor struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Name     string `json:"name"`
}

func adminTaskToken(t *testing.T, id, role string) string {
	t.Helper()
	accessToken, err := token.NewAccessToken(id, role, testJWTSecret, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	return accessToken
}

func TestAdminTasks_SharedStatusAndAudit(t *testing.T) {
	r, db := setupRouter(t)
	superID := createUser(t, db, loginmodel.RoleSuperAdmin, "super.task", loginmodel.UserStatusActive)
	adminOneID := createUser(t, db, loginmodel.RoleAdmin, "admin.task.one", loginmodel.UserStatusActive)
	adminTwoID := createUser(t, db, loginmodel.RoleAdmin, "admin.task.two", loginmodel.UserStatusActive)

	superToken := adminTaskToken(t, superID, loginmodel.RoleSuperAdmin)
	adminOneToken := adminTaskToken(t, adminOneID, loginmodel.RoleAdmin)
	adminTwoToken := adminTaskToken(t, adminTwoID, loginmodel.RoleAdmin)

	res := sendRequest(r, http.MethodPost, "/api/v1/tasks", `{"text":"  Upload question bank  "}`, superToken)
	if res.Code != http.StatusCreated {
		t.Fatalf("create task: status %d, body: %s", res.Code, res.Body)
	}
	var created adminTaskResponse
	decodeData(t, res, &created)
	if created.Text != "Upload question bank" || created.Status != "pending" ||
		created.CreatedBy.ID != superID || created.StatusUpdatedBy != nil || created.StatusUpdatedAt != nil {
		t.Fatalf("unexpected created task: %+v", created)
	}

	for _, accessToken := range []string{adminOneToken, adminTwoToken, superToken} {
		res = sendRequest(r, http.MethodGet, "/api/v1/tasks", "", accessToken)
		if res.Code != http.StatusOK {
			t.Fatalf("list tasks: status %d, body: %s", res.Code, res.Body)
		}
		var tasks []adminTaskResponse
		decodeData(t, res, &tasks)
		if len(tasks) != 1 || tasks[0].ID != created.ID {
			t.Fatalf("task not shared with all Admin and Super Admin: %+v", tasks)
		}
	}

	res = sendRequest(r, http.MethodPatch, "/api/v1/tasks/"+created.ID+"/status", `{"status":"completed"}`, adminTwoToken)
	if res.Code != http.StatusOK {
		t.Fatalf("update task status: status %d, body: %s", res.Code, res.Body)
	}
	var updated adminTaskResponse
	decodeData(t, res, &updated)
	if updated.Status != "completed" || updated.StatusUpdatedBy == nil ||
		updated.StatusUpdatedBy.ID != adminTwoID || updated.StatusUpdatedBy.Username != "admin.task.two" ||
		updated.StatusUpdatedAt == nil || updated.StatusUpdatedAt.Before(created.CreatedAt) {
		t.Fatalf("status update audit is missing: %+v", updated)
	}

	res = sendRequest(r, http.MethodGet, "/api/v1/tasks", "", adminOneToken)
	if res.Code != http.StatusOK {
		t.Fatalf("list updated task as another Admin: status %d, body: %s", res.Code, res.Body)
	}
	var visibleToAdmin []adminTaskResponse
	decodeData(t, res, &visibleToAdmin)
	if len(visibleToAdmin) != 1 || visibleToAdmin[0].Status != "completed" ||
		visibleToAdmin[0].StatusUpdatedBy == nil ||
		visibleToAdmin[0].StatusUpdatedBy.ID != adminTwoID ||
		visibleToAdmin[0].StatusUpdatedAt == nil {
		t.Fatalf("shared task update is not visible to other Admin: %+v", visibleToAdmin)
	}

	res = sendRequest(r, http.MethodGet, "/api/v1/tasks/"+created.ID, "", superToken)
	if res.Code != http.StatusOK {
		t.Fatalf("get task as Super Admin: status %d, body: %s", res.Code, res.Body)
	}
	var visibleToSuperAdmin adminTaskResponse
	decodeData(t, res, &visibleToSuperAdmin)
	if visibleToSuperAdmin.Status != "completed" ||
		visibleToSuperAdmin.StatusUpdatedBy == nil ||
		visibleToSuperAdmin.StatusUpdatedBy.ID != adminTwoID ||
		visibleToSuperAdmin.StatusUpdatedAt == nil {
		t.Fatalf("Super Admin cannot see latest status update: %+v", visibleToSuperAdmin)
	}

	if res := sendRequest(r, http.MethodPost, "/api/v1/tasks", `{"text":"Not allowed"}`, adminOneToken); res.Code != http.StatusForbidden {
		t.Errorf("Admin should not create tasks: status %d, body: %s", res.Code, res.Body)
	}
	if res := sendRequest(r, http.MethodPatch, "/api/v1/tasks/"+created.ID+"/status", `{"status":"pending"}`, superToken); res.Code != http.StatusForbidden {
		t.Errorf("Super Admin should not update task status: status %d, body: %s", res.Code, res.Body)
	}
	if res := sendRequest(r, http.MethodGet, "/api/v1/tasks", "", tokenFor(t, loginmodel.RoleTutor)); res.Code != http.StatusForbidden {
		t.Errorf("Tutor should not access tasks: status %d, body: %s", res.Code, res.Body)
	}
}

func TestAdminTasks_ValidationAndNotFound(t *testing.T) {
	r, db := setupRouter(t)
	superID := createUser(t, db, loginmodel.RoleSuperAdmin, "super.task.validation", loginmodel.UserStatusActive)
	adminID := createUser(t, db, loginmodel.RoleAdmin, "admin.task.validation", loginmodel.UserStatusActive)
	superToken := adminTaskToken(t, superID, loginmodel.RoleSuperAdmin)
	adminToken := adminTaskToken(t, adminID, loginmodel.RoleAdmin)

	res := sendRequest(r, http.MethodPost, "/api/v1/tasks", `{"text":"   "}`, superToken)
	if res.Code != http.StatusUnprocessableEntity || errorFields(t, res.Body.String())["text"] != "wajib diisi" {
		t.Errorf("blank task text: status %d, body: %s", res.Code, res.Body)
	}

	res = sendRequest(r, http.MethodPost, "/api/v1/tasks", `{"text":"Valid task"}`, superToken)
	if res.Code != http.StatusCreated {
		t.Fatalf("create task: status %d, body: %s", res.Code, res.Body)
	}
	var created adminTaskResponse
	decodeData(t, res, &created)

	res = sendRequest(r, http.MethodPatch, "/api/v1/tasks/"+created.ID+"/status", `{"status":"done"}`, adminToken)
	if res.Code != http.StatusUnprocessableEntity || errorFields(t, res.Body.String())["status"] == "" {
		t.Errorf("invalid task status: status %d, body: %s", res.Code, res.Body)
	}

	res = sendRequest(r, http.MethodPatch, "/api/v1/tasks/not-a-uuid/status", `{"status":"completed"}`, adminToken)
	if res.Code != http.StatusNotFound {
		t.Errorf("invalid task ID: status %d, body: %s", res.Code, res.Body)
	}
}
