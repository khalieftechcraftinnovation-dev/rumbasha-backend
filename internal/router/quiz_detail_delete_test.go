// Test GET /quizzes/:id (detail) dan DELETE /quizzes (hapus banyak, soft delete).
package router

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"

	loginmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/login"
)

func TestGetQuizDetail(t *testing.T) {
	r, db := setupRouter(t)
	p := seedPlacementInGrade(t, db, "GRADE_6")
	accessToken := adminToken(t, db)

	q1 := createQuestionID(t, r, accessToken, questionBody(p, "single_choice", singleChoiceAnswer, nil))
	q2 := createQuestionID(t, r, accessToken, questionBody(p, "free_text", freeTextNumberAnswer, nil))
	created := createQuiz(t, r, accessToken, quizBody(basePlacement(p), []string{q2, q1}, map[string]string{"name": `"Uji Kom Detail"`}))

	res := sendRequest(r, http.MethodGet, "/api/v1/quizzes/"+created.ID, "", accessToken)
	if res.Code != http.StatusOK {
		t.Fatalf("detail: status %d. body: %s", res.Code, res.Body)
	}
	var detail quizDetail
	decodeData(t, res, &detail)
	if detail.ID != created.ID || detail.Name != "Uji Kom Detail" || detail.AccessCode != created.AccessCode ||
		detail.ScheduleStatus != "upcoming" || detail.Placement.Grade.Name != "Kelas 6" ||
		!detail.StartAt.Equal(created.StartAt) || !detail.EndAt.Equal(created.EndAt) ||
		strings.Join(questionIDsOf(detail), ",") != q2+","+q1 || detail.CreatedBy.Name != "User admin.soal" {
		t.Errorf("detail salah: %+v", detail)
	}
	// Daftar soal ringkas: tanpa kunci jawaban.
	if strings.Contains(res.Body.String(), `"answer"`) || strings.Contains(res.Body.String(), "correct_key") {
		t.Errorf("detail quiz tidak boleh berisi kunci jawaban. body: %s", res.Body)
	}

	// Tidak ada / bukan UUID / sudah dihapus → 404. Bukan admin → 403.
	db.Exec(`UPDATE quizzes SET deleted_at = NOW() WHERE id = ?`, created.ID)
	for _, id := range []string{uuid.NewString(), "bukan-uuid", created.ID} {
		if res := sendRequest(r, http.MethodGet, "/api/v1/quizzes/"+id, "", accessToken); res.Code != http.StatusNotFound ||
			!strings.Contains(res.Body.String(), "Quiz tidak ditemukan") {
			t.Errorf("detail %s: status %d, seharusnya 404", id, res.Code)
		}
	}
	if res := sendRequest(r, http.MethodGet, "/api/v1/quizzes/"+created.ID, "", tokenFor(t, loginmodel.RoleTutor)); res.Code != http.StatusForbidden {
		t.Errorf("tutor: status %d, seharusnya 403", res.Code)
	}
}

func TestDeleteQuizzes_SoftDeleteBanyakSekaligus(t *testing.T) {
	r, db := setupRouter(t)
	p := seedPlacementInGrade(t, db, "GRADE_6")
	accessToken := adminToken(t, db)
	q := createQuestionID(t, r, accessToken, questionBody(p, "single_choice", singleChoiceAnswer, nil))

	upcoming := createQuiz(t, r, accessToken, quizBody(basePlacement(p), []string{q}, map[string]string{"name": `"Akan Datang"`}))
	finished := createQuiz(t, r, accessToken, quizBody(basePlacement(p), []string{q}, map[string]string{"name": `"Sudah Lewat"`}))
	ongoing := createQuiz(t, r, accessToken, quizBody(basePlacement(p), []string{q}, map[string]string{"name": `"Sedang Jalan"`}))
	keep := createQuiz(t, r, accessToken, quizBody(basePlacement(p), []string{q}, map[string]string{"name": `"Tetap Ada"`}))
	db.Exec(`UPDATE quizzes SET start_at = NOW() - INTERVAL '3 hours', end_at = NOW() - INTERVAL '2 hours' WHERE id = ?`, finished.ID)
	db.Exec(`UPDATE quizzes SET start_at = NOW() - INTERVAL '30 minutes', end_at = NOW() + INTERVAL '2 hours' WHERE id = ?`, ongoing.ID)

	deleteBody := func(ids ...string) string { return `{"ids":` + jsonIDs(ids...) + `}` }
	isDeleted := func(id string) bool {
		var deleted bool
		db.Raw(`SELECT deleted_at IS NOT NULL AND deleted_by IS NOT NULL FROM quizzes WHERE id = ?`, id).Scan(&deleted)
		return deleted
	}

	// Ada yang sedang berjalan & ada yang tidak ditemukan → semua ditolak, tidak ada yang terhapus.
	missing := uuid.NewString()
	res := sendRequest(r, http.MethodDelete, "/api/v1/quizzes", deleteBody(upcoming.ID, ongoing.ID, missing), accessToken)
	if res.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status %d, seharusnya 422. body: %s", res.Code, res.Body)
	}
	fields := errorFields(t, res.Body.String())
	if !strings.HasPrefix(fields["ids[1]"], `quiz "Sedang Jalan" sedang berjalan (sampai `) ||
		fields["ids[2]"] != "quiz tidak ditemukan atau sudah dihapus" || fields["ids[0]"] != "" {
		t.Errorf("error hapus: %v", fields)
	}
	if isDeleted(upcoming.ID) {
		t.Error("quiz yang valid tidak boleh ikut terhapus kalau ada yang ditolak")
	}

	// Yang belum dimulai & sudah lewat boleh dihapus sekaligus.
	res = sendRequest(r, http.MethodDelete, "/api/v1/quizzes", deleteBody(upcoming.ID, finished.ID), accessToken)
	if res.Code != http.StatusOK || !strings.Contains(res.Body.String(), `"2 quiz berhasil dihapus"`) {
		t.Fatalf("hapus: status %d. body: %s", res.Code, res.Body)
	}
	if !isDeleted(upcoming.ID) || !isDeleted(finished.ID) || isDeleted(ongoing.ID) || isDeleted(keep.ID) {
		t.Error("hanya quiz yang dipilih yang terhapus (soft delete: baris tetap ada, deleted_at & deleted_by terisi)")
	}

	// Setelah dihapus: tidak tampil di list/bundle, tidak bisa dibuka, jadwal & kode aksesnya bebas dipakai lagi.
	items, _ := listQuizzes(t, r, accessToken, nil)
	if quizNames(items) != "Tetap Ada,Sedang Jalan" {
		t.Errorf("list setelah hapus = %s", quizNames(items))
	}
	if res := sendRequest(r, http.MethodGet, "/api/v1/quizzes/"+upcoming.ID, "", accessToken); res.Code != http.StatusNotFound {
		t.Errorf("detail quiz terhapus: status %d, seharusnya 404", res.Code)
	}
	startAt := upcoming.StartAt.Format("2006-01-02T15:04")
	endAt := upcoming.EndAt.Format("2006-01-02T15:04")
	createQuiz(t, r, accessToken, quizBody(basePlacement(p), []string{q}, schedule(startAt, endAt))) // jadwal sama: boleh

	// Sudah dihapus → ditolak; validasi body; hanya admin.
	res = sendRequest(r, http.MethodDelete, "/api/v1/quizzes", deleteBody(upcoming.ID), accessToken)
	if errorFields(t, res.Body.String())["ids[0]"] != "quiz tidak ditemukan atau sudah dihapus" {
		t.Errorf("hapus 2x: body %s", res.Body)
	}
	for body, field := range map[string]string{
		fmt.Sprintf(`{"ids":["%s","%s"]}`, keep.ID, keep.ID): "ids",    // kembar
		`{"ids":["bukan-uuid"]}`:                             "ids[0]", // bukan UUID
		`{"ids":[]}`:                                         "ids",    // kosong
	} {
		res := sendRequest(r, http.MethodDelete, "/api/v1/quizzes", body, accessToken)
		if res.Code != http.StatusUnprocessableEntity || errorFields(t, res.Body.String())[field] == "" {
			t.Errorf("body %s: status %d, body %s, seharusnya error di %s", body, res.Code, res.Body, field)
		}
	}
	if res := sendRequest(r, http.MethodDelete, "/api/v1/quizzes", deleteBody(keep.ID), tokenFor(t, loginmodel.RoleTutor)); res.Code != http.StatusForbidden {
		t.Errorf("tutor: status %d, seharusnya 403", res.Code)
	}
}
