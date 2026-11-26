package router

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	loginmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/auth/login"
)

type notificationAnnouncement struct {
	ID        string     `json:"id"`
	Title     string     `json:"title"`
	Body      string     `json:"body"`
	Status    string     `json:"status"`
	PublishAt *time.Time `json:"publish_at"`
	IsRead    bool       `json:"is_read"`
	ReadAt    *time.Time `json:"read_at"`
	GradeIDs  []string   `json:"grade_ids"`
}

func TestAnnouncements_TargetingScheduleAndPerStudentReadState(t *testing.T) {
	r, db := setupRouter(t)
	admin := adminToken(t, db)
	studentOne := studentToken(t, db, "GRADE_5")
	studentTwo := studentToken(t, db, "GRADE_5")
	studentOtherGrade := studentToken(t, db, "GRADE_6")

	var targetGradeID string
	if err := db.Raw(`SELECT id FROM master_data_items WHERE type = 'grade' AND code = 'GRADE_5'`).Scan(&targetGradeID).Error; err != nil {
		t.Fatal(err)
	}
	if targetGradeID == "" {
		t.Fatal("fixture GRADE_5 not found")
	}

	createBody := fmt.Sprintf(
		`{"title":"Info kelas 5","body":"Pengumuman umum untuk kelas 5.","grade_ids":[%q],"status":"published"}`,
		targetGradeID,
	)
	res := sendRequest(r, http.MethodPost, "/api/v1/announcements", createBody, admin)
	if res.Code != http.StatusCreated {
		t.Fatalf("create announcement: status %d, body: %s", res.Code, res.Body)
	}
	var published notificationAnnouncement
	decodeData(t, res, &published)
	if published.Status != "published" || published.PublishAt == nil || len(published.GradeIDs) != 1 {
		t.Fatalf("unexpected published announcement: %+v", published)
	}

	futureAt := time.Now().UTC().Add(2 * time.Hour).Format(time.RFC3339)
	scheduledBody := fmt.Sprintf(
		`{"title":"Info nanti","body":"Pengumuman terjadwal.","grade_ids":[%q],"status":"published","publish_at":%q}`,
		targetGradeID, futureAt,
	)
	res = sendRequest(r, http.MethodPost, "/api/v1/announcements", scheduledBody, admin)
	if res.Code != http.StatusCreated {
		t.Fatalf("schedule announcement: status %d, body: %s", res.Code, res.Body)
	}
	var scheduled notificationAnnouncement
	decodeData(t, res, &scheduled)

	res = sendRequest(r, http.MethodGet, "/api/v1/student/notifications", "", studentOne)
	if res.Code != http.StatusOK {
		t.Fatalf("list student notifications: status %d, body: %s", res.Code, res.Body)
	}
	var feed []notificationAnnouncement
	decodeData(t, res, &feed)
	if len(feed) != 1 || feed[0].ID != published.ID || feed[0].IsRead {
		t.Fatalf("feed must contain only due announcements for the student's grade: %+v", feed)
	}

	res = sendRequest(r, http.MethodGet, "/api/v1/student/notifications/unread-count", "", studentOne)
	var unread struct {
		Count int64 `json:"count"`
	}
	decodeData(t, res, &unread)
	if unread.Count != 1 {
		t.Fatalf("unread count = %d, want 1", unread.Count)
	}

	res = sendRequest(r, http.MethodPost, "/api/v1/student/notifications/"+published.ID+"/read", "", studentOne)
	if res.Code != http.StatusOK {
		t.Fatalf("mark notification read: status %d, body: %s", res.Code, res.Body)
	}
	res = sendRequest(r, http.MethodGet, "/api/v1/student/notifications/unread-count", "", studentOne)
	decodeData(t, res, &unread)
	if unread.Count != 0 {
		t.Fatalf("unread count after marking read = %d, want 0", unread.Count)
	}

	res = sendRequest(r, http.MethodGet, "/api/v1/student/notifications", "", studentTwo)
	decodeData(t, res, &feed)
	if len(feed) != 1 || feed[0].IsRead {
		t.Fatalf("read state must be isolated per student: %+v", feed)
	}
	res = sendRequest(r, http.MethodGet, "/api/v1/student/notifications", "", studentOtherGrade)
	decodeData(t, res, &feed)
	if len(feed) != 0 {
		t.Fatalf("announcement for grade 5 was visible to another grade: %+v", feed)
	}

	res = sendRequest(r, http.MethodGet, "/api/v1/announcements?status=published", "", admin)
	if res.Code != http.StatusOK {
		t.Fatalf("admin list announcements: status %d, body: %s", res.Code, res.Body)
	}
	var managed []notificationAnnouncement
	decodeData(t, res, &managed)
	if len(managed) != 2 || managed[0].ID == "" {
		t.Fatalf("admin should see published and scheduled announcements: %+v", managed)
	}

	if res := sendRequest(r, http.MethodGet, "/api/v1/announcements", "", tokenFor(t, loginmodel.RoleTutor)); res.Code != http.StatusForbidden {
		t.Errorf("Tutor should not manage announcements: status %d, body: %s", res.Code, res.Body)
	}
	if res := sendRequest(r, http.MethodGet, "/api/v1/announcements", "", tokenFor(t, loginmodel.RoleSuperAdmin)); res.Code != http.StatusOK {
		t.Errorf("Super Admin should manage announcements: status %d, body: %s", res.Code, res.Body)
	}
	if res := sendRequest(r, http.MethodPost, "/api/v1/announcements", createBody, studentOne); res.Code != http.StatusForbidden {
		t.Errorf("Student should not create announcements: status %d, body: %s", res.Code, res.Body)
	}
	if res := sendRequest(r, http.MethodPost, "/api/v1/student/notifications/"+scheduled.ID+"/read", "", studentOne); res.Code != http.StatusNotFound {
		t.Errorf("Student should not read a scheduled announcement early: status %d, body: %s", res.Code, res.Body)
	}
}

func TestAnnouncements_DraftAndMarkAllRead(t *testing.T) {
	r, db := setupRouter(t)
	admin := adminToken(t, db)
	student := studentToken(t, db, "GRADE_5")

	var gradeID string
	if err := db.Raw(`SELECT id FROM master_data_items WHERE type = 'grade' AND code = 'GRADE_5'`).Scan(&gradeID).Error; err != nil {
		t.Fatal(err)
	}
	body := fmt.Sprintf(`{"title":"Draf","body":"Belum tayang.","grade_ids":[%q]}`, gradeID)
	res := sendRequest(r, http.MethodPost, "/api/v1/announcements", body, admin)
	if res.Code != http.StatusCreated {
		t.Fatalf("create draft: status %d, body: %s", res.Code, res.Body)
	}
	var draft notificationAnnouncement
	decodeData(t, res, &draft)
	if draft.Status != "draft" || draft.PublishAt != nil {
		t.Fatalf("new announcement should default to draft: %+v", draft)
	}

	res = sendRequest(r, http.MethodGet, "/api/v1/student/notifications", "", student)
	var feed []notificationAnnouncement
	decodeData(t, res, &feed)
	if len(feed) != 0 {
		t.Fatalf("draft announcement was visible to student: %+v", feed)
	}

	publishBody := fmt.Sprintf(`{"title":"Draf","body":"Sudah tayang.","grade_ids":[%q],"status":"published"}`, gradeID)
	res = sendRequest(r, http.MethodPut, "/api/v1/announcements/"+draft.ID, publishBody, admin)
	if res.Code != http.StatusOK {
		t.Fatalf("publish announcement: status %d, body: %s", res.Code, res.Body)
	}

	secondBody := fmt.Sprintf(`{"title":"Kabar baru","body":"Pengumuman kedua.","grade_ids":[%q],"status":"published"}`, gradeID)
	res = sendRequest(r, http.MethodPost, "/api/v1/announcements", secondBody, admin)
	if res.Code != http.StatusCreated {
		t.Fatalf("create second announcement: status %d, body: %s", res.Code, res.Body)
	}

	res = sendRequest(r, http.MethodPost, "/api/v1/student/notifications/read-all", "", student)
	if res.Code != http.StatusOK {
		t.Fatalf("mark all notifications read: status %d, body: %s", res.Code, res.Body)
	}
	var marked struct {
		MarkedCount int64 `json:"marked_count"`
	}
	decodeData(t, res, &marked)
	if marked.MarkedCount != 2 {
		t.Fatalf("marked count = %d, want 2", marked.MarkedCount)
	}

	res = sendRequest(r, http.MethodDelete, "/api/v1/announcements/"+draft.ID, "", admin)
	if res.Code != http.StatusOK {
		t.Fatalf("delete announcement: status %d, body: %s", res.Code, res.Body)
	}

	res = sendRequest(r, http.MethodGet, "/api/v1/student/notifications/unread-count", "", student)
	var unread struct {
		Count int64 `json:"count"`
	}
	decodeData(t, res, &unread)
	if unread.Count != 0 {
		t.Fatalf("unread count after mark-all = %d, want 0", unread.Count)
	}
}
