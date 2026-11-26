// Package announcement implements announcement validation and notification rules.
package announcement

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	announcementmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/notification/announcement"
	announcementrepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/notification/announcement"
)

type AnnouncementService struct {
	repo *announcementrepo.AnnouncementRepository
}

func NewAnnouncementService(repo *announcementrepo.AnnouncementRepository) *AnnouncementService {
	return &AnnouncementService{repo: repo}
}

func (s *AnnouncementService) Create(
	ctx context.Context,
	title, body string,
	gradeIDs []string,
	status string,
	publishAt *time.Time,
	createdBy string,
) (announcementmodel.Announcement, error) {
	item, err := s.validate(title, body, gradeIDs, status, publishAt, true)
	if err != nil {
		return announcementmodel.Announcement{}, err
	}
	item.CreatedByID = createdBy
	if err := s.repo.ValidateGradeIDs(ctx, item.GradeIDs); err != nil {
		if err == announcementmodel.ErrInvalidTargets {
			return announcementmodel.Announcement{}, &announcementmodel.ValidationError{
				Field: "grade_ids", Message: "pilih kelas siswa yang aktif",
			}
		}
		return announcementmodel.Announcement{}, err
	}
	return s.repo.Create(ctx, item)
}

func (s *AnnouncementService) Update(
	ctx context.Context,
	id, title, body string,
	gradeIDs []string,
	status string,
	publishAt *time.Time,
	updatedBy string,
) (announcementmodel.Announcement, error) {
	if _, err := uuid.Parse(id); err != nil {
		return announcementmodel.Announcement{}, announcementmodel.ErrNotFound
	}
	item, err := s.validate(title, body, gradeIDs, status, publishAt, false)
	if err != nil {
		return announcementmodel.Announcement{}, err
	}
	item.ID = id
	updatedByCopy := updatedBy
	item.UpdatedByID = &updatedByCopy
	if err := s.repo.ValidateGradeIDs(ctx, item.GradeIDs); err != nil {
		if err == announcementmodel.ErrInvalidTargets {
			return announcementmodel.Announcement{}, &announcementmodel.ValidationError{
				Field: "grade_ids", Message: "pilih kelas siswa yang aktif",
			}
		}
		return announcementmodel.Announcement{}, err
	}
	return s.repo.Update(ctx, item)
}

func (s *AnnouncementService) ListAdmin(
	ctx context.Context,
	page, perPage int,
	search, status string,
) ([]announcementmodel.Announcement, int64, error) {
	return s.repo.ListAdmin(ctx, page, perPage, strings.TrimSpace(search), status)
}

func (s *AnnouncementService) GetAdminByID(ctx context.Context, id string) (announcementmodel.Announcement, error) {
	if _, err := uuid.Parse(id); err != nil {
		return announcementmodel.Announcement{}, announcementmodel.ErrNotFound
	}
	return s.repo.GetAdminByID(ctx, id)
}

func (s *AnnouncementService) Delete(ctx context.Context, id string) error {
	if _, err := uuid.Parse(id); err != nil {
		return announcementmodel.ErrNotFound
	}
	return s.repo.Delete(ctx, id)
}

func (s *AnnouncementService) ListForStudent(
	ctx context.Context,
	userID string,
	page, perPage int,
) ([]announcementmodel.Announcement, int64, error) {
	return s.repo.ListForStudent(ctx, userID, page, perPage)
}

func (s *AnnouncementService) UnreadCount(ctx context.Context, userID string) (int64, error) {
	return s.repo.UnreadCount(ctx, userID)
}

func (s *AnnouncementService) MarkRead(ctx context.Context, id, userID string) error {
	if _, err := uuid.Parse(id); err != nil {
		return announcementmodel.ErrNotFound
	}
	return s.repo.MarkRead(ctx, id, userID)
}

func (s *AnnouncementService) MarkAllRead(ctx context.Context, userID string) (int64, error) {
	return s.repo.MarkAllRead(ctx, userID)
}

func (s *AnnouncementService) validate(
	title, body string,
	gradeIDs []string,
	status string,
	publishAt *time.Time,
	isCreate bool,
) (announcementmodel.Announcement, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return announcementmodel.Announcement{}, &announcementmodel.ValidationError{
			Field: "title", Message: "wajib diisi",
		}
	}
	if utf8.RuneCountInString(title) > 200 {
		return announcementmodel.Announcement{}, &announcementmodel.ValidationError{
			Field: "title", Message: "maksimal 200 karakter",
		}
	}
	body = strings.TrimSpace(body)
	if body == "" {
		return announcementmodel.Announcement{}, &announcementmodel.ValidationError{
			Field: "body", Message: "wajib diisi",
		}
	}
	if utf8.RuneCountInString(body) > 20000 {
		return announcementmodel.Announcement{}, &announcementmodel.ValidationError{
			Field: "body", Message: "maksimal 20000 karakter",
		}
	}
	if status == "" && isCreate {
		status = announcementmodel.StatusDraft
	}
	if status != announcementmodel.StatusDraft && status != announcementmodel.StatusPublished {
		return announcementmodel.Announcement{}, &announcementmodel.ValidationError{
			Field: "status", Message: "pilih draft atau published",
		}
	}
	if status == announcementmodel.StatusDraft && publishAt != nil {
		return announcementmodel.Announcement{}, &announcementmodel.ValidationError{
			Field: "publish_at", Message: "hanya diisi untuk status published",
		}
	}
	if status == announcementmodel.StatusPublished {
		if publishAt == nil {
			now := time.Now().UTC()
			publishAt = &now
		} else if publishAt.IsZero() {
			return announcementmodel.Announcement{}, &announcementmodel.ValidationError{
				Field: "publish_at", Message: "waktu publikasi tidak valid",
			}
		} else {
			publishAt = timePointer(publishAt.UTC())
		}
	}

	seen := make(map[string]struct{}, len(gradeIDs))
	normalizedGradeIDs := make([]string, 0, len(gradeIDs))
	for _, gradeID := range gradeIDs {
		parsed, err := uuid.Parse(gradeID)
		if err != nil {
			return announcementmodel.Announcement{}, &announcementmodel.ValidationError{
				Field: "grade_ids", Message: "ID kelas tidak valid",
			}
		}
		normalizedID := parsed.String()
		if _, exists := seen[normalizedID]; exists {
			return announcementmodel.Announcement{}, &announcementmodel.ValidationError{
				Field: "grade_ids", Message: "kelas tidak boleh duplikat",
			}
		}
		seen[normalizedID] = struct{}{}
		normalizedGradeIDs = append(normalizedGradeIDs, normalizedID)
	}
	if len(normalizedGradeIDs) == 0 {
		return announcementmodel.Announcement{}, &announcementmodel.ValidationError{
			Field: "grade_ids", Message: "pilih minimal satu kelas",
		}
	}

	return announcementmodel.Announcement{
		Title:       title,
		Body:        body,
		Status:      status,
		PublishedAt: publishAt,
		GradeIDs:    normalizedGradeIDs,
	}, nil
}

func timePointer(value time.Time) *time.Time {
	return &value
}
