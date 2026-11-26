// Package user (dto) berisi bentuk request dan response JSON untuk menu User (panel admin).
//
// Di package lain, import dengan alias userdto:
//
//	userdto "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/dto/user"
package user

import (
	"time"

	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/clock"
	subscriptiondto "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/dto/subscription"
	usermodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/user"
)

// ListQuery adalah query string GET /api/v1/users. Semua opsional.
//
//	?role=student&education_level_id=...&grade_id=...&search=budi&page=1&per_page=20
type ListQuery struct {
	Page    int `form:"page" binding:"omitempty,min=1"`
	PerPage int `form:"per_page" binding:"omitempty,min=1,max=100"`

	Role             string `form:"role" binding:"omitempty,oneof=super_admin admin tutor student"`
	EducationLevelID string `form:"education_level_id" binding:"omitempty,uuid"` // jenjang
	GradeID          string `form:"grade_id" binding:"omitempty,uuid"`           // kelas
	BranchID         string `form:"branch_id" binding:"omitempty,uuid"`          // cabang
	Status           string `form:"status" binding:"omitempty,oneof=active inactive"`
	Search           string `form:"search" binding:"omitempty,max=100"` // nama, username, atau email
	// Filter paket akses (hasilnya hanya siswa): paket aktif hari ini = plan_id.
	PlanID string `form:"plan_id" binding:"omitempty,uuid"`
	// active = punya paket aktif, expiring = paket aktif berakhir ≤ 7 hari lagi, none = tanpa paket aktif.
	Subscription string `form:"subscription" binding:"omitempty,oneof=active expiring none"`
}

// RefResponse adalah pasangan id + nama.
type RefResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func toRef(r *usermodel.Ref) *RefResponse {
	if r == nil {
		return nil
	}
	return &RefResponse{ID: r.ID, Name: r.Name}
}

// UserResponse adalah 1 baris list user.
type UserResponse struct {
	ID             string       `json:"id"`
	Username       string       `json:"username"`
	Name           string       `json:"name"`
	Email          *string      `json:"email"`
	Phone          *string      `json:"phone"`
	Role           string       `json:"role"`
	Branch         *RefResponse `json:"branch"`          // null untuk super admin ("Semua cabang")
	EducationLevel *RefResponse `json:"education_level"` // jenjang (siswa)
	Grade          *RefResponse `json:"grade"`           // kelas (siswa)
	StudentCode    *string      `json:"student_code"`    // hanya siswa
	CoinBalance    *int         `json:"coin_balance"`    // hanya siswa
	Status         string       `json:"status"`
	JoinDate       string       `json:"join_date"` // tanggal bergabung "2026-01-15"
	EndDate        *string      `json:"end_date"`  // tanggal berakhir (inklusif); null = tanpa batas
	// true kalau end_date sudah lewat (WIB): akun tidak bisa login walaupun status active.
	Ended       bool       `json:"ended"`
	LastLoginAt *time.Time `json:"last_login_at"` // WIB
	CreatedAt   time.Time  `json:"created_at"`    // WIB
	// Akses hari ini (kolom Paket, Akses, Berlaku s/d); hanya siswa, null untuk role lain.
	Access *subscriptiondto.AccessResponse `json:"access"`
}

// ToUserResponse mengubah 1 user menjadi response.
func ToUserResponse(u usermodel.ListItem) UserResponse {
	r := UserResponse{
		ID: u.ID, Username: u.Username, Name: u.RealName, Email: u.Email, Phone: u.Phone, Role: u.Role,
		Branch: toRef(u.Branch), EducationLevel: toRef(u.EducationLevel), Grade: toRef(u.Grade),
		StudentCode: u.StudentCode, CoinBalance: u.CoinBalance, Status: u.Status,
		JoinDate: u.JoinDate.Format(DateLayout), Ended: isEnded(u.EndDate, time.Now()),
		CreatedAt: u.CreatedAt.In(clock.Jakarta),
	}
	if u.EndDate != nil {
		s := u.EndDate.Format(DateLayout)
		r.EndDate = &s
	}
	if u.LastLoginAt != nil {
		t := u.LastLoginAt.In(clock.Jakarta)
		r.LastLoginAt = &t
	}
	if u.Access != nil {
		a := subscriptiondto.ToAccessResponse(*u.Access)
		r.Access = &a
	}
	return r
}

// ToUserResponses mengubah list user menjadi response. Kosong = [].
func ToUserResponses(items []usermodel.ListItem) []UserResponse {
	result := make([]UserResponse, 0, len(items))
	for _, u := range items {
		result = append(result, ToUserResponse(u))
	}
	return result
}

// RoleCountsResponse adalah angka di tab menu User. Semua role selalu ada (0 kalau kosong).
type RoleCountsResponse struct {
	All        int `json:"all"`
	SuperAdmin int `json:"super_admin"`
	Admin      int `json:"admin"`
	Tutor      int `json:"tutor"`
	Student    int `json:"student"`
}

// ToRoleCountsResponse mengubah jumlah per role menjadi response.
func ToRoleCountsResponse(c usermodel.RoleCounts) RoleCountsResponse {
	return RoleCountsResponse{
		All:        c.SuperAdmin + c.Admin + c.Tutor + c.Student,
		SuperAdmin: c.SuperAdmin, Admin: c.Admin, Tutor: c.Tutor, Student: c.Student,
	}
}

// UserDetailResponse adalah data lengkap 1 user (response detail & ubah).
type UserDetailResponse struct {
	UserResponse
	ParentPhone  *string      `json:"parent_phone"`
	SchoolOrigin *string      `json:"school_origin"`
	DateOfBirth  *string      `json:"date_of_birth"` // "2012-05-17", hanya siswa
	TargetSchool *string      `json:"target_school"` // hanya siswa
	UpdatedBy    *RefResponse `json:"updated_by"`    // admin yang terakhir mengubah; null = belum pernah
	UpdatedAt    time.Time    `json:"updated_at"`    // WIB
}

// ToUserDetailResponse mengubah detail user menjadi response.
func ToUserDetailResponse(d usermodel.Detail) UserDetailResponse {
	r := UserDetailResponse{
		UserResponse: ToUserResponse(d.ListItem),
		ParentPhone:  d.ParentPhone, SchoolOrigin: d.SchoolOrigin, TargetSchool: d.TargetSchool,
		UpdatedBy: toRef(d.UpdatedBy), UpdatedAt: d.UpdatedAt.In(clock.Jakarta),
	}
	if d.DateOfBirth != nil {
		s := d.DateOfBirth.Format(DateLayout)
		r.DateOfBirth = &s
	}
	return r
}

// DateLayout adalah format tanggal lahir: "2012-05-17".
const DateLayout = "2006-01-02"

// UpdateRequest adalah body PUT /api/v1/users/:id (ganti penuh). Role tidak bisa diubah.
//
//	{
//	  "username": "siswa.budi", "name": "Budi Santoso", "email": null, "phone": "0812...",
//	  "parent_phone": "0813...", "school_origin": "SMPN 1 Surabaya",
//	  "branch_id": "...", "education_level_id": "...", "grade_id": "...", "status": "active",
//	  "password": "passwordBaru123", "date_of_birth": "2012-05-17", "target_school": "SMAN 5 Surabaya"
//	}
type UpdateRequest struct {
	Username     string  `json:"username" binding:"required,max=30"` // huruf kecil, angka, titik, garis bawah; 3-30
	Name         string  `json:"name" binding:"required,max=150"`
	Email        *string `json:"email" binding:"omitempty,max=255,email"`
	Phone        *string `json:"phone" binding:"omitempty,max=20"`
	ParentPhone  *string `json:"parent_phone" binding:"omitempty,max=20"`
	SchoolOrigin *string `json:"school_origin" binding:"omitempty,max=150"`
	// Cabang: wajib untuk siswa, tutor, admin; harus kosong untuk super admin (semua cabang).
	BranchID *string `json:"branch_id" binding:"omitempty,uuid"`
	// Jenjang & kelas: wajib untuk siswa (kelas harus milik jenjang); harus kosong untuk role lain.
	EducationLevelID *string `json:"education_level_id" binding:"omitempty,uuid"`
	GradeID          *string `json:"grade_id" binding:"omitempty,uuid"`
	Status           string  `json:"status" binding:"required,oneof=active inactive"`
	// Tanggal bergabung, wajib ("2026-01-15").
	JoinDate string `json:"join_date" binding:"required,datetime=2006-01-02"`
	// Tanggal berakhir, opsional, tidak boleh sebelum join_date. Lewat tanggal ini akun tidak bisa login.
	EndDate *string `json:"end_date" binding:"omitempty,datetime=2006-01-02"`
	// Opsional: diisi = reset password (8-72 karakter). Kosong / tidak dikirim = password tetap.
	Password *string `json:"password" binding:"omitempty,min=8,max=72"`
	// Khusus siswa (diabaikan untuk role lain).
	DateOfBirth  *string `json:"date_of_birth" binding:"omitempty,datetime=2006-01-02"`
	TargetSchool *string `json:"target_school" binding:"omitempty,max=150"`
}

// DeleteRequest adalah body DELETE /api/v1/users: hapus 1-100 user (soft delete).
type DeleteRequest struct {
	IDs []string `json:"ids" binding:"required,min=1,max=100,unique,dive,uuid"`
}

// DeleteResponse berisi ID user yang dihapus.
type DeleteResponse struct {
	DeletedIDs []string `json:"deleted_ids"`
}

// CreateRequest adalah body POST /api/v1/users: tambah 1-100 akun sekaligus (semua atau tidak sama
// sekali). Contoh:
//
//	{"users": [
//	  {"role": "student", "username": "siswa.andi", "name": "Andi Wijaya", "password": "password123",
//	   "branch_id": "...", "education_level_id": "...", "grade_id": "...", "parent_phone": "0813...",
//	   "school_origin": "SDN 1 Surabaya", "date_of_birth": "2014-02-01", "target_school": "SMPN 1 Surabaya"},
//	  {"role": "tutor", "username": "tutor.sari", "name": "Sari Dewi", "password": "password123", "branch_id": "..."}
//	]}
type CreateRequest struct {
	Users []CreateUserItem `json:"users" binding:"required,min=1,max=100,dive"`
}

// CreateUserItem adalah 1 akun baru. Aturan field sama dengan UpdateRequest, ditambah role & password
// wajib. status opsional (default active).
type CreateUserItem struct {
	Role         string  `json:"role" binding:"required,oneof=super_admin admin tutor student"`
	Username     string  `json:"username" binding:"required,max=30"`
	Name         string  `json:"name" binding:"required,max=150"`
	Password     string  `json:"password" binding:"required,min=8,max=72"`
	Email        *string `json:"email" binding:"omitempty,max=255,email"`
	Phone        *string `json:"phone" binding:"omitempty,max=20"`
	ParentPhone  *string `json:"parent_phone" binding:"omitempty,max=20"`
	SchoolOrigin *string `json:"school_origin" binding:"omitempty,max=150"`
	// Cabang: wajib untuk siswa, tutor, admin; harus kosong untuk super admin.
	BranchID *string `json:"branch_id" binding:"omitempty,uuid"`
	// Jenjang & kelas: wajib untuk siswa; harus kosong untuk role lain.
	EducationLevelID *string `json:"education_level_id" binding:"omitempty,uuid"`
	GradeID          *string `json:"grade_id" binding:"omitempty,uuid"`
	Status           string  `json:"status" binding:"omitempty,oneof=active inactive"`
	// Tanggal bergabung, wajib ("2026-01-15").
	JoinDate string `json:"join_date" binding:"required,datetime=2006-01-02"`
	// Tanggal berakhir, opsional, tidak boleh sebelum join_date. Lewat tanggal ini akun tidak bisa login.
	EndDate *string `json:"end_date" binding:"omitempty,datetime=2006-01-02"`
	// Khusus siswa.
	DateOfBirth  *string `json:"date_of_birth" binding:"omitempty,datetime=2006-01-02"`
	TargetSchool *string `json:"target_school" binding:"omitempty,max=150"`
}

// isEnded = true kalau tanggal berakhir sudah lewat pada now (WIB). Sama dengan aturan login.
func isEnded(endDate *time.Time, now time.Time) bool {
	return endDate != nil && endDate.Format(DateLayout) < now.In(clock.Jakarta).Format(DateLayout)
}

// ImportCheckResponse adalah hasil POST /api/v1/users/import/check yang valid. "users" bisa langsung
// dikirim FE sebagai body POST /api/v1/users: {"users": data.users} (field row & *_name diabaikan).
type ImportCheckResponse struct {
	Total int                    `json:"total"`
	Users []ImportedUserResponse `json:"users"`
}

// ImportedUserResponse adalah 1 baris Excel yang valid (bentuk sama dengan CreateUserItem + preview).
type ImportedUserResponse struct {
	Row                int     `json:"row"` // nomor baris di Excel
	Role               string  `json:"role"`
	Username           string  `json:"username"`
	Name               string  `json:"name"`
	Password           string  `json:"password"`
	Email              *string `json:"email"`
	Phone              *string `json:"phone"`
	ParentPhone        *string `json:"parent_phone"`
	BranchID           *string `json:"branch_id"`
	BranchName         *string `json:"branch_name"`
	EducationLevelID   *string `json:"education_level_id"`
	EducationLevelName *string `json:"education_level_name"`
	GradeID            *string `json:"grade_id"`
	GradeName          *string `json:"grade_name"`
	SchoolOrigin       *string `json:"school_origin"`
	JoinDate           string  `json:"join_date"`
	EndDate            *string `json:"end_date"`
	Status             string  `json:"status"`
	DateOfBirth        *string `json:"date_of_birth"`
	TargetSchool       *string `json:"target_school"`
}

// ToImportCheckResponse mengubah hasil cek Excel menjadi response.
func ToImportCheckResponse(items []usermodel.ImportedUser) ImportCheckResponse {
	r := ImportCheckResponse{Total: len(items), Users: make([]ImportedUserResponse, 0, len(items))}
	for _, u := range items {
		r.Users = append(r.Users, ImportedUserResponse{
			Row: u.Row, Role: u.Role, Username: u.Username, Name: u.Name, Password: u.Password,
			Email: u.Email, Phone: u.Phone, ParentPhone: u.ParentPhone,
			BranchID: u.BranchID, BranchName: u.BranchName,
			EducationLevelID: u.EducationLevelID, EducationLevelName: u.EducationLevelName,
			GradeID: u.GradeID, GradeName: u.GradeName, SchoolOrigin: u.SchoolOrigin,
			JoinDate: u.JoinDate, EndDate: u.EndDate, Status: u.Status,
			DateOfBirth: u.DateOfBirth, TargetSchool: u.TargetSchool,
		})
	}
	return r
}
