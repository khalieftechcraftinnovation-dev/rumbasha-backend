// Package passwordreset (dto) berisi bentuk request dan response JSON "Lupa kata sandi" (form publik)
// dan daftar permintaan untuk admin.
//
// Di package lain, import dengan alias passwordresetdto:
//
//	passwordresetdto "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/dto/auth/passwordreset"
package passwordreset

import (
	"time"

	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/clock"
	passwordresetmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/auth/passwordreset"
)

// ForgotRequest adalah body POST /api/v1/auth/forgot-password (tanpa login).
//
//	{"identifier": "siswa.budi", "contact": "0812-3456-7890", "note": "Lupa sandi setelah ganti HP"}
type ForgotRequest struct {
	Identifier string  `json:"identifier" binding:"required,max=255"` // username, email, atau no HP akun
	Contact    *string `json:"contact" binding:"omitempty,max=100"`   // no HP / WA yang bisa dihubungi admin
	Note       *string `json:"note" binding:"omitempty,max=500"`      // pesan untuk admin
}

// ListQuery adalah query string GET /api/v1/password-reset-requests.
type ListQuery struct {
	Status  string `form:"status" binding:"omitempty,oneof=pending resolved rejected all"` // default pending
	Search  string `form:"search" binding:"omitempty,max=100"`
	Page    int    `form:"page" binding:"omitempty,min=1"`
	PerPage int    `form:"per_page" binding:"omitempty,min=1,max=100"`
}

// ResolveRequest adalah body POST /api/v1/password-reset-requests/:id/resolve.
type ResolveRequest struct {
	NewPassword string  `json:"new_password" binding:"required,min=8,max=72"`
	Note        *string `json:"note" binding:"omitempty,max=500"`
}

// RejectRequest adalah body POST /api/v1/password-reset-requests/:id/reject (opsional).
type RejectRequest struct {
	Note *string `json:"note" binding:"omitempty,max=500"`
}

// RefResponse adalah pasangan id + nama.
type RefResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func toRef(r *passwordresetmodel.Ref) *RefResponse {
	if r == nil {
		return nil
	}
	return &RefResponse{ID: r.ID, Name: r.Name}
}

// UserResponse = akun pemilik permintaan (data untuk menghubungi user).
type UserResponse struct {
	ID          string       `json:"id"`
	Username    string       `json:"username"`
	Name        string       `json:"name"`
	Role        string       `json:"role"`
	Status      string       `json:"status"`
	Email       *string      `json:"email"`
	Phone       *string      `json:"phone"`
	ParentPhone *string      `json:"parent_phone"`
	Branch      *RefResponse `json:"branch"`
}

// RequestResponse adalah 1 permintaan lupa kata sandi.
type RequestResponse struct {
	ID             string       `json:"id"`
	User           UserResponse `json:"user"`
	Identifier     string       `json:"identifier"` // yang diketik user di form
	Contact        *string      `json:"contact"`
	Note           *string      `json:"note"`
	RequestCount   int          `json:"request_count"` // berapa kali dikirim selama menunggu
	Status         string       `json:"status"`        // pending, resolved, rejected
	RequestedAt    time.Time    `json:"requested_at"`  // permintaan terakhir (WIB)
	ResolvedAt     *time.Time   `json:"resolved_at"`
	ResolvedBy     *RefResponse `json:"resolved_by"`
	ResolutionNote *string      `json:"resolution_note"`
	CreatedAt      time.Time    `json:"created_at"`
}

// ToRequestResponse mengubah 1 permintaan menjadi response.
func ToRequestResponse(r passwordresetmodel.Request) RequestResponse {
	u := r.User
	res := RequestResponse{
		ID: r.ID, Identifier: r.Identifier, Contact: r.Contact, Note: r.Note, RequestCount: r.RequestCount,
		Status: r.Status, RequestedAt: r.RequestedAt.In(clock.Jakarta), ResolvedBy: toRef(r.ResolvedBy),
		ResolutionNote: r.ResolutionNote, CreatedAt: r.CreatedAt.In(clock.Jakarta),
		User: UserResponse{
			ID: u.ID, Username: u.Username, Name: u.RealName, Role: u.Role, Status: u.Status,
			Email: u.Email, Phone: u.Phone, ParentPhone: u.ParentPhone, Branch: toRef(u.Branch),
		},
	}
	if r.ResolvedAt != nil {
		t := r.ResolvedAt.In(clock.Jakarta)
		res.ResolvedAt = &t
	}
	return res
}

// ToRequestResponses mengubah daftar permintaan menjadi response. Kosong = [].
func ToRequestResponses(items []passwordresetmodel.Request) []RequestResponse {
	result := make([]RequestResponse, 0, len(items))
	for _, r := range items {
		result = append(result, ToRequestResponse(r))
	}
	return result
}

// CountsResponse = jumlah permintaan per status (badge menu = pending).
type CountsResponse struct {
	Pending  int `json:"pending"`
	Resolved int `json:"resolved"`
	Rejected int `json:"rejected"`
}
