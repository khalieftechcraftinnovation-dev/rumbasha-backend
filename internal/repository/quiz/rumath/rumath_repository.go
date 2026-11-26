// Package rumath persists RUMATH sessions and their generated questions.
package rumath

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	rumathmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/quiz/rumath"
)

type RumathRepository struct {
	db *gorm.DB
}

func NewRumathRepository(db *gorm.DB) *RumathRepository {
	return &RumathRepository{db: db}
}

func (r *RumathRepository) CreateSession(ctx context.Context, session rumathmodel.Session, questions []rumathmodel.Question) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&session).Error; err != nil {
			return fmt.Errorf("create RUMATH session: %w", err)
		}
		if err := tx.Create(&questions).Error; err != nil {
			return fmt.Errorf("create RUMATH questions: %w", err)
		}
		return nil
	})
}
