// Package task implements business rules for shared Admin tasks.
package task

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"

	taskmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/admin/task"
	taskrepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/admin/task"
)

type TaskService struct {
	repo *taskrepo.TaskRepository
}

func NewTaskService(repo *taskrepo.TaskRepository) *TaskService {
	return &TaskService{repo: repo}
}

func (s *TaskService) Create(ctx context.Context, text, createdBy string) (taskmodel.Task, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return taskmodel.Task{}, &taskmodel.ValidationError{Field: "text", Message: "wajib diisi"}
	}
	if utf8.RuneCountInString(text) > 5000 {
		return taskmodel.Task{}, &taskmodel.ValidationError{Field: "text", Message: "maksimal 5000 karakter"}
	}
	return s.repo.Create(ctx, text, createdBy)
}

func (s *TaskService) List(ctx context.Context, page, perPage int) ([]taskmodel.Task, int64, error) {
	return s.repo.List(ctx, page, perPage)
}

func (s *TaskService) GetByID(ctx context.Context, id string) (taskmodel.Task, error) {
	if _, err := uuid.Parse(id); err != nil {
		return taskmodel.Task{}, taskmodel.ErrNotFound
	}
	return s.repo.GetByID(ctx, id)
}

func (s *TaskService) UpdateStatus(ctx context.Context, id, status, updatedBy string) (taskmodel.Task, error) {
	if _, err := uuid.Parse(id); err != nil {
		return taskmodel.Task{}, taskmodel.ErrNotFound
	}
	if status != taskmodel.StatusPending && status != taskmodel.StatusCompleted {
		return taskmodel.Task{}, &taskmodel.ValidationError{Field: "status", Message: "pilih pending atau completed"}
	}
	if err := s.repo.UpdateStatus(ctx, id, status, updatedBy); err != nil {
		return taskmodel.Task{}, err
	}
	return s.repo.GetByID(ctx, id)
}
