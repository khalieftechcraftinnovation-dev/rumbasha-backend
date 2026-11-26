// Package task contains request and response types for Admin tasks.
package task

import (
	"time"

	taskmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/admin/task"
)

type CreateRequest struct {
	Text string `json:"text" binding:"required,max=5000"`
}

type UpdateStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=pending completed"`
}

type ListQuery struct {
	Page    int `form:"page" binding:"omitempty,min=1"`
	PerPage int `form:"per_page" binding:"omitempty,min=1,max=100"`
}

func (q *ListQuery) SetDefaults() {
	if q.Page == 0 {
		q.Page = 1
	}
	if q.PerPage == 0 {
		q.PerPage = 20
	}
}

type ActorResponse struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Name     string `json:"name"`
}

type TaskResponse struct {
	ID              string         `json:"id"`
	Text            string         `json:"text"`
	Status          string         `json:"status"`
	CreatedBy       ActorResponse  `json:"created_by"`
	CreatedAt       time.Time      `json:"created_at"`
	StatusUpdatedBy *ActorResponse `json:"status_updated_by"`
	StatusUpdatedAt *time.Time     `json:"status_updated_at"`
}

func ToTaskResponse(task taskmodel.Task) TaskResponse {
	createdByName := task.CreatedByName
	if createdByName == "" {
		createdByName = task.CreatedByUsername
	}

	response := TaskResponse{
		ID:     task.ID,
		Text:   task.Text,
		Status: task.Status,
		CreatedBy: ActorResponse{
			ID:       task.CreatedByID,
			Username: task.CreatedByUsername,
			Name:     createdByName,
		},
		CreatedAt:       task.CreatedAt,
		StatusUpdatedAt: task.StatusUpdatedAt,
	}

	if task.StatusUpdatedByID != nil {
		name := ""
		if task.StatusUpdatedByName != nil {
			name = *task.StatusUpdatedByName
		}
		username := ""
		if task.StatusUpdatedByUser != nil {
			username = *task.StatusUpdatedByUser
		}
		if name == "" {
			name = username
		}
		response.StatusUpdatedBy = &ActorResponse{
			ID:       *task.StatusUpdatedByID,
			Username: username,
			Name:     name,
		}
	}

	return response
}

func ToTaskResponses(tasks []taskmodel.Task) []TaskResponse {
	responses := make([]TaskResponse, 0, len(tasks))
	for _, task := range tasks {
		responses = append(responses, ToTaskResponse(task))
	}
	return responses
}
