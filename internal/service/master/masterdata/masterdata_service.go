// Package masterdata (service) berisi logika menu Master Data & Cabang di panel admin:
//
//   - list (tab per type, filter letak, cari, status) + jumlah per tab + detail, lengkap dengan
//     pemakaian data (kolom "Isi": soal, modul PDF, quiz, gambar, akun, item anak);
//   - tambah: mapel, bidang materi (pilih mapel + kelas; jenjang mengikuti kelas), bab / sub bab /
//     topik (cukup pilih induk; mapel, jenjang, kelas disalin dari induk), kelas (pilih jenjang), sekolah;
//   - ubah: nama, kode, urutan, status (type, induk, dan letak tetap). Data sistem (jenjang, kelas
//     bawaan, tipe soal, tipe quiz, level): hanya nama & urutan (FR-MST-009);
//   - hapus banyak (soft delete): ditolak kalau data sistem atau masih dipakai data lain (FR-MST-003).
//
// Di package lain, import dengan alias masterdataservice:
//
//	masterdataservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/master/masterdata"
package masterdata

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"

	masterdatadto "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/dto/master/masterdata"
	masterdatamodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/master/masterdata"
	masterdatarepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/master/masterdata"
)

// MasterDataService berisi logika Master Data & Cabang.
type MasterDataService struct {
	repo *masterdatarepo.MasterDataRepository
}

// NewMasterDataService membuat MasterDataService. Dipanggil sekali di main.go.
func NewMasterDataService(repo *masterdatarepo.MasterDataRepository) *MasterDataService {
	return &MasterDataService{repo: repo}
}

// codeFormat = kode data: huruf, angka, titik, garis bawah, strip (contoh MTK, GRADE_13).
var codeFormat = regexp.MustCompile(`^[A-Za-z0-9_.-]{1,30}$`)

// filterFrom menyusun filter list dari query (default halaman 1, 20 per halaman).
func filterFrom(q masterdatadto.ListQuery) masterdatamodel.ListFilter {
	f := masterdatamodel.ListFilter{
		Types: masterdatamodel.ManageableTypes, Search: strings.TrimSpace(q.Search), Status: q.Status,
		ParentID: q.ParentID, SubjectID: q.SubjectID, EducationLevelID: q.EducationLevelID, GradeID: q.GradeID,
		Page: q.Page, PerPage: q.PerPage,
	}
	if q.Type != "" {
		f.Types = []string{q.Type}
	}
	if f.Page == 0 {
		f.Page = 1
	}
	if f.PerPage == 0 {
		f.PerPage = 20
	}
	return f
}

// List mengambil 1 halaman data master + pemakaiannya (2 query untuk 1 halaman, bukan per baris).
func (s *MasterDataService) List(ctx context.Context, q masterdatadto.ListQuery) ([]masterdatamodel.Item, masterdatamodel.ListFilter, int, error) {
	f := filterFrom(q)
	items, total, err := s.repo.List(ctx, f)
	if err != nil {
		return nil, f, 0, err
	}
	return items, f, total, s.attachUsage(ctx, items)
}

// Counts menghitung angka tab (filter search & status ikut; type diabaikan) + jumlah cabang.
func (s *MasterDataService) Counts(ctx context.Context, q masterdatadto.ListQuery) (masterdatamodel.Counts, int, error) {
	f := filterFrom(q)
	counts, err := s.repo.CountByType(ctx, f)
	if err != nil {
		return nil, 0, err
	}
	branches, err := s.repo.CountBranches(ctx, masterdatamodel.BranchFilter{Search: f.Search, Status: f.Status})
	return counts, branches, err
}

// Get mengambil 1 data master + pemakaiannya. Bukan UUID / tidak ada / dihapus → ErrNotFound.
func (s *MasterDataService) Get(ctx context.Context, id string) (masterdatamodel.Item, error) {
	if uuid.Validate(id) != nil {
		return masterdatamodel.Item{}, masterdatamodel.ErrNotFound
	}
	item, err := s.repo.FindItem(ctx, id)
	if err != nil {
		return item, err
	}
	items := []masterdatamodel.Item{item}
	if err := s.attachUsage(ctx, items); err != nil {
		return item, err
	}
	return items[0], nil
}

func (s *MasterDataService) attachUsage(ctx context.Context, items []masterdatamodel.Item) error {
	ids := make([]string, 0, len(items))
	for _, i := range items {
		ids = append(ids, i.ID)
	}
	used, err := s.repo.Usage(ctx, ids)
	if err != nil {
		return err
	}
	for i := range items {
		items[i].Usage = used[items[i].ID]
	}
	return nil
}

// Create menambah 1 data master. Kesalahan dikumpulkan sekaligus (422).
func (s *MasterDataService) Create(ctx context.Context, req masterdatadto.CreateRequest, adminID string) (masterdatamodel.Item, error) {
	errs := fieldErrors{}
	rule := masterdatamodel.Types[req.Type]
	n := masterdatamodel.NewItem{Type: req.Type, Name: strings.TrimSpace(req.Name), SortOrder: req.SortOrder, Status: req.Status}
	if n.Name == "" {
		errs.add("name", "wajib diisi")
	}
	if n.Status == "" {
		n.Status = masterdatamodel.StatusActive
	}
	n.Code = cleanCode(req.Code, &errs)

	if err := s.placeNewItem(ctx, rule, req, &n, &errs); err != nil {
		return masterdatamodel.Item{}, err
	}
	if len(errs) > 0 {
		return masterdatamodel.Item{}, &masterdatamodel.ValidationError{Fields: errs}
	}

	id, err := s.repo.CreateItem(ctx, n, adminID)
	if err != nil {
		return masterdatamodel.Item{}, duplicateError(err, rule)
	}
	return s.Get(ctx, id)
}

// placeNewItem mengecek induk (dan kelas untuk bidang materi), lalu menyalin letak ke item baru:
//
//	mapel, sekolah      → tanpa induk
//	kelas               → induk jenjang
//	bidang materi       → induk mapel + kelas (jenjang = jenjang kelas)
//	bab, sub bab, topik → induk bidang materi / bab / sub bab; mapel, jenjang, kelas dari induk
func (s *MasterDataService) placeNewItem(ctx context.Context, rule masterdatamodel.TypeRule, req masterdatadto.CreateRequest, n *masterdatamodel.NewItem, errs *fieldErrors) error {
	if req.GradeID != nil && req.Type != masterdatamodel.TypeSubjectField {
		errs.add("grade_id", "hanya diisi untuk bidang materi (bab s/d topik mengikuti induknya)")
	}
	if rule.ParentType == "" {
		if req.ParentID != nil {
			errs.add("parent_id", rule.Label+" tidak punya induk, kosongkan")
		}
		return nil
	}

	parentLabel := masterdatamodel.Types[rule.ParentType].Label
	if req.ParentID == nil {
		errs.add("parent_id", "wajib diisi ("+strings.ToLower(parentLabel)+")")
		return nil
	}
	parent, err := s.repo.FindParent(ctx, *req.ParentID)
	switch {
	case err == masterdatamodel.ErrNotFound:
		errs.add("parent_id", "data induk tidak ditemukan")
		return nil
	case err != nil:
		return err
	case parent.Type != rule.ParentType:
		errs.add("parent_id", fmt.Sprintf("induk harus %s, bukan %s", strings.ToLower(parentLabel), strings.ToLower(masterdatamodel.Types[parent.Type].Label)))
		return nil
	}
	n.ParentID = &parent.ID

	switch req.Type {
	case masterdatamodel.TypeSubjectField:
		if req.GradeID == nil {
			errs.add("grade_id", "wajib diisi (kelas bidang materi)")
			return nil
		}
		grade, err := s.repo.FindParent(ctx, *req.GradeID)
		switch {
		case err == masterdatamodel.ErrNotFound || (err == nil && grade.Type != masterdatamodel.TypeGrade):
			errs.add("grade_id", "kelas tidak ditemukan")
			return nil
		case err != nil:
			return err
		}
		n.SubjectID, n.GradeID, n.EducationLevelID = &parent.ID, &grade.ID, grade.ParentID
	case masterdatamodel.TypeChapter, masterdatamodel.TypeSubchapter, masterdatamodel.TypeTopic:
		n.SubjectID, n.EducationLevelID, n.GradeID = parent.SubjectID, parent.EducationLevelID, parent.GradeID
	}
	return nil
}

// Update mengganti nama, kode, urutan, dan status 1 data (type, induk, letak tetap).
// Data sistem: kode & status tidak boleh berubah (code boleh dikosongkan = tetap).
func (s *MasterDataService) Update(ctx context.Context, id string, req masterdatadto.UpdateRequest, adminID string) (masterdatamodel.Item, error) {
	if uuid.Validate(id) != nil {
		return masterdatamodel.Item{}, masterdatamodel.ErrNotFound
	}
	current, err := s.repo.FindItem(ctx, id)
	if err != nil {
		return masterdatamodel.Item{}, err
	}
	errs := fieldErrors{}
	u := masterdatamodel.ItemUpdate{ID: id, Name: strings.TrimSpace(req.Name), SortOrder: *req.SortOrder, Status: req.Status}
	if u.Name == "" {
		errs.add("name", "wajib diisi")
	}
	u.Code = cleanCode(req.Code, &errs)

	if current.IsSystem {
		if u.Code == nil || (current.Code != nil && *u.Code == *current.Code) {
			u.Code = current.Code
		} else {
			errs.add("code", "kode data sistem tidak bisa diubah (dipakai program)")
		}
		if u.Status != current.Status {
			errs.add("status", "status data sistem tidak bisa diubah")
		}
	}
	if len(errs) > 0 {
		return masterdatamodel.Item{}, &masterdatamodel.ValidationError{Fields: errs}
	}
	if err := s.repo.UpdateItem(ctx, u, adminID); err != nil {
		return masterdatamodel.Item{}, duplicateError(err, masterdatamodel.Types[current.Type])
	}
	return s.Get(ctx, id)
}

// Delete menghapus 1-100 data (soft delete), semua atau tidak sama sekali. Kesalahan per "ids[i]":
// tidak ditemukan, data sistem, atau masih dipakai (dengan rinciannya).
func (s *MasterDataService) Delete(ctx context.Context, ids []string, adminID string) ([]string, error) {
	missing, blocked, err := s.repo.SoftDeleteItems(ctx, ids, adminID)
	if err != nil {
		return nil, err
	}
	if len(missing) == 0 && len(blocked) == 0 {
		return ids, nil
	}
	messages := map[string]string{}
	for _, id := range missing {
		messages[id] = "data tidak ditemukan atau sudah dihapus"
	}
	for _, b := range blocked {
		if b.IsSystem {
			messages[b.ID] = b.Name + " adalah data sistem, tidak bisa dihapus"
			continue
		}
		messages[b.ID] = b.Name + " masih dipakai " + usageText(b.Usage) + "; nonaktifkan atau pindahkan dulu"
	}
	errs := fieldErrors{}
	for i, id := range ids {
		if m, ok := messages[id]; ok {
			errs.add(fmt.Sprintf("ids[%d]", i), m)
		}
	}
	return nil, &masterdatamodel.ValidationError{Fields: errs}
}

// usageText = rincian pemakaian, contoh "3 item anak, 12 soal, 2 modul".
func usageText(u masterdatamodel.Usage) string {
	var parts []string
	for _, p := range []struct {
		n     int
		label string
	}{
		{u.Children, "item anak"}, {u.Questions, "soal"}, {u.Modules, "modul"},
		{u.Quizzes, "quiz"}, {u.Images, "gambar"}, {u.Users, "akun"},
	} {
		if p.n > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", p.n, p.label))
		}
	}
	return strings.Join(parts, ", ")
}

// cleanCode merapikan kode (trim). Kosong = nil; format salah dicatat di errs.
func cleanCode(code *string, errs *fieldErrors) *string {
	if code == nil {
		return nil
	}
	c := strings.TrimSpace(*code)
	if c == "" {
		return nil
	}
	if !codeFormat.MatchString(c) {
		errs.add("code", "hanya huruf, angka, titik, garis bawah, dan strip (maks 30)")
	}
	return &c
}

// duplicateError mengubah ErrNameTaken / ErrCodeTaken menjadi pesan validasi.
func duplicateError(err error, rule masterdatamodel.TypeRule) error {
	switch err {
	case masterdatamodel.ErrNameTaken:
		where := ""
		if rule.ParentType != "" {
			where = " di induk"
			if rule.Curriculum {
				where += " dan kelas"
			}
			where += " yang sama"
		}
		return &masterdatamodel.ValidationError{Fields: []masterdatamodel.FieldError{{Field: "name", Message: "nama " + strings.ToLower(rule.Label) + " sudah dipakai" + where}}}
	case masterdatamodel.ErrCodeTaken:
		return &masterdatamodel.ValidationError{Fields: []masterdatamodel.FieldError{{Field: "code", Message: "kode sudah dipakai " + strings.ToLower(rule.Label) + " lain (termasuk yang sudah dihapus)"}}}
	}
	return err
}

// fieldErrors mengumpulkan kesalahan per field.
type fieldErrors []masterdatamodel.FieldError

func (e *fieldErrors) add(field, message string) {
	*e = append(*e, masterdatamodel.FieldError{Field: field, Message: message})
}
