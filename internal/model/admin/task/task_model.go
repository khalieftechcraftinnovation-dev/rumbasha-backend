// Package task defines the Admin task model and domain errors.
package task

import (
	"errors"
	"time"
)

const (
	StatusPending   = "pending"
	StatusCompleted = "completed"
)

var (
	ErrNotFound      = errors.New("task not found")
	ErrInvalidStatus = errors.New("invalid task status")
	ErrInvalidText   = errors.New("invalid task text")
)

type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}

type Task struct {
	ID                  string     `gorm:"column:id;primaryKey"`
	Text                string     `gorm:"column:task_text"`
	Status              string     `gorm:"column:status"`
	CreatedByID         string     `gorm:"column:created_by"`
	CreatedByUsername   string     `gorm:"column:created_by_username;->"`
	CreatedByName       string     `gorm:"column:created_by_name;->"`
	StatusUpdatedByID   *string    `gorm:"column:status_updated_by"`
	StatusUpdatedByUser *string    `gorm:"column:status_updated_by_username;->"`
	StatusUpdatedByName *string    `gorm:"column:status_updated_by_name;->"`
	CreatedAt           time.Time  `gorm:"column:created_at"`
	StatusUpdatedAt     *time.Time `gorm:"column:status_updated_at"`
}

func (Task) TableName() string {
	return "admin_tasks"
}
