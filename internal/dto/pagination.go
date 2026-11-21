// Package dto (Data Transfer Object) berisi bentuk request dan response JSON.
//
// File di folder ini (root) dipakai BERSAMA oleh semua fitur, misalnya pagination.
// DTO khusus satu fitur ada di subfolder fiturnya: dto/login, dto/subject, ...
package dto

// PaginationQuery adalah query string ?page=1&per_page=20 untuk endpoint list.
// Tag form:"..." dipakai untuk query string (bukan body JSON).
type PaginationQuery struct {
	Page    int `form:"page" binding:"omitempty,min=1"`
	PerPage int `form:"per_page" binding:"omitempty,min=1,max=100"`
}

// SetDefaults mengisi nilai default kalau client tidak mengirim page/per_page.
func (q *PaginationQuery) SetDefaults() {
	if q.Page == 0 {
		q.Page = 1
	}
	if q.PerPage == 0 {
		q.PerPage = 20
	}
}
