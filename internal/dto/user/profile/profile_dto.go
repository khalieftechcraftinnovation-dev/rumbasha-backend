// Package profile (dto) berisi bentuk request dan response JSON halaman Profil siswa.
//
// Di package lain, import dengan alias profiledto:
//
//	profiledto "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/dto/user/profile"
package profile

import (
	"time"

	subscriptiondto "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/dto/billing/subscription"
	subscriptionmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/billing/subscription"
	profilemodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/user/profile"
)

// DateLayout = format tanggal "2012-05-17".
const DateLayout = "2006-01-02"

// UpdateRequest adalah body PUT /api/v1/student/profile (Data diri, ganti penuh).
//
//	{"name": "Citra Lestari", "email": "citra@example.com", "phone": "0812-3456-7890", "date_of_birth": "2015-03-02"}
type UpdateRequest struct {
	Name        string  `json:"name" binding:"required,max=150"`
	Email       *string `json:"email" binding:"omitempty,max=255,email"`               // null / "" = dikosongkan
	Phone       *string `json:"phone" binding:"omitempty,max=20"`                      // null / "" = dikosongkan
	DateOfBirth *string `json:"date_of_birth" binding:"omitempty,datetime=2006-01-02"` // null = dikosongkan
}

// ParentContactRequest adalah body PUT /api/v1/student/profile/parent-contact.
type ParentContactRequest struct {
	ParentPhone *string `json:"parent_phone" binding:"omitempty,max=20"` // null / "" = dikosongkan
}

// RefResponse adalah pasangan id + nama.
type RefResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func toRef(r *profilemodel.Ref) *RefResponse {
	if r == nil {
		return nil
	}
	return &RefResponse{ID: r.ID, Name: r.Name}
}

// ProfileResponse adalah data diri siswa.
type ProfileResponse struct {
	ID                string       `json:"id"`
	Username          string       `json:"username"`
	Name              string       `json:"name"`
	StudentCode       *string      `json:"student_code"`
	Email             *string      `json:"email"`
	Phone             *string      `json:"phone"`
	ParentPhone       *string      `json:"parent_phone"`        // untuk form edit
	ParentPhoneMasked *string      `json:"parent_phone_masked"` // untuk ditampilkan, contoh "0812-xxxx-2210"
	DateOfBirth       *string      `json:"date_of_birth"`
	SchoolOrigin      *string      `json:"school_origin"` // diatur admin
	TargetSchool      *string      `json:"target_school"` // diatur admin, contoh "SMAN 5 Surabaya"
	EducationLevel    *RefResponse `json:"education_level"`
	Grade             *RefResponse `json:"grade"`
	Branch            *RefResponse `json:"branch"`
	AvatarURL         *string      `json:"avatar_url"` // null = belum ada foto
	JoinDate          string       `json:"join_date"`
	EndDate           *string      `json:"end_date"`
	// Field yang boleh diubah siswa sendiri (FE memakai ini untuk mengunci input lain).
	EditableFields []string `json:"editable_fields"`
}

// EditableFields = field profil yang boleh diubah siswa sendiri (keputusan BD-014).
var EditableFields = []string{"name", "email", "phone", "date_of_birth", "avatar", "parent_phone"}

// ToProfileResponse mengubah profil menjadi response.
func ToProfileResponse(p profilemodel.Profile) ProfileResponse {
	r := ProfileResponse{
		ID: p.ID, Username: p.Username, Name: p.RealName, StudentCode: p.StudentCode,
		Email: p.Email, Phone: p.Phone, ParentPhone: p.ParentPhone,
		SchoolOrigin: p.SchoolOrigin, TargetSchool: p.TargetSchool,
		EducationLevel: toRef(p.EducationLevel), Grade: toRef(p.Grade), Branch: toRef(p.Branch),
		AvatarURL: p.AvatarURL, JoinDate: p.JoinDate.Format(DateLayout), EditableFields: EditableFields,
	}
	if p.ParentPhone != nil {
		masked := profilemodel.MaskPhone(*p.ParentPhone)
		r.ParentPhoneMasked = &masked
	}
	if p.DateOfBirth != nil {
		s := p.DateOfBirth.Format(DateLayout)
		r.DateOfBirth = &s
	}
	if p.EndDate != nil {
		s := p.EndDate.Format(DateLayout)
		r.EndDate = &s
	}
	return r
}

// SummaryResponse adalah GET /api/v1/student/profile: data diri + paket + ringkasan (FR-PRO-001).
type SummaryResponse struct {
	Profile ProfileResponse                `json:"profile"`
	Access  subscriptiondto.AccessResponse `json:"access"` // paket aktif, berlaku s/d, sisa hari, fitur
	Coin    int                            `json:"coin_balance"`
	Stats   StatsResponse                  `json:"stats"`
}

// StatsResponse = statistik di halaman Profil.
type StatsResponse struct {
	Month            string    `json:"month"`              // bulan WIB, contoh "2026-11"
	DailyRank        *RankInfo `json:"daily_quiz_rank"`    // null = belum ikut Daily Quiz bulan ini
	PointsThisMonth  int       `json:"points_this_month"`  // point Daily Quiz bulan ini
	TotalPoints      int       `json:"total_points"`       // semua point (Daily Quiz + Uji Kom + Tryout)
	QuizzesThisMonth int       `json:"quizzes_this_month"` // quiz yang dikumpulkan bulan ini
}

// RankInfo = peringkat Daily Quiz bulan ini di kelasnya, contoh "#12 dari 47 siswa".
type RankInfo struct {
	Rank  int `json:"rank"`
	Total int `json:"total"`
}

// ToSummaryResponse menyusun response halaman Profil.
func ToSummaryResponse(p profilemodel.Profile, access subscriptionmodel.Access, s profilemodel.Stats) SummaryResponse {
	r := SummaryResponse{
		Profile: ToProfileResponse(p), Access: subscriptiondto.ToAccessResponse(access), Coin: p.CoinBalance,
		Stats: StatsResponse{
			Month: s.Month, PointsThisMonth: s.PointsThisMonth, TotalPoints: s.TotalPoints, QuizzesThisMonth: s.QuizzesThisMonth,
		},
	}
	if s.DailyRank != nil {
		r.Stats.DailyRank = &RankInfo{Rank: *s.DailyRank, Total: s.DailyRankTotal}
	}
	return r
}

// ParseDate membaca tanggal "2012-05-17" (format sudah dicek binding).
func ParseDate(s string) time.Time {
	t, _ := time.Parse(DateLayout, s)
	return t
}
