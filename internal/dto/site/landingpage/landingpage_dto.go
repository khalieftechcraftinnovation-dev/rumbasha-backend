// Package landingpage (dto) berisi bentuk request dan response JSON fitur Landing Page.
//
// Di package lain, import dengan alias landingpagedto:
//
//	landingpagedto "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/dto/site/landingpage"
package landingpage

import (
	"encoding/json"
	"time"

	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/clock"
	landingpagemodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/site/landingpage"
)

// UpdateContentRequest adalah body PUT /landing-pages/:page/sections/:key.
// content = isi section (object JSON, bentuknya ditentukan FE per section, maks 200 KB).
type UpdateContentRequest struct {
	Content json.RawMessage `json:"content" binding:"required"`
}

// SetVisibleRequest adalah body PATCH /landing-pages/:page/sections/:key/visibility (saklar "Tampil").
type SetVisibleRequest struct {
	IsVisible *bool `json:"is_visible" binding:"required"`
}

// ReorderRequest adalah body PUT /landing-pages/:page/sections/order: SEMUA key section, urut baru.
type ReorderRequest struct {
	Keys []string `json:"keys" binding:"required,min=1,max=50,dive,required,max=30"`
}

// UserRef = user pelaku (id + nama). null kalau tidak ada.
type UserRef struct {
	ID   string  `json:"id"`
	Name *string `json:"name"` // null kalau akun sudah dihapus
}

func toUserRef(id, name *string) *UserRef {
	if id == nil {
		return nil
	}
	return &UserRef{ID: *id, Name: name}
}

func toWIB(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	w := t.In(clock.Jakarta)
	return &w
}

// PageResponse = 1 tab Landing Page.
type PageResponse struct {
	Code         string     `json:"code"` // home, pricing, achievers
	Name         string     `json:"name"`
	PublishedAt  *time.Time `json:"published_at"` // WIB; null = belum pernah dipublish
	PublishedBy  *UserRef   `json:"published_by"`
	PendingCount int        `json:"pending_count"` // jumlah perubahan belum dipublish
}

// ToPageResponse mengubah model halaman menjadi response.
func ToPageResponse(p landingpagemodel.Page) PageResponse {
	return PageResponse{
		Code: p.Code, Name: p.Name, PublishedAt: toWIB(p.PublishedAt),
		PublishedBy: toUserRef(p.PublishedBy, p.PublishedByName), PendingCount: p.PendingCount,
	}
}

// ToPageResponses mengubah daftar halaman menjadi response.
func ToPageResponses(pages []landingpagemodel.Page) []PageResponse {
	result := make([]PageResponse, 0, len(pages))
	for _, p := range pages {
		result = append(result, ToPageResponse(p))
	}
	return result
}

// SectionResponse = 1 section draft.
type SectionResponse struct {
	Key       string          `json:"key"` // kode tetap, contoh hero, faq
	Name      string          `json:"name"`
	Content   json.RawMessage `json:"content"`
	IsVisible bool            `json:"is_visible"`
	SortOrder int             `json:"sort_order"`
	UpdatedBy *UserRef        `json:"updated_by"`
	UpdatedAt time.Time       `json:"updated_at"` // WIB
}

// ChangeResponse = 1 perubahan belum dipublish.
type ChangeResponse struct {
	SectionKey  *string   `json:"section_key"` // null = perubahan halaman (urutan section)
	Description string    `json:"description"` // contoh "Hero · isi diubah"
	ChangedBy   *UserRef  `json:"changed_by"`
	ChangedAt   time.Time `json:"changed_at"` // WIB
}

// DraftResponse adalah data GET /landing-pages/:page (panel admin).
type DraftResponse struct {
	Page           PageResponse      `json:"page"`
	Sections       []SectionResponse `json:"sections"`
	PendingChanges []ChangeResponse  `json:"pending_changes"`
}

// ToDraftResponse menyusun data panel admin 1 halaman.
func ToDraftResponse(page landingpagemodel.Page, sections []landingpagemodel.Section, changes []landingpagemodel.Change) DraftResponse {
	res := DraftResponse{
		Page:           ToPageResponse(page),
		Sections:       make([]SectionResponse, 0, len(sections)),
		PendingChanges: make([]ChangeResponse, 0, len(changes)),
	}
	for _, s := range sections {
		res.Sections = append(res.Sections, SectionResponse{
			Key: s.Key, Name: s.Name, Content: s.Content, IsVisible: s.IsVisible, SortOrder: s.SortOrder,
			UpdatedBy: toUserRef(s.UpdatedBy, s.UpdatedByName), UpdatedAt: s.UpdatedAt.In(clock.Jakarta),
		})
	}
	for _, c := range changes {
		res.PendingChanges = append(res.PendingChanges, ChangeResponse{
			SectionKey: c.SectionKey, Description: c.Description,
			ChangedBy: toUserRef(c.ChangedBy, c.ChangedByName), ChangedAt: c.ChangedAt.In(clock.Jakarta),
		})
	}
	return res
}

// PublishResponse adalah data POST /landing-pages/:page/publish.
type PublishResponse struct {
	PublishedAt time.Time `json:"published_at"` // WIB
	Changes     []string  `json:"changes"`      // perubahan yang ikut dipublish
}

// ToPublishResponse mengubah hasil publish menjadi response.
func ToPublishResponse(r landingpagemodel.PublishResult) PublishResponse {
	changes := r.Changes
	if changes == nil {
		changes = []string{}
	}
	return PublishResponse{PublishedAt: r.PublishedAt.In(clock.Jakarta), Changes: changes}
}

// PublicPageResponse adalah data GET /public/landing-pages/:page (situs publik, tanpa login).
// Hanya section yang tampil saat publish terakhir, urut tampil.
type PublicPageResponse struct {
	Code        string                              `json:"code"`
	Name        string                              `json:"name"`
	Sections    []landingpagemodel.PublishedSection `json:"sections"`     // kosong kalau belum pernah dipublish
	PublishedAt *time.Time                          `json:"published_at"` // WIB
}

// ToPublicPageResponse mengubah isi halaman publik menjadi response.
func ToPublicPageResponse(p landingpagemodel.PublishedPage) PublicPageResponse {
	return PublicPageResponse{Code: p.Code, Name: p.Name, Sections: p.Sections, PublishedAt: toWIB(p.PublishedAt)}
}
