// Package user (service) berisi logika menu User di panel admin: list (filter & cari), jumlah per
// role, detail, tambah (1-100 sekaligus), ubah, dan hapus (soft delete) akun.
//
// Hak akses: admin & super admin boleh melihat semua user. Menambah / mengubah / menghapus: super
// admin semua akun; admin hanya tutor & siswa. Akun sendiri tidak bisa dihapus atau dinonaktifkan.
//
// Di package lain, import dengan alias userservice:
//
//	userservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/user"
package user

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/clock"
	userdto "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/dto/user"
	loginmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/login"
	usermodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/user"
	userrepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/user"
	loginservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/login"
	subscriptionservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/subscription"
)

// UserService berisi logika menu User.
type UserService struct {
	repo          *userrepo.UserRepository
	subscriptions *subscriptionservice.SubscriptionService // akses siswa (kolom Paket, Akses, Berlaku s/d)
}

// NewUserService membuat UserService. Dipanggil sekali di main.go.
func NewUserService(repo *userrepo.UserRepository, subscriptions *subscriptionservice.SubscriptionService) *UserService {
	return &UserService{repo: repo, subscriptions: subscriptions}
}

// attachAccess mengisi akses hari ini untuk setiap siswa di items (2 query untuk semua siswa,
// bukan per baris). Role lain tetap nil.
func (s *UserService) attachAccess(ctx context.Context, items []usermodel.ListItem) error {
	var studentIDs []string
	for _, u := range items {
		if u.Role == loginmodel.RoleStudent {
			studentIDs = append(studentIDs, u.ID)
		}
	}
	if len(studentIDs) == 0 {
		return nil
	}
	access, err := s.subscriptions.AccessMany(ctx, studentIDs)
	if err != nil {
		return err
	}
	for i := range items {
		if a, ok := access[items[i].ID]; ok {
			items[i].Access = &a
		}
	}
	return nil
}

// usernameFormat sama dengan constraint users_username_format di database.
var usernameFormat = regexp.MustCompile(`^[a-z0-9._]{3,30}$`)

// roleNames dipakai di pesan error.
var roleNames = map[string]string{
	loginmodel.RoleSuperAdmin: "super admin",
	loginmodel.RoleAdmin:      "admin",
	loginmodel.RoleTutor:      "tutor",
	loginmodel.RoleStudent:    "siswa",
}

// filterFrom menyusun filter list dari query (default halaman 1, 20 per halaman).
func filterFrom(q userdto.ListQuery) usermodel.ListFilter {
	now := time.Now()
	f := usermodel.ListFilter{
		Role: q.Role, EducationLevelID: q.EducationLevelID, GradeID: q.GradeID, BranchID: q.BranchID,
		Status: q.Status, Search: strings.TrimSpace(q.Search), Page: q.Page, PerPage: q.PerPage,
		PlanID: q.PlanID, Subscription: q.Subscription,
		Today: clock.TodayInJakarta(now), ExpiringUntil: clock.TodayInJakarta(now.AddDate(0, 0, 6)),
	}
	if f.Page == 0 {
		f.Page = 1
	}
	if f.PerPage == 0 {
		f.PerPage = 20
	}
	return f
}

// List mengambil 1 halaman user (terbaru dulu) sesuai filter.
func (s *UserService) List(ctx context.Context, q userdto.ListQuery) ([]usermodel.ListItem, usermodel.ListFilter, int, error) {
	f := filterFrom(q)
	items, total, err := s.repo.List(ctx, f)
	if err != nil {
		return nil, f, 0, err
	}
	return items, f, total, s.attachAccess(ctx, items)
}

// RoleCounts menghitung user per role dengan filter yang sama seperti list (role diabaikan).
func (s *UserService) RoleCounts(ctx context.Context, q userdto.ListQuery) (usermodel.RoleCounts, error) {
	return s.repo.CountByRole(ctx, filterFrom(q))
}

// GetDetail mengambil 1 user lengkap. Bukan UUID / tidak ada / sudah dihapus → ErrUserNotFound.
func (s *UserService) GetDetail(ctx context.Context, id string) (usermodel.Detail, error) {
	if uuid.Validate(id) != nil {
		return usermodel.Detail{}, usermodel.ErrUserNotFound
	}
	d, err := s.repo.FindDetail(ctx, id)
	if err != nil {
		return usermodel.Detail{}, err
	}
	items := []usermodel.ListItem{d.ListItem}
	if err := s.attachAccess(ctx, items); err != nil {
		return usermodel.Detail{}, err
	}
	d.ListItem = items[0]
	return d, nil
}

// ---------------------------------------------------------------------------
// Aturan data akun (dipakai bersama oleh tambah & ubah)
// ---------------------------------------------------------------------------

// userInput adalah isi 1 akun dari request tambah / ubah (belum dirapikan).
type userInput struct {
	Role                                    string
	Username, Name, Status                  string
	Email, Phone, ParentPhone, SchoolOrigin *string
	BranchID, EducationLevelID, GradeID     *string
	DateOfBirth, TargetSchool               *string
	JoinDate                                string
	EndDate                                 *string
}

// cleanUser adalah isi 1 akun yang sudah dirapikan & lolos validasi.
type cleanUser struct {
	Username, RealName, Status              string
	Email, Phone, ParentPhone, SchoolOrigin *string
	BranchID, EducationLevelID, GradeID     *string
	DateOfBirth                             *time.Time
	TargetSchool                            *string
	JoinDate                                time.Time
	EndDate                                 *time.Time
}

// loadLookups mengambil data pendukung validasi untuk semua akun di request sekaligus (maks 3
// query, bukan per akun): cabang aktif, jenjang/kelas, dan (kalau checkUsernames) username terpakai.
func (s *UserService) loadLookups(ctx context.Context, inputs []userInput, checkUsernames bool) (usermodel.Lookups, error) {
	branchIDs, masterIDs, usernames := []string{}, []string{}, []string{}
	for _, in := range inputs {
		if b := optional(in.BranchID); b != nil {
			branchIDs = append(branchIDs, *b)
		}
		for _, id := range []*string{in.EducationLevelID, in.GradeID} {
			if v := optional(id); v != nil {
				masterIDs = append(masterIDs, *v)
			}
		}
		usernames = append(usernames, normalizeUsername(in.Username))
	}

	var lk usermodel.Lookups
	var err error
	if lk.ActiveBranches, err = s.repo.ActiveBranches(ctx, branchIDs); err != nil {
		return lk, err
	}
	lk.MasterData = map[string]usermodel.MasterDataItem{}
	if len(masterIDs) > 0 {
		if lk.MasterData, err = s.repo.FindMasterData(ctx, masterIDs); err != nil {
			return lk, err
		}
	}
	lk.TakenUsernames = map[string]bool{}
	if checkUsernames {
		if lk.TakenUsernames, err = s.repo.TakenUsernames(ctx, usernames); err != nil {
			return lk, err
		}
	}
	return lk, nil
}

// validateInput merapikan & mengecek 1 akun (tanpa query database: semua data dari lk).
// prefix = awalan nama field di pesan error, contoh "users[2]." ("" untuk ubah 1 user).
func validateInput(in userInput, prefix string, lk usermodel.Lookups, errs *fieldErrors) cleanUser {
	add := func(field, message string) { errs.add(prefix+field, message) }

	u := cleanUser{
		Username:     normalizeUsername(in.Username),
		RealName:     strings.TrimSpace(in.Name),
		Status:       in.Status,
		Email:        optional(in.Email),
		Phone:        optional(in.Phone),
		ParentPhone:  optional(in.ParentPhone),
		SchoolOrigin: optional(in.SchoolOrigin),
		BranchID:     optional(in.BranchID),
	}
	if u.Status == "" {
		u.Status = loginmodel.UserStatusActive
	}
	if !usernameFormat.MatchString(u.Username) {
		add("username", "3-30 karakter, hanya huruf kecil, angka, titik, atau garis bawah")
	} else if lk.TakenUsernames[u.Username] {
		add("username", "sudah dipakai akun lain")
	}
	if u.RealName == "" {
		add("name", "wajib diisi")
	}
	checkJoinDates(in, &u, add)
	if _, known := roleNames[in.Role]; !known {
		return u // role tidak valid (sudah dilaporkan): aturan per role tidak bisa dicek
	}

	// Cabang: siswa, tutor, admin wajib 1 cabang aktif; super admin tanpa cabang (semua cabang).
	switch {
	case in.Role == loginmodel.RoleSuperAdmin && u.BranchID != nil:
		add("branch_id", "super admin berlaku untuk semua cabang, kosongkan")
	case in.Role != loginmodel.RoleSuperAdmin && u.BranchID == nil:
		add("branch_id", "wajib diisi untuk "+roleNames[in.Role])
	case u.BranchID != nil && !lk.ActiveBranches[*u.BranchID]:
		add("branch_id", "cabang tidak ditemukan atau tidak aktif")
	}

	levelID, gradeID := optional(in.EducationLevelID), optional(in.GradeID)
	if in.Role != loginmodel.RoleStudent {
		if levelID != nil {
			add("education_level_id", "hanya untuk siswa, kosongkan untuk "+roleNames[in.Role])
		}
		if gradeID != nil {
			add("grade_id", "hanya untuk siswa, kosongkan untuk "+roleNames[in.Role])
		}
		return u
	}

	// Siswa: jenjang & kelas wajib, aktif, dan kelas milik jenjang itu.
	if levelID == nil {
		add("education_level_id", "wajib diisi untuk siswa")
	}
	if gradeID == nil {
		add("grade_id", "wajib diisi untuk siswa")
	}
	if levelID != nil && gradeID != nil {
		level, levelOK := lk.MasterData[*levelID]
		grade, gradeOK := lk.MasterData[*gradeID]
		switch {
		case !levelOK || level.Type != "education_level" || level.Status != "active":
			add("education_level_id", "jenjang tidak ditemukan atau tidak aktif")
		case !gradeOK || grade.Type != "grade" || grade.Status != "active":
			add("grade_id", "kelas tidak ditemukan atau tidak aktif")
		case grade.ParentID == nil || *grade.ParentID != level.ID:
			add("grade_id", fmt.Sprintf("%s bukan bagian dari jenjang %s", grade.Name, level.Name))
		}
	}
	u.EducationLevelID, u.GradeID = levelID, gradeID
	u.TargetSchool = optional(in.TargetSchool)
	if dob := optional(in.DateOfBirth); dob != nil {
		t, err := time.ParseInLocation(userdto.DateLayout, *dob, clock.Jakarta)
		if err != nil || t.After(time.Now()) {
			add("date_of_birth", "format 2012-05-17 dan tidak boleh di masa depan")
		} else {
			u.DateOfBirth = &t
		}
	}
	return u
}

// ---------------------------------------------------------------------------
// Tambah user
// ---------------------------------------------------------------------------

// Create menambah 1-100 akun sekaligus (semua atau tidak sama sekali), lalu mengembalikan datanya
// sesuai urutan request. Kesalahan per field "users[i].<field>", contoh username kembar di request
// maupun dengan akun yang sudah ada. Admin hanya boleh menambah tutor & siswa.
//
// Performa: cek data pendukung 3 query untuk semua akun, hash password paralel, simpan 2 query.
func (s *UserService) Create(ctx context.Context, req userdto.CreateRequest, actorID, actorRole string) ([]usermodel.ListItem, error) {
	inputs := make([]userInput, len(req.Users))
	for i, item := range req.Users {
		inputs[i] = inputFromCreateItem(item)
	}
	lk, err := s.loadLookups(ctx, inputs, true)
	if err != nil {
		return nil, err
	}

	errs := &fieldErrors{}
	users := make([]usermodel.NewUser, len(inputs))
	firstIndex := map[string]int{} // username → nomor akun pertama yang memakainya di request
	for i, in := range inputs {
		prefix := fmt.Sprintf("users[%d].", i)
		if !usermodel.CanManage(actorRole, in.Role) {
			errs.add(prefix+"role", fmt.Sprintf("%s tidak boleh menambah akun %s", roleNames[actorRole], roleNames[in.Role]))
		}
		u := validateInput(in, prefix, lk, errs)
		if j, dup := firstIndex[u.Username]; dup && u.Username != "" {
			errs.add(prefix+"username", fmt.Sprintf("sama dengan users[%d]", j))
		} else {
			firstIndex[u.Username] = i
		}
		users[i] = usermodel.NewUser{
			Role: in.Role, Username: u.Username, RealName: u.RealName, Email: u.Email, Phone: u.Phone,
			ParentPhone: u.ParentPhone, SchoolOrigin: u.SchoolOrigin, BranchID: u.BranchID,
			EducationLevelID: u.EducationLevelID, GradeID: u.GradeID, Status: u.Status,
			DateOfBirth: u.DateOfBirth, TargetSchool: u.TargetSchool,
			JoinDate: u.JoinDate, EndDate: u.EndDate,
		}
	}
	if errs.any() {
		return nil, errs.toError()
	}

	passwords := make([]string, len(req.Users))
	for i, item := range req.Users {
		passwords[i] = item.Password
	}
	hashes, err := hashPasswords(passwords)
	if err != nil {
		return nil, err
	}
	for i := range users {
		users[i].PasswordHash = hashes[i]
	}

	ids, err := s.repo.Create(ctx, users, actorID, time.Now())
	if err == usermodel.ErrUsernameTaken { // 2 request bersamaan memakai username yang sama
		return nil, &usermodel.ValidationError{Fields: []usermodel.FieldError{{
			Field: "users", Message: "ada username yang baru saja dipakai akun lain, cek ulang lalu simpan lagi",
		}}}
	}
	if err != nil {
		return nil, err
	}
	items, err := s.repo.FindByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	return items, s.attachAccess(ctx, items)
}

// ---------------------------------------------------------------------------
// Ubah user
// ---------------------------------------------------------------------------

// Update mengganti data 1 user (ganti penuh, role tetap). Semua kesalahan dikembalikan sekaligus
// (ValidationError). Dinonaktifkan atau password diganti → semua sesi login user itu diakhiri.
func (s *UserService) Update(ctx context.Context, id string, req userdto.UpdateRequest, actorID, actorRole string) (usermodel.Detail, error) {
	if uuid.Validate(id) != nil {
		return usermodel.Detail{}, usermodel.ErrUserNotFound
	}
	targets, err := s.repo.FindTargets(ctx, []string{id})
	if err != nil {
		return usermodel.Detail{}, err
	}
	target, ok := targets[id]
	if !ok {
		return usermodel.Detail{}, usermodel.ErrUserNotFound
	}
	if !usermodel.CanManage(actorRole, target.Role) {
		return usermodel.Detail{}, usermodel.ErrForbidden
	}

	in := userInput{
		Role: target.Role, Username: req.Username, Name: req.Name, Status: req.Status,
		Email: req.Email, Phone: req.Phone, ParentPhone: req.ParentPhone, SchoolOrigin: req.SchoolOrigin,
		BranchID: req.BranchID, EducationLevelID: req.EducationLevelID, GradeID: req.GradeID,
		DateOfBirth: req.DateOfBirth, TargetSchool: req.TargetSchool,
		JoinDate: req.JoinDate, EndDate: req.EndDate,
	}
	// Username kembar dicek database saat UPDATE (username lama milik user ini sendiri boleh).
	lk, err := s.loadLookups(ctx, []userInput{in}, false)
	if err != nil {
		return usermodel.Detail{}, err
	}
	errs := &fieldErrors{}
	u := validateInput(in, "", lk, errs)
	if id == actorID && u.Status != loginmodel.UserStatusActive {
		errs.add("status", "tidak bisa menonaktifkan akun sendiri")
	}
	if errs.any() {
		return usermodel.Detail{}, errs.toError()
	}

	update := usermodel.Update{
		ID: id, Username: u.Username, RealName: u.RealName, Email: u.Email, Phone: u.Phone,
		ParentPhone: u.ParentPhone, SchoolOrigin: u.SchoolOrigin, BranchID: u.BranchID,
		EducationLevelID: u.EducationLevelID, GradeID: u.GradeID, Status: u.Status,
		DateOfBirth: u.DateOfBirth, TargetSchool: u.TargetSchool,
		JoinDate: u.JoinDate, EndDate: u.EndDate,
	}
	if req.Password != nil && *req.Password != "" {
		hash, err := loginservice.HashPassword(*req.Password)
		if err != nil {
			return usermodel.Detail{}, err
		}
		update.PasswordHash = &hash
	}
	// Sesi login diakhiri kalau password diganti, dinonaktifkan, atau tanggal berakhir sudah lewat.
	ended := update.EndDate != nil && update.EndDate.Format(time.DateOnly) < time.Now().In(clock.Jakarta).Format(time.DateOnly)
	revokeSessions := update.PasswordHash != nil || update.Status != loginmodel.UserStatusActive || ended

	err = s.repo.Update(ctx, update, target.Role == loginmodel.RoleStudent, revokeSessions, actorID, time.Now())
	if err == usermodel.ErrUsernameTaken {
		return usermodel.Detail{}, &usermodel.ValidationError{Fields: []usermodel.FieldError{{
			Field: "username", Message: "sudah dipakai akun lain",
		}}}
	}
	if err != nil {
		return usermodel.Detail{}, err
	}
	return s.GetDetail(ctx, id)
}

// ---------------------------------------------------------------------------
// Hapus user
// ---------------------------------------------------------------------------

// Delete menghapus 1-100 user (soft delete) dan mengakhiri sesi login mereka. Semua atau tidak sama
// sekali: ID tidak ada, akun sendiri, atau akun yang tidak boleh dihapus actor → 422 per "ids[i]".
func (s *UserService) Delete(ctx context.Context, ids []string, actorID, actorRole string) ([]string, error) {
	targets, err := s.repo.FindTargets(ctx, ids)
	if err != nil {
		return nil, err
	}
	errs := &fieldErrors{}
	for i, id := range ids {
		field := fmt.Sprintf("ids[%d]", i)
		t, ok := targets[id]
		switch {
		case !ok:
			errs.add(field, "user tidak ditemukan atau sudah dihapus")
		case id == actorID:
			errs.add(field, "tidak bisa menghapus akun sendiri")
		case !usermodel.CanManage(actorRole, t.Role):
			errs.add(field, fmt.Sprintf("%s tidak boleh menghapus akun %s (%s)", roleNames[actorRole], roleNames[t.Role], t.RealName))
		}
	}
	if errs.any() {
		return nil, errs.toError()
	}
	if err := s.repo.SoftDelete(ctx, ids, actorID, time.Now()); err != nil {
		return nil, err
	}
	return ids, nil
}

// ---------------------------------------------------------------------------
// Bantuan
// ---------------------------------------------------------------------------

// normalizeUsername: huruf kecil tanpa spasi di ujung (sama dengan aturan login).
func normalizeUsername(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// optional merapikan teks opsional: spasi di ujung dibuang, kosong = nil (NULL).
func optional(s *string) *string {
	if s == nil {
		return nil
	}
	v := strings.TrimSpace(*s)
	if v == "" {
		return nil
	}
	return &v
}

// fieldErrors mengumpulkan kesalahan supaya semuanya dikembalikan sekaligus.
type fieldErrors struct {
	list []usermodel.FieldError
}

func (e *fieldErrors) add(field, message string) {
	e.list = append(e.list, usermodel.FieldError{Field: field, Message: message})
}

func (e *fieldErrors) any() bool {
	return len(e.list) > 0
}

func (e *fieldErrors) toError() error {
	return &usermodel.ValidationError{Fields: e.list}
}

// checkJoinDates membaca join_date (wajib) & end_date (opsional, tidak boleh sebelum join_date).
// Format "2026-01-15"; tanggal dibaca sebagai tanggal WIB.
func checkJoinDates(in userInput, u *cleanUser, add func(field, message string)) {
	const formatMsg = "format tanggal 2026-01-15"
	join, err := time.ParseInLocation(userdto.DateLayout, strings.TrimSpace(in.JoinDate), clock.Jakarta)
	if err != nil {
		add("join_date", "wajib diisi, "+formatMsg)
	} else {
		u.JoinDate = join
	}
	end := optional(in.EndDate)
	if end == nil {
		return
	}
	t, err := time.ParseInLocation(userdto.DateLayout, *end, clock.Jakarta)
	switch {
	case err != nil:
		add("end_date", formatMsg)
	case !u.JoinDate.IsZero() && t.Before(u.JoinDate):
		add("end_date", "tidak boleh sebelum join_date ("+u.JoinDate.Format(userdto.DateLayout)+")")
	default:
		u.EndDate = &t
	}
}
