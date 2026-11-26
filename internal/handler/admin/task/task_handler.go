// Package task handles the Admin task endpoints.
package task

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	taskdto "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/dto/admin/task"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/middleware"
	taskmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/admin/task"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/response"
	taskservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/admin/task"
)

type TaskHandler struct {
	service *taskservice.TaskService
}

func NewTaskHandler(service *taskservice.TaskService) *TaskHandler {
	return &TaskHandler{service: service}
}

func (h *TaskHandler) List(c *gin.Context) {
	var query taskdto.ListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.ValidationError(c, err)
		return
	}
	query.SetDefaults()

	tasks, total, err := h.service.List(c.Request.Context(), query.Page, query.PerPage)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.SuccessWithMeta(c, "Daftar task", taskdto.ToTaskResponses(tasks),
		response.Meta{Page: query.Page, PerPage: query.PerPage, Total: int(total)})
}

func (h *TaskHandler) Get(c *gin.Context) {
	task, err := h.service.GetByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Detail task", taskdto.ToTaskResponse(task))
}

func (h *TaskHandler) Create(c *gin.Context) {
	var request taskdto.CreateRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		response.ValidationError(c, err)
		return
	}
	task, err := h.service.Create(c.Request.Context(), request.Text, middleware.GetUserID(c))
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusCreated, "Task berhasil dibuat", taskdto.ToTaskResponse(task))
}

func (h *TaskHandler) UpdateStatus(c *gin.Context) {
	var request taskdto.UpdateStatusRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		response.ValidationError(c, err)
		return
	}
	task, err := h.service.UpdateStatus(
		c.Request.Context(),
		c.Param("id"),
		request.Status,
		middleware.GetUserID(c),
	)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Status task berhasil diperbarui", taskdto.ToTaskResponse(task))
}

func (h *TaskHandler) handleError(c *gin.Context, err error) {
	_ = c.Error(err)

	var validationErr *taskmodel.ValidationError
	switch {
	case errors.As(err, &validationErr):
		response.ErrorWithFields(c, http.StatusUnprocessableEntity, "Validasi gagal", []response.FieldError{
			{Field: validationErr.Field, Message: validationErr.Message},
		})
	case errors.Is(err, taskmodel.ErrNotFound):
		response.Error(c, http.StatusNotFound, "Task tidak ditemukan")
	default:
		log.Printf("error tidak terduga di %s %s: %v", c.Request.Method, c.Request.URL.Path, err)
		response.Error(c, http.StatusInternalServerError, "Terjadi kesalahan pada server")
	}
}
