// Package task persists shared Admin tasks.
package task

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	taskmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/admin/task"
)

type TaskRepository struct {
	db *gorm.DB
}

func NewTaskRepository(db *gorm.DB) *TaskRepository {
	return &TaskRepository{db: db}
}

const taskSelect = `
	admin_tasks.id,
	admin_tasks.task_text,
	admin_tasks.status,
	admin_tasks.created_by,
	creator.username AS created_by_username,
	COALESCE(NULLIF(creator.real_name, ''), creator.username) AS created_by_name,
	admin_tasks.status_updated_by,
	editor.username AS status_updated_by_username,
	COALESCE(NULLIF(editor.real_name, ''), editor.username) AS status_updated_by_name,
	admin_tasks.created_at,
	admin_tasks.status_updated_at`

func (r *TaskRepository) taskQuery(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx).
		Table("admin_tasks").
		Joins("JOIN users AS creator ON creator.id = admin_tasks.created_by").
		Joins("LEFT JOIN users AS editor ON editor.id = admin_tasks.status_updated_by")
}

func (r *TaskRepository) Create(ctx context.Context, text, createdBy string) (taskmodel.Task, error) {
	task := taskmodel.Task{
		ID:          uuid.NewString(),
		Text:        text,
		Status:      taskmodel.StatusPending,
		CreatedByID: createdBy,
	}
	if err := r.db.WithContext(ctx).Create(&task).Error; err != nil {
		return taskmodel.Task{}, err
	}
	return r.GetByID(ctx, task.ID)
}

func (r *TaskRepository) List(ctx context.Context, page, perPage int) ([]taskmodel.Task, int64, error) {
	query := r.taskQuery(ctx)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	tasks := make([]taskmodel.Task, 0)
	err := query.Select(taskSelect).
		Order("admin_tasks.created_at DESC, admin_tasks.id DESC").
		Offset((page - 1) * perPage).
		Limit(perPage).
		Scan(&tasks).Error
	if err != nil {
		return nil, 0, err
	}
	return tasks, total, nil
}

func (r *TaskRepository) GetByID(ctx context.Context, id string) (taskmodel.Task, error) {
	var task taskmodel.Task
	err := r.taskQuery(ctx).
		Select(taskSelect).
		Where("admin_tasks.id = ?", id).
		Take(&task).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return taskmodel.Task{}, taskmodel.ErrNotFound
	}
	if err != nil {
		return taskmodel.Task{}, err
	}
	return task, nil
}

func (r *TaskRepository) UpdateStatus(ctx context.Context, id, status, updatedBy string) error {
	result := r.db.WithContext(ctx).
		Model(&taskmodel.Task{}).
		Where("id = ? AND status <> ?", id, status).
		Updates(map[string]any{
			"status":            status,
			"status_updated_by": updatedBy,
			"status_updated_at": gorm.Expr("NOW()"),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected > 0 {
		return nil
	}

	var count int64
	if err := r.db.WithContext(ctx).
		Model(&taskmodel.Task{}).
		Where("id = ?", id).
		Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return taskmodel.ErrNotFound
	}
	return nil
}
