// Package announcement persists web announcements and per-student read state.
package announcement

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	announcementmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/notification/announcement"
)

type AnnouncementRepository struct {
	db *gorm.DB
}

func NewAnnouncementRepository(db *gorm.DB) *AnnouncementRepository {
	return &AnnouncementRepository{db: db}
}

type announcementGrade struct {
	AnnouncementID string `gorm:"column:announcement_id"`
	GradeID        string `gorm:"column:grade_id"`
}

func (announcementGrade) TableName() string {
	return "announcement_grades"
}

func (r *AnnouncementRepository) ValidateGradeIDs(ctx context.Context, gradeIDs []string) error {
	var count int64
	if err := r.db.WithContext(ctx).
		Table("master_data_items").
		Where("id IN ? AND type = 'grade' AND status = 'active' AND deleted_at IS NULL", gradeIDs).
		Count(&count).Error; err != nil {
		return err
	}
	if count != int64(len(gradeIDs)) {
		return announcementmodel.ErrInvalidTargets
	}
	return nil
}

func (r *AnnouncementRepository) Create(ctx context.Context, item announcementmodel.Announcement) (announcementmodel.Announcement, error) {
	item.ID = uuid.NewString()
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&item).Error; err != nil {
			return err
		}
		if err := replaceGrades(tx, item.ID, item.GradeIDs); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return announcementmodel.Announcement{}, err
	}
	return r.GetAdminByID(ctx, item.ID)
}

func (r *AnnouncementRepository) ListAdmin(ctx context.Context, page, perPage int, search, status string) ([]announcementmodel.Announcement, int64, error) {
	query := r.adminQuery(ctx)
	if status != "" {
		query = query.Where("a.status = ?", status)
	}
	if search != "" {
		pattern := "%" + search + "%"
		query = query.Where("a.title ILIKE ? OR a.body ILIKE ?", pattern, pattern)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	items := make([]announcementmodel.Announcement, 0)
	err := query.Select(adminSelect).
		Order("a.created_at DESC, a.id DESC").
		Offset((page - 1) * perPage).
		Limit(perPage).
		Scan(&items).Error
	if err != nil {
		return nil, 0, err
	}
	if err := r.loadGrades(ctx, items); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *AnnouncementRepository) GetAdminByID(ctx context.Context, id string) (announcementmodel.Announcement, error) {
	var item announcementmodel.Announcement
	err := r.adminQuery(ctx).
		Select(adminSelect).
		Where("a.id = ?", id).
		Take(&item).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return announcementmodel.Announcement{}, announcementmodel.ErrNotFound
	}
	if err != nil {
		return announcementmodel.Announcement{}, err
	}
	items := []announcementmodel.Announcement{item}
	if err := r.loadGrades(ctx, items); err != nil {
		return announcementmodel.Announcement{}, err
	}
	return items[0], nil
}

func (r *AnnouncementRepository) Update(ctx context.Context, item announcementmodel.Announcement) (announcementmodel.Announcement, error) {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&announcementmodel.Announcement{}).
			Where("id = ? AND deleted_at IS NULL", item.ID).
			Updates(map[string]any{
				"title":        item.Title,
				"body":         item.Body,
				"status":       item.Status,
				"published_at": item.PublishedAt,
				"updated_by":   item.UpdatedByID,
				"updated_at":   gorm.Expr("NOW()"),
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return announcementmodel.ErrNotFound
		}
		return replaceGrades(tx, item.ID, item.GradeIDs)
	})
	if err != nil {
		return announcementmodel.Announcement{}, err
	}
	return r.GetAdminByID(ctx, item.ID)
}

func (r *AnnouncementRepository) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).
		Model(&announcementmodel.Announcement{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Updates(map[string]any{"deleted_at": gorm.Expr("NOW()")})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return announcementmodel.ErrNotFound
	}
	return nil
}

func (r *AnnouncementRepository) ListForStudent(ctx context.Context, userID string, page, perPage int) ([]announcementmodel.Announcement, int64, error) {
	base := r.studentFeedQuery(ctx, userID)
	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	items := make([]announcementmodel.Announcement, 0)
	err := base.Select(`
		a.id, a.title, a.body, a.status, a.published_at, a.created_at, a.updated_at,
		EXISTS (
			SELECT 1 FROM announcement_reads ar
			WHERE ar.announcement_id = a.id AND ar.user_id = ?
		) AS is_read,
		(
			SELECT ar.read_at FROM announcement_reads ar
			WHERE ar.announcement_id = a.id AND ar.user_id = ?
		) AS read_at`, userID, userID).
		Order("a.published_at DESC, a.id DESC").
		Offset((page - 1) * perPage).
		Limit(perPage).
		Scan(&items).Error
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *AnnouncementRepository) UnreadCount(ctx context.Context, userID string) (int64, error) {
	var count int64
	err := r.studentFeedQuery(ctx, userID).
		Where(`NOT EXISTS (
			SELECT 1 FROM announcement_reads ar
			WHERE ar.announcement_id = a.id AND ar.user_id = ?
		)`, userID).
		Count(&count).Error
	return count, err
}

func (r *AnnouncementRepository) MarkRead(ctx context.Context, id, userID string) error {
	var visible bool
	if err := r.db.WithContext(ctx).Raw(`
		SELECT EXISTS (
			SELECT 1
			FROM announcements a
			JOIN announcement_grades ag ON ag.announcement_id = a.id
			JOIN users student ON student.id = ? AND student.grade_id = ag.grade_id
			WHERE a.id = ?
			  AND student.role = 'student'
			  AND student.status = 'active'
			  AND student.deleted_at IS NULL
			  AND a.status = 'published'
			  AND a.published_at <= NOW()
			  AND a.deleted_at IS NULL
		)`, userID, id).Scan(&visible).Error; err != nil {
		return err
	}
	if !visible {
		return announcementmodel.ErrNotFound
	}
	return r.db.WithContext(ctx).Exec(`
		INSERT INTO announcement_reads (announcement_id, user_id)
		VALUES (?, ?)
		ON CONFLICT (announcement_id, user_id) DO NOTHING`, id, userID).Error
}

func (r *AnnouncementRepository) MarkAllRead(ctx context.Context, userID string) (int64, error) {
	result := r.db.WithContext(ctx).Exec(`
		INSERT INTO announcement_reads (announcement_id, user_id)
		SELECT a.id, student.id
		FROM users student
		JOIN announcement_grades ag ON ag.grade_id = student.grade_id
		JOIN announcements a ON a.id = ag.announcement_id
		WHERE student.id = ?
		  AND student.role = 'student'
		  AND student.status = 'active'
		  AND student.deleted_at IS NULL
		  AND a.status = 'published'
		  AND a.published_at <= NOW()
		  AND a.deleted_at IS NULL
		ON CONFLICT (announcement_id, user_id) DO NOTHING`, userID)
	if result.Error != nil {
		return 0, result.Error
	}
	return result.RowsAffected, nil
}

const adminSelect = `
	a.id,
	a.title,
	a.body,
	a.status,
	a.published_at,
	a.created_by,
	COALESCE(NULLIF(creator.real_name, ''), creator.username) AS created_by_name,
	a.updated_by,
	COALESCE(NULLIF(editor.real_name, ''), editor.username) AS updated_by_name,
	a.created_at,
	a.updated_at`

func (r *AnnouncementRepository) adminQuery(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx).
		Table("announcements AS a").
		Joins("JOIN users AS creator ON creator.id = a.created_by").
		Joins("LEFT JOIN users AS editor ON editor.id = a.updated_by").
		Where("a.deleted_at IS NULL")
}

func (r *AnnouncementRepository) studentFeedQuery(ctx context.Context, userID string) *gorm.DB {
	return r.db.WithContext(ctx).
		Table("announcements AS a").
		Joins("JOIN announcement_grades ag ON ag.announcement_id = a.id").
		Joins("JOIN users AS student ON student.id = ? AND student.grade_id = ag.grade_id", userID).
		Where(`
			student.role = 'student'
			AND student.status = 'active'
			AND student.deleted_at IS NULL
			AND a.status = 'published'
			AND a.published_at <= NOW()
			AND a.deleted_at IS NULL`)
}

func (r *AnnouncementRepository) loadGrades(ctx context.Context, items []announcementmodel.Announcement) error {
	if len(items) == 0 {
		return nil
	}
	ids := make([]string, len(items))
	for i := range items {
		ids[i] = items[i].ID
		items[i].GradeIDs = []string{}
	}

	var rows []announcementGrade
	if err := r.db.WithContext(ctx).
		Where("announcement_id IN ?", ids).
		Order("grade_id").
		Find(&rows).Error; err != nil {
		return err
	}
	indices := make(map[string]int, len(items))
	for i := range items {
		indices[items[i].ID] = i
	}
	for _, row := range rows {
		if index, ok := indices[row.AnnouncementID]; ok {
			items[index].GradeIDs = append(items[index].GradeIDs, row.GradeID)
		}
	}
	return nil
}

func replaceGrades(tx *gorm.DB, announcementID string, gradeIDs []string) error {
	if err := tx.Where("announcement_id = ?", announcementID).Delete(&announcementGrade{}).Error; err != nil {
		return err
	}
	grades := make([]announcementGrade, 0, len(gradeIDs))
	for _, gradeID := range gradeIDs {
		grades = append(grades, announcementGrade{AnnouncementID: announcementID, GradeID: gradeID})
	}
	return tx.Create(&grades).Error
}
