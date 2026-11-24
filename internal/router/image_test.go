// Test endpoint Image Library: POST /images (upload) dan DELETE /images (hapus permanen).
package router

import (
	"bytes"
	"fmt"
	stdimage "image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	loginmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/login"
)

// uploadFile adalah 1 file untuk dikirim di field "files".
type uploadFile struct {
	Name string
	Data []byte
}

// makeImage membuat gambar kecil (w x h) dalam format png / jpeg / gif.
func makeImage(t *testing.T, format string, w, h int) []byte {
	t.Helper()
	img := stdimage.NewRGBA(stdimage.Rect(0, 0, w, h))
	for x := range w {
		img.Set(x, 0, color.RGBA{R: 255, A: 255})
	}
	buf := &bytes.Buffer{}
	var err error
	switch format {
	case "png":
		err = png.Encode(buf, img)
	case "jpeg":
		err = jpeg.Encode(buf, img, nil)
	case "gif":
		err = gif.Encode(buf, img, nil)
	}
	if err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// sendUpload mengirim multipart ke POST /api/v1/images.
func sendUpload(r *gin.Engine, fields map[string]string, files []uploadFile, accessToken string) *httptest.ResponseRecorder {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	for k, v := range fields {
		_ = writer.WriteField(k, v)
	}
	for _, f := range files {
		part, _ := writer.CreateFormFile("files", f.Name)
		_, _ = part.Write(f.Data)
	}
	_ = writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/images", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+accessToken)
	}
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

type imageData struct {
	ID             string  `json:"id"`
	Category       string  `json:"category"`
	Name           string  `json:"name"`
	URL            string  `json:"url"`
	MimeType       string  `json:"mime_type"`
	Width          int     `json:"width"`
	Height         int     `json:"height"`
	SubjectID      *string `json:"subject_id"`
	ChapterID      *string `json:"chapter_id"`
	TopicID        *string `json:"topic_id"`
	UploadedBy     string  `json:"uploaded_by"`
	SubjectFieldID *string `json:"subject_field_id"`
}

// uploadImages meng-upload gambar & mengembalikan datanya (test gagal kalau bukan 201).
func uploadImages(t *testing.T, r *gin.Engine, accessToken string, fields map[string]string, files ...uploadFile) []imageData {
	t.Helper()
	res := sendUpload(r, fields, files, accessToken)
	if res.Code != http.StatusCreated {
		t.Fatalf("upload gagal: status %d. body: %s", res.Code, res.Body)
	}
	var list []imageData
	decodeData(t, res, &list)
	return list
}

// storagePathOf mengubah URL gambar menjadi lokasi file di folder storage test.
func storagePathOf(app testApp, url string) string {
	rel := strings.TrimPrefix(url, testAppURL+"/storage/")
	return filepath.Join(app.StorageDir, filepath.FromSlash(rel))
}

func TestUploadImage_SatuGambarDenganHierarkiLaluBisaDibuka(t *testing.T) {
	app := setupApp(t)
	p := seedPlacement(t, app.DB)
	accessToken := adminToken(t, app.DB)

	fields := map[string]string{
		"category": "question", "name": "Segitiga siku-siku",
		"subject_id": p.SubjectID, "subject_field_id": p.FieldID, "chapter_id": p.ChapterID,
		"subchapter_id": p.SubchapterID, "topic_id": p.TopicID,
	}
	list := uploadImages(t, app.Router, accessToken, fields, uploadFile{"segitiga.png", makeImage(t, "png", 40, 30)})
	img := list[0]

	if img.Name != "Segitiga siku-siku" || img.Category != "question" || img.MimeType != "image/png" ||
		img.Width != 40 || img.Height != 30 || img.TopicID == nil || *img.TopicID != p.TopicID || img.UploadedBy == "" {
		t.Errorf("data gambar tidak sesuai: %+v", img)
	}
	if !strings.HasPrefix(img.URL, testAppURL+"/storage/images/question/") || !strings.HasSuffix(img.URL, img.ID+".png") {
		t.Errorf("url = %s", img.URL)
	}

	// File tersimpan di disk & bisa dibuka publik (tanpa login) lewat URL-nya.
	if _, err := os.Stat(storagePathOf(app, img.URL)); err != nil {
		t.Errorf("file tidak ada di disk: %v", err)
	}
	res := sendRequest(app.Router, http.MethodGet, strings.TrimPrefix(img.URL, testAppURL), "", "")
	if res.Code != http.StatusOK || res.Header().Get("Content-Type") != "image/png" {
		t.Errorf("gambar tidak bisa dibuka: status %d, content-type %s", res.Code, res.Header().Get("Content-Type"))
	}

	// Gambar bisa langsung dipakai di soal (image_id).
	body := questionBody(p, "single_choice", singleChoiceAnswer, map[string]string{
		"image_id": fmt.Sprintf("%q", img.ID), "display_order": `["question","image"]`,
	})
	if res := sendRequest(app.Router, http.MethodPost, "/api/v1/questions", body, accessToken); res.Code != http.StatusCreated {
		t.Errorf("soal dengan gambar hasil upload gagal dibuat: %d %s", res.Code, res.Body)
	}
}

func TestUploadImage_BanyakGambarNamaDiberiNomor(t *testing.T) {
	app := setupApp(t)
	accessToken := adminToken(t, app.DB)

	list := uploadImages(t, app.Router, accessToken, map[string]string{"category": "banner", "name": "Promo Desember"},
		uploadFile{"a.png", makeImage(t, "png", 10, 10)},
		uploadFile{"b.jpg", makeImage(t, "jpeg", 20, 10)},
		uploadFile{"c.gif", makeImage(t, "gif", 10, 20)},
	)
	wantTypes := []string{"image/png", "image/jpeg", "image/gif"}
	wantExt := []string{".png", ".jpg", ".gif"}
	for i, img := range list {
		if img.Name != fmt.Sprintf("Promo Desember %d", i+1) || img.Category != "banner" || img.MimeType != wantTypes[i] ||
			!strings.HasSuffix(img.URL, wantExt[i]) || img.SubjectID != nil {
			t.Errorf("gambar ke-%d tidak sesuai: %+v", i+1, img)
		}
	}
}

func TestUploadImage_Validasi(t *testing.T) {
	app := setupApp(t)
	p := seedPlacement(t, app.DB)
	accessToken := adminToken(t, app.DB)
	png1 := uploadFile{"a.png", makeImage(t, "png", 5, 5)}

	tooMany := make([]uploadFile, 11)
	for i := range tooMany {
		tooMany[i] = png1
	}
	bigFile := uploadFile{"besar.png", append(makeImage(t, "png", 5, 5), make([]byte, 5<<20)...)}

	tests := []struct {
		name       string
		fields     map[string]string
		files      []uploadFile
		wantFields []string
	}{
		{"tanpa file", map[string]string{"category": "question", "name": "x"}, nil, []string{"files"}},
		{"lebih dari 10 file", map[string]string{"category": "question", "name": "x"}, tooMany, []string{"files"}},
		{"kategori & nama salah", map[string]string{"category": "poster"}, []uploadFile{png1}, []string{"category", "name"}},
		{
			"bukan gambar / svg / terlalu besar (file valid ikut ditolak)",
			map[string]string{"category": "question", "name": "x"},
			[]uploadFile{png1, {"virus.png", []byte("MZ bukan gambar")}, {"x.svg", []byte(`<svg xmlns="http://www.w3.org/2000/svg"></svg>`)}, bigFile},
			[]string{"files[1]", "files[2]", "files[3]"},
		},
		{"hierarki di kategori banner", map[string]string{"category": "banner", "name": "x", "subject_id": p.SubjectID}, []uploadFile{png1}, []string{"subject_id"}},
		{"bab tanpa bidang materi", map[string]string{"category": "question", "name": "x", "subject_id": p.SubjectID, "chapter_id": p.ChapterID}, []uploadFile{png1}, []string{"subject_field_id"}},
		{
			"bab bukan bagian bidang materi (ID tertukar)",
			map[string]string{"category": "question", "name": "x", "subject_id": p.SubjectID, "subject_field_id": p.FieldID, "chapter_id": p.SubchapterID},
			[]uploadFile{png1}, []string{"chapter_id"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := sendUpload(app.Router, tt.fields, tt.files, accessToken)
			if res.Code != http.StatusUnprocessableEntity {
				t.Fatalf("status = %d, seharusnya 422. body: %s", res.Code, res.Body)
			}
			got := errorFields(t, res.Body.String())
			for _, f := range tt.wantFields {
				if _, ok := got[f]; !ok {
					t.Errorf("error untuk field %q tidak ada. errors: %v", f, got)
				}
			}
		})
	}

	// Tidak ada yang tersimpan (database & disk).
	var count int64
	app.DB.Raw(`SELECT COUNT(*) FROM media_assets`).Scan(&count)
	entries, _ := os.ReadDir(app.StorageDir)
	if count != 0 || len(entries) != 0 {
		t.Errorf("tidak boleh ada gambar tersimpan: %d baris, %d isi folder", count, len(entries))
	}

	// Siswa & tutor tidak boleh upload.
	for _, role := range []string{loginmodel.RoleStudent, loginmodel.RoleTutor} {
		if res := sendUpload(app.Router, map[string]string{"category": "question", "name": "x"}, []uploadFile{png1}, tokenFor(t, role)); res.Code != http.StatusForbidden {
			t.Errorf("%s: status = %d, seharusnya 403", role, res.Code)
		}
	}
}

func TestDeleteImages_BanyakSekaligusHapusPermanen(t *testing.T) {
	app := setupApp(t)
	accessToken := adminToken(t, app.DB)
	list := uploadImages(t, app.Router, accessToken, map[string]string{"category": "landing", "name": "Hero"},
		uploadFile{"a.png", makeImage(t, "png", 5, 5)}, uploadFile{"b.png", makeImage(t, "png", 5, 5)}, uploadFile{"c.png", makeImage(t, "png", 5, 5)})

	body := fmt.Sprintf(`{"ids":[%q,%q]}`, list[0].ID, list[1].ID)
	res := sendRequest(app.Router, http.MethodDelete, "/api/v1/images", body, accessToken)
	if res.Code != http.StatusOK || !strings.Contains(res.Body.String(), "2 gambar berhasil dihapus") {
		t.Fatalf("status = %d. body: %s", res.Code, res.Body)
	}

	// Data & file terhapus permanen; gambar ketiga tetap ada.
	for i, img := range list {
		var count int64
		app.DB.Raw(`SELECT COUNT(*) FROM media_assets WHERE id = ?`, img.ID).Scan(&count)
		_, statErr := os.Stat(storagePathOf(app, img.URL))
		deleted := i < 2
		if (count == 0) != deleted || os.IsNotExist(statErr) != deleted {
			t.Errorf("gambar ke-%d: baris=%d, file ada=%v, seharusnya terhapus=%v", i+1, count, statErr == nil, deleted)
		}
	}
}

func TestDeleteImages_DitolakKalauTidakAdaAtauMasihDipakaiSoal(t *testing.T) {
	app := setupApp(t)
	p := seedPlacement(t, app.DB)
	accessToken := adminToken(t, app.DB)
	list := uploadImages(t, app.Router, accessToken, map[string]string{"category": "question", "name": "Gbr"},
		uploadFile{"a.png", makeImage(t, "png", 5, 5)}, uploadFile{"b.png", makeImage(t, "png", 5, 5)},
		uploadFile{"c.png", makeImage(t, "png", 5, 5)}, uploadFile{"d.png", makeImage(t, "png", 5, 5)})
	soalImg, opsiImg, bebasImg, soalTerhapusImg := list[0].ID, list[1].ID, list[2].ID, list[3].ID

	// Soal aktif memakai gambar sebagai gambar soal & gambar opsi.
	answer := fmt.Sprintf(`{"options":[{"key":"A","content":"","image_id":%q},{"key":"B","content":"2"}],"correct_key":"A"}`, opsiImg)
	createQuestionID(t, app.Router, accessToken, questionBody(p, "single_choice", answer, map[string]string{
		"image_id": fmt.Sprintf("%q", soalImg), "display_order": `["question","image"]`,
	}))
	// Soal yang sudah dihapus (soft delete) memakai gambar lain.
	deletedQ := createQuestionID(t, app.Router, accessToken, questionBody(p, "single_choice", singleChoiceAnswer, map[string]string{
		"image_id": fmt.Sprintf("%q", soalTerhapusImg), "display_order": `["question","image"]`,
	}))
	app.DB.Exec(`UPDATE questions SET deleted_at = NOW() WHERE id = ?`, deletedQ)

	body := fmt.Sprintf(`{"ids":[%q,%q,%q,"00000000-0000-0000-0000-000000000000"]}`, bebasImg, soalImg, opsiImg)
	res := sendRequest(app.Router, http.MethodDelete, "/api/v1/images", body, accessToken)
	if res.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, seharusnya 422. body: %s", res.Code, res.Body)
	}
	got := errorFields(t, res.Body.String())
	if !strings.Contains(got["ids[1]"], "masih dipakai soal Q-") || !strings.Contains(got["ids[2]"], "masih dipakai soal Q-") ||
		got["ids[3]"] != "gambar tidak ditemukan" || got["ids[0]"] != "" {
		t.Errorf("errors tidak sesuai: %v", got)
	}
	var count int64
	app.DB.Raw(`SELECT COUNT(*) FROM media_assets`).Scan(&count)
	if count != 4 {
		t.Errorf("tidak boleh ada gambar terhapus, sisa %d", count)
	}

	// Gambar yang hanya dipakai soal terhapus boleh dihapus; soal itu dilepas dari gambarnya.
	res = sendRequest(app.Router, http.MethodDelete, "/api/v1/images", fmt.Sprintf(`{"ids":[%q]}`, soalTerhapusImg), accessToken)
	if res.Code != http.StatusOK {
		t.Fatalf("status = %d. body: %s", res.Code, res.Body)
	}
	var imageID *string
	app.DB.Raw(`SELECT image_id FROM questions WHERE id = ?`, deletedQ).Scan(&imageID)
	if imageID != nil {
		t.Errorf("image_id soal terhapus seharusnya NULL, = %v", *imageID)
	}
}
