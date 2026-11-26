// Package announcement handles announcement and student notification endpoints.
package announcement

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	announcementdto "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/dto/notification/announcement"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/middleware"
	announcementmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/notification/announcement"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/response"
	announcementservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/notification/announcement"
)

type AnnouncementHandler struct {
	service *announcementservice.AnnouncementService
}

func NewAnnouncementHandler(service *announcementservice.AnnouncementService) *AnnouncementHandler {
	return &AnnouncementHandler{service: service}
}

func (h *AnnouncementHandler) ListAdmin(c *gin.Context) {
	var query announcementdto.ListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.ValidationError(c, err)
		return
	}
	query.SetDefaults()

	items, total, err := h.service.ListAdmin(c.Request.Context(), query.Page, query.PerPage, query.Search, query.Status)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.SuccessWithMeta(c, "Daftar pengumuman", announcementdto.ToResponses(items),
		response.Meta{Page: query.Page, PerPage: query.PerPage, Total: int(total)})
}

func (h *AnnouncementHandler) GetAdmin(c *gin.Context) {
	item, err := h.service.GetAdminByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Detail pengumuman", announcementdto.ToResponse(item))
}

func (h *AnnouncementHandler) Create(c *gin.Context) {
	var request announcementdto.CreateRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		response.ValidationError(c, err)
		return
	}
	item, err := h.service.Create(
		c.Request.Context(),
		request.Title,
		request.Body,
		request.GradeIDs,
		request.Status,
		request.PublishAt,
		middleware.GetUserID(c),
	)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusCreated, "Pengumuman berhasil dibuat", announcementdto.ToResponse(item))
}

func (h *AnnouncementHandler) Update(c *gin.Context) {
	var request announcementdto.UpdateRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		response.ValidationError(c, err)
		return
	}
	item, err := h.service.Update(
		c.Request.Context(),
		c.Param("id"),
		request.Title,
		request.Body,
		request.GradeIDs,
		request.Status,
		request.PublishAt,
		middleware.GetUserID(c),
	)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Pengumuman berhasil diperbarui", announcementdto.ToResponse(item))
}

func (h *AnnouncementHandler) Delete(c *gin.Context) {
	if err := h.service.Delete(c.Request.Context(), c.Param("id")); err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Pengumuman berhasil dihapus", nil)
}

func (h *AnnouncementHandler) ListForStudent(c *gin.Context) {
	var query announcementdto.StudentListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.ValidationError(c, err)
		return
	}
	query.SetDefaults()

	items, total, err := h.service.ListForStudent(
		c.Request.Context(), middleware.GetUserID(c), query.Page, query.PerPage,
	)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.SuccessWithMeta(c, "Notifikasi", announcementdto.ToStudentResponses(items),
		response.Meta{Page: query.Page, PerPage: query.PerPage, Total: int(total)})
}

func (h *AnnouncementHandler) UnreadCount(c *gin.Context) {
	count, err := h.service.UnreadCount(c.Request.Context(), middleware.GetUserID(c))
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Jumlah notifikasi belum dibaca", announcementdto.UnreadCountResponse{Count: count})
}

func (h *AnnouncementHandler) MarkRead(c *gin.Context) {
	if err := h.service.MarkRead(c.Request.Context(), c.Param("id"), middleware.GetUserID(c)); err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Notifikasi ditandai sudah dibaca", gin.H{"is_read": true})
}

func (h *AnnouncementHandler) MarkAllRead(c *gin.Context) {
	count, err := h.service.MarkAllRead(c.Request.Context(), middleware.GetUserID(c))
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Semua notifikasi ditandai sudah dibaca",
		announcementdto.MarkAllReadResponse{MarkedCount: count})
}

func (h *AnnouncementHandler) handleError(c *gin.Context, err error) {
	_ = c.Error(err)

	var validationErr *announcementmodel.ValidationError
	switch {
	case errors.As(err, &validationErr):
		response.ErrorWithFields(c, http.StatusUnprocessableEntity, "Validasi gagal", []response.FieldError{
			{Field: validationErr.Field, Message: validationErr.Message},
		})
	case errors.Is(err, announcementmodel.ErrNotFound):
		response.Error(c, http.StatusNotFound, "Pengumuman tidak ditemukan")
	default:
		log.Printf("error tidak terduga di %s %s: %v", c.Request.Method, c.Request.URL.Path, err)
		response.Error(c, http.StatusInternalServerError, "Terjadi kesalahan pada server")
	}
}
