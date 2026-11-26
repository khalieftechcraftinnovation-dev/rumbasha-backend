// Package announcement defines in-app announcements and notification read state.
package announcement

import (
	"errors"
	"time"
)

const (
	StatusDraft     = "draft"
	StatusPublished = "published"
)

var (
	ErrNotFound       = errors.New("announcement not found")
	ErrInvalidTargets = errors.New("invalid announcement targets")
)

type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}

type Announcement struct {
	ID            string     `gorm:"column:id;primaryKey"`
	Title         string     `gorm:"column:title"`
	Body          string     `gorm:"column:body"`
	Status        string     `gorm:"column:status"`
	PublishedAt   *time.Time `gorm:"column:published_at"`
	CreatedByID   string     `gorm:"column:created_by"`
	CreatedByName string     `gorm:"column:created_by_name;->"`
	UpdatedByID   *string    `gorm:"column:updated_by"`
	UpdatedByName *string    `gorm:"column:updated_by_name;->"`
	CreatedAt     time.Time  `gorm:"column:created_at"`
	UpdatedAt     time.Time  `gorm:"column:updated_at"`
	DeletedAt     *time.Time `gorm:"column:deleted_at"`
	GradeIDs      []string   `gorm:"-"`
	IsRead        bool       `gorm:"column:is_read;->"`
	ReadAt        *time.Time `gorm:"column:read_at;->"`
}

func (Announcement) TableName() string {
	return "announcements"
}
