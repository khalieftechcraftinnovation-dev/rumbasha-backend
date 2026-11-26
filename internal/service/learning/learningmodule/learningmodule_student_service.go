package learningmodule

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"

	learningmoduledto "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/dto/learning/learningmodule"
	subscriptionmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/billing/subscription"
	learningmodulemodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/learning/learningmodule"
)

// Aturan akses modul untuk siswa:
//
//  1. Hanya modul PUBLISH di JENJANG akun siswa (siswa SD tidak melihat modul SMP, dst.).
//  2. Bisa dibuka kalau fitur modul terbuka DAN:
//     - modul tersedia di paket aktif siswa hari ini (learning_module_plans), atau
//     - fitur modul dibuka admin lewat akses khusus (student_feature_overrides) tanpa paket yang
//       punya fitur modul → semua modul di jenjangnya terbuka.
//  3. Modul lain di jenjangnya tetap tampil di list dengan locked = true (+ paket yang membukanya);
//     membuka / mengunduhnya ditolak 403.
//
// Paket aktif & fitur dihitung oleh SubscriptionService (aturan yang sama dengan GET /auth/me).

// StudentFileURL = link unduh PDF untuk siswa (API wajib login, akses dicek).
func (s *LearningModuleService) StudentFileURL(id string) string {
	return s.storage.BaseURL + "/api/v1/student/learning-modules/" + id + "/file"
}

// StudentAccess menghitung akses modul siswa hari ini: jenjang akun + paket aktif + fitur modul.
func (s *LearningModuleService) StudentAccess(ctx context.Context, studentID string) (learningmodulemodel.StudentAccess, error) {
	var a learningmodulemodel.StudentAccess
	level, err := s.repo.FindStudentLevel(ctx, studentID)
	if err != nil {
		return a, err
	}
	a.EducationLevel = level
	if level != nil {
		a.EducationLevelID = &level.ID
	}

	access, err := s.subscriptions.GetTodayAccess(ctx, studentID)
	if err != nil {
		return a, err
	}
	a.ModuleFeature = access.Features[subscriptionmodel.FeatureModule]
	if access.Subscription != nil {
		a.Plan = &learningmodulemodel.Ref{ID: access.Subscription.PlanID, Name: access.Subscription.PlanName}
	}
	// Fitur modul terbuka tapi BUKAN dari paket (paket tidak ada / tidak punya fitur modul)
	// → dibuka admin lewat akses khusus: semua modul di jenjang terbuka.
	a.AllModules = a.ModuleFeature && (access.Subscription == nil || !access.Subscription.HasModule)
	return a, nil
}

// studentFilter menyusun filter dasar (jenjang, paket, siswa) dari akses siswa.
func studentFilter(a learningmodulemodel.StudentAccess, studentID string) learningmodulemodel.StudentFilter {
	f := learningmodulemodel.StudentFilter{
		ModuleFeature: a.ModuleFeature, AllModules: a.AllModules, StudentID: studentID,
	}
	if a.EducationLevelID != nil {
		f.EducationLevelID = *a.EducationLevelID
	}
	if a.Plan != nil {
		f.PlanID = a.Plan.ID
	}
	return f
}

// StudentList mengambil 1 halaman modul di jenjang siswa (yang terkunci ikut, locked = true).
// Siswa tanpa jenjang → daftar kosong.
func (s *LearningModuleService) StudentList(ctx context.Context, studentID string, q learningmoduledto.StudentListQuery) (learningmodulemodel.StudentAccess, []learningmodulemodel.StudentModule, learningmodulemodel.StudentFilter, int, error) {
	a, err := s.StudentAccess(ctx, studentID)
	if err != nil {
		return a, nil, learningmodulemodel.StudentFilter{}, 0, err
	}
	f := studentFilter(a, studentID)
	f.GradeID, f.SubjectID, f.SubjectFieldID, f.ChapterID = q.GradeID, q.SubjectID, q.SubjectFieldID, q.ChapterID
	f.Search, f.Access, f.Page, f.PerPage = strings.TrimSpace(q.Search), q.Access, q.Page, q.PerPage
	if f.Page == 0 {
		f.Page = 1
	}
	if f.PerPage == 0 {
		f.PerPage = 20
	}
	if a.EducationLevelID == nil {
		return a, []learningmodulemodel.StudentModule{}, f, 0, nil
	}
	items, total, err := s.repo.StudentList(ctx, f)
	return a, items, f, total, err
}

// Open membuka 1 modul untuk siswa: cek jenjang & paket, catat riwayat buka (terakhir dibuka,
// jumlah buka), lalu mengembalikan detail + link PDF.
// Bukan jenjang siswa / draf / tidak ada → ErrModuleNotFound; di luar paket → *LockedError.
func (s *LearningModuleService) Open(ctx context.Context, studentID, moduleID string) (learningmodulemodel.StudentModule, error) {
	m, err := s.studentModule(ctx, studentID, moduleID)
	if err != nil {
		return m, err
	}
	first, last, count, err := s.repo.RecordOpen(ctx, studentID, moduleID, time.Now())
	if err != nil {
		return m, err
	}
	m.FirstOpenedAt, m.LastOpenedAt, m.OpenCount = &first, &last, count
	return m, nil
}

// StudentFilePath mengembalikan lokasi file PDF untuk siswa (akses dicek sama dengan Open).
func (s *LearningModuleService) StudentFilePath(ctx context.Context, studentID, moduleID string) (string, string, error) {
	m, err := s.studentModule(ctx, studentID, moduleID)
	if err != nil {
		return "", "", err
	}
	filePath, err := s.storage.FilePath(m.StoragePath)
	return filePath, m.FileName, err
}

// studentModule mengambil modul untuk siswa dan memastikan siswa boleh membukanya.
func (s *LearningModuleService) studentModule(ctx context.Context, studentID, moduleID string) (learningmodulemodel.StudentModule, error) {
	if uuid.Validate(moduleID) != nil {
		return learningmodulemodel.StudentModule{}, learningmodulemodel.ErrModuleNotFound
	}
	a, err := s.StudentAccess(ctx, studentID)
	if err != nil {
		return learningmodulemodel.StudentModule{}, err
	}
	if a.EducationLevelID == nil {
		return learningmodulemodel.StudentModule{}, learningmodulemodel.ErrModuleNotFound
	}
	m, err := s.repo.FindStudentModule(ctx, moduleID, studentFilter(a, studentID))
	if err != nil {
		return m, err
	}
	if m.Locked {
		return m, lockedError(a, m.Plans)
	}
	return m, nil
}

// lockedError menyusun pesan modul terkunci sesuai kondisi akses siswa.
func lockedError(a learningmodulemodel.StudentAccess, plans []learningmodulemodel.Ref) error {
	names := make([]string, 0, len(plans))
	for _, p := range plans {
		names = append(names, p.Name)
	}
	message := "Modul ini tersedia di paket " + strings.Join(names, ", ")
	switch {
	case a.Plan == nil:
		message = "Kamu belum punya paket aktif. " + message
	case !a.ModuleFeature:
		message = "Paket " + a.Plan.Name + " tidak termasuk akses modul. " + message
	default:
		message = "Modul ini tidak termasuk paket " + a.Plan.Name + ". " + message
	}
	return &learningmodulemodel.LockedError{Message: message, Plans: plans}
}
