// Package subscription (handler) menangani endpoint paket akses: CRUD paket (admin), daftar paket
// untuk siswa, dan pengaturan akses siswa (beri paket, batal, akses khusus).
//
// Di package lain, import dengan alias subscriptionhandler:
//
//	subscriptionhandler "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/handler/subscription"
package subscription

import (
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/clock"
	subscriptiondto "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/dto/subscription"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/middleware"
	loginmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/login"
	subscriptionmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/subscription"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/response"
	subscriptionservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/subscription"
)

// SubscriptionHandler menangani endpoint paket akses.
type SubscriptionHandler struct {
	service *subscriptionservice.SubscriptionService
}

// NewSubscriptionHandler membuat SubscriptionHandler. Dipanggil sekali di main.go.
func NewSubscriptionHandler(service *subscriptionservice.SubscriptionService) *SubscriptionHandler {
	return &SubscriptionHandler{service: service}
}

func today() string {
	return clock.TodayInJakarta(time.Now())
}

// ListPlans menangani GET /api/v1/subscription-plans.
// Siswa: akses hari ini + paket aktif untuk jenjangnya (halaman Paket & Coin), tanpa data admin.
// Admin & super admin: semua paket (filter status, jenjang, nama) + jumlah pengguna & modul.
func (h *SubscriptionHandler) ListPlans(c *gin.Context) {
	if middleware.GetRole(c) == loginmodel.RoleStudent {
		access, plans, err := h.service.StudentPlans(c.Request.Context(), middleware.GetUserID(c))
		if err != nil {
			h.handleError(c, err)
			return
		}
		response.Success(c, http.StatusOK, "Daftar paket akses", subscriptiondto.ToStudentPlansResponse(access, plans))
		return
	}

	var q subscriptiondto.ListPlansQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.ValidationError(c, err)
		return
	}
	plans, err := h.service.ListPlans(c.Request.Context(), q)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Daftar paket akses", subscriptiondto.ToPlanResponses(plans))
}

// GetPlan menangani GET /api/v1/subscription-plans/:id (admin & super admin).
func (h *SubscriptionHandler) GetPlan(c *gin.Context) {
	p, err := h.service.GetPlan(c.Request.Context(), c.Param("id"))
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Detail paket akses", subscriptiondto.ToPlanResponse(p))
}

// CreatePlan menangani POST /api/v1/subscription-plans (admin & super admin).
func (h *SubscriptionHandler) CreatePlan(c *gin.Context) {
	var req subscriptiondto.PlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err)
		return
	}
	p, err := h.service.CreatePlan(c.Request.Context(), req, middleware.GetUserID(c))
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusCreated, "Paket akses berhasil dibuat", subscriptiondto.ToPlanResponse(p))
}

// UpdatePlan menangani PUT /api/v1/subscription-plans/:id (admin & super admin, ganti penuh).
func (h *SubscriptionHandler) UpdatePlan(c *gin.Context) {
	var req subscriptiondto.PlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err)
		return
	}
	p, err := h.service.UpdatePlan(c.Request.Context(), c.Param("id"), req, middleware.GetUserID(c))
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Paket akses berhasil diubah", subscriptiondto.ToPlanResponse(p))
}

// DeletePlans menangani DELETE /api/v1/subscription-plans, body {"ids": [...]} (soft delete).
func (h *SubscriptionHandler) DeletePlans(c *gin.Context) {
	var req subscriptiondto.DeletePlansRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err)
		return
	}
	ids, err := h.service.DeletePlans(c.Request.Context(), req.IDs, middleware.GetUserID(c))
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Paket akses berhasil dihapus", subscriptiondto.DeletePlansResponse{DeletedIDs: ids})
}

// StudentAccess menangani GET /api/v1/students/:id/access: akses hari ini, rincian per fitur, dan
// riwayat paket 1 siswa (form edit akses di menu User).
func (h *SubscriptionHandler) StudentAccess(c *gin.Context) {
	student, access, subs, err := h.service.StudentAccess(c.Request.Context(), c.Param("id"))
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Akses siswa", subscriptiondto.ToStudentAccessResponse(student, access, subs, today()))
}

// Assign menangani POST /api/v1/student-subscriptions: beri 1 paket ke 1-100 siswa sekaligus.
func (h *SubscriptionHandler) Assign(c *gin.Context) {
	var req subscriptiondto.AssignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err)
		return
	}
	plan, results, err := h.service.Assign(c.Request.Context(), req, middleware.GetUserID(c))
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusCreated, "Paket akses berhasil diberikan", subscriptiondto.ToAssignResponse(plan, req.Period, results, today()))
}

// Cancel menangani POST /api/v1/student-subscriptions/:id/cancel: batalkan 1 paket siswa.
func (h *SubscriptionHandler) Cancel(c *gin.Context) {
	var req subscriptiondto.CancelRequest
	if c.Request.ContentLength > 0 { // body opsional
		if err := c.ShouldBindJSON(&req); err != nil {
			response.ValidationError(c, err)
			return
		}
	}
	sub, err := h.service.CancelSubscription(c.Request.Context(), c.Param("id"), req, middleware.GetUserID(c))
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Paket siswa berhasil dibatalkan", subscriptiondto.ToStudentSubscriptionResponse(sub, today()))
}

// SetOverrides menangani PUT /api/v1/student-access-overrides: atur akses khusus 1-100 siswa.
func (h *SubscriptionHandler) SetOverrides(c *gin.Context) {
	var req subscriptiondto.OverridesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err)
		return
	}
	students, access, err := h.service.SetOverrides(c.Request.Context(), req, middleware.GetUserID(c))
	if err != nil {
		h.handleError(c, err)
		return
	}
	items := make([]subscriptiondto.StudentAccessItemResponse, 0, len(students))
	for _, st := range students {
		items = append(items, subscriptiondto.StudentAccessItemResponse{
			Student: subscriptiondto.AccessStudentResponse{ID: st.ID, Name: st.Name, StudentCode: st.StudentCode, CoinBalance: st.CoinBalance},
			Access:  subscriptiondto.ToAccessResponse(access[st.ID]),
		})
	}
	response.Success(c, http.StatusOK, "Akses khusus siswa berhasil diubah", items)
}

// handleError mengubah error dari service menjadi response HTTP yang sesuai.
func (h *SubscriptionHandler) handleError(c *gin.Context, err error) {
	_ = c.Error(err) // dicatat di log API (field "process")

	var validationErr *subscriptionmodel.ValidationError
	switch {
	case errors.As(err, &validationErr):
		fields := make([]response.FieldError, 0, len(validationErr.Fields))
		for _, f := range validationErr.Fields {
			fields = append(fields, response.FieldError{Field: f.Field, Message: f.Message})
		}
		response.ErrorWithFields(c, http.StatusUnprocessableEntity, "Validasi gagal", fields)
	case errors.Is(err, subscriptionmodel.ErrPlanNotFound):
		response.Error(c, http.StatusNotFound, "Paket akses tidak ditemukan")
	case errors.Is(err, subscriptionmodel.ErrStudentNotFound):
		response.Error(c, http.StatusNotFound, "Siswa tidak ditemukan")
	case errors.Is(err, subscriptionmodel.ErrSubscriptionNotFound):
		response.Error(c, http.StatusNotFound, "Paket siswa tidak ditemukan")
	case errors.Is(err, subscriptionmodel.ErrSubscriptionCancelled):
		response.Error(c, http.StatusConflict, "Paket siswa sudah dibatalkan sebelumnya")
	case errors.Is(err, subscriptionmodel.ErrAssignDuplicate):
		response.Error(c, http.StatusConflict, "Pemberian paket ini sudah diproses sebelumnya (request_id sama), paket tidak ditambah lagi")
	case errors.Is(err, subscriptionmodel.ErrSubscriptionOverlap):
		response.Error(c, http.StatusConflict, "Tanggal paket bertumpuk dengan paket lain siswa, coba lagi")
	default:
		log.Printf("error tidak terduga di %s %s: %v", c.Request.Method, c.Request.URL.Path, err)
		response.Error(c, http.StatusInternalServerError, "Terjadi kesalahan pada server")
	}
}
