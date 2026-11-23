// Package login (dto) berisi bentuk request dan response JSON untuk fitur login.
//
// Di package lain, import dengan alias logindto:
//
//	logindto "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/dto/login"
package login

import (
	"time"

	loginmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/login"
	subscriptionmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/subscription"
)

// LoginRequest adalah body JSON untuk POST /api/v1/auth/login.
type LoginRequest struct {
	Username string `json:"username" binding:"required,max=30"`
	Password string `json:"password" binding:"required,max=72"` // bcrypt hanya membaca 72 byte pertama
}

// RefreshTokenRequest adalah body JSON untuk /auth/refresh dan /auth/logout.
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// SessionResponse dikirim setelah login atau refresh berhasil.
type SessionResponse struct {
	AccessToken           string       `json:"access_token"`
	AccessTokenExpiresAt  time.Time    `json:"access_token_expires_at"`
	RefreshToken          string       `json:"refresh_token"`
	RefreshTokenExpiresAt time.Time    `json:"refresh_token_expires_at"` // akhir sesi: tanggal 1 bulan depan, 00:00 WIB
	User                  UserResponse `json:"user"`
}

// UserResponse adalah data user yang aman dikirim ke client (tanpa password_hash).
type UserResponse struct {
	ID             string              `json:"id"`
	Role           string              `json:"role"`
	Username       string              `json:"username"`
	RealName       string              `json:"real_name"`
	Email          *string             `json:"email"`
	Phone          *string             `json:"phone"`
	ParentPhone    *string             `json:"parent_phone"`
	SchoolOrigin   *string             `json:"school_origin"`
	BranchID       *string             `json:"branch_id"`
	EducationLevel *MasterDataResponse `json:"education_level"` // jenjang; null kalau tidak ada
	Grade          *MasterDataResponse `json:"grade"`           // kelas; null kalau tidak ada
	LastLoginAt    *time.Time          `json:"last_login_at"`
}

// MasterDataResponse adalah satu item dari tabel master_data_items (contoh: jenjang SMP, Kelas 7).
type MasterDataResponse struct {
	ID   string  `json:"id"`
	Type string  `json:"type"` // education_level atau grade
	Code *string `json:"code"` // contoh: SMP, GRADE_7. Boleh null untuk item buatan admin
	Name string  `json:"name"` // contoh: SMP, Kelas 7
}

// MeResponse adalah response GET /api/v1/auth/me.
type MeResponse struct {
	User UserResponse `json:"user"`
	// Access hanya untuk siswa; null untuk admin/tutor.
	Access *AccessResponse `json:"access"`
}

// AccessResponse adalah paket aktif siswa hari ini beserta fitur yang terbuka.
type AccessResponse struct {
	Subscription *SubscriptionResponse `json:"subscription"` // null = tidak ada paket aktif hari ini
	Features     map[string]bool       `json:"features"`
}

// SubscriptionResponse adalah info paket aktif siswa.
type SubscriptionResponse struct {
	PlanName  string `json:"plan_name"`
	Channel   string `json:"channel"`    // offline atau online
	StartDate string `json:"start_date"` // format 2026-11-01
	EndDate   string `json:"end_date"`   // hari terakhir paket aktif
}

// ToSessionResponse mengubah hasil login/refresh menjadi SessionResponse.
func ToSessionResponse(accessToken string, accessExpiresAt time.Time, refreshToken string, refreshExpiresAt time.Time, user loginmodel.UserDetail) SessionResponse {
	return SessionResponse{
		AccessToken:           accessToken,
		AccessTokenExpiresAt:  accessExpiresAt,
		RefreshToken:          refreshToken,
		RefreshTokenExpiresAt: refreshExpiresAt,
		User:                  ToUserResponse(user),
	}
}

// ToUserResponse mengubah loginmodel.UserDetail menjadi UserResponse.
func ToUserResponse(user loginmodel.UserDetail) UserResponse {
	return UserResponse{
		ID:             user.ID,
		Role:           user.Role,
		Username:       user.Username,
		RealName:       user.RealName,
		Email:          user.Email,
		Phone:          user.Phone,
		ParentPhone:    user.ParentPhone,
		SchoolOrigin:   user.SchoolOrigin,
		BranchID:       user.BranchID,
		EducationLevel: toMasterDataResponse(user.EducationLevelID, user.EducationLevelType, user.EducationLevelCode, user.EducationLevelName),
		Grade:          toMasterDataResponse(user.GradeID, user.GradeType, user.GradeCode, user.GradeName),
		LastLoginAt:    user.LastLoginAt,
	}
}

// toMasterDataResponse menyusun MasterDataResponse dari kolom hasil JOIN.
// Kalau id-nya nil (user tidak punya jenjang/kelas), hasilnya nil → null di JSON.
func toMasterDataResponse(id, itemType, code, name *string) *MasterDataResponse {
	if id == nil || itemType == nil || name == nil {
		return nil
	}
	return &MasterDataResponse{
		ID:   *id,
		Type: *itemType,
		Code: code,
		Name: *name,
	}
}

// ToSubscriptionResponse mengubah paket aktif menjadi SubscriptionResponse (nil tetap nil).
func ToSubscriptionResponse(subscription *subscriptionmodel.ActiveSubscription) *SubscriptionResponse {
	if subscription == nil {
		return nil
	}
	return &SubscriptionResponse{
		PlanName:  subscription.PlanName,
		Channel:   subscription.Channel,
		StartDate: subscription.StartDate.Format(time.DateOnly),
		EndDate:   subscription.EndDate.Format(time.DateOnly),
	}
}
