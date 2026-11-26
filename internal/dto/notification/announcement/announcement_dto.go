// Package announcement contains request and response types for announcements.
package announcement

import (
	"time"

	announcementmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/notification/announcement"
)

type CreateRequest struct {
	Title     string     `json:"title" binding:"required,max=200"`
	Body      string     `json:"body" binding:"required,max=20000"`
	GradeIDs  []string   `json:"grade_ids" binding:"required,min=1,max=100,unique,dive,uuid"`
	Status    string     `json:"status" binding:"omitempty,oneof=draft published"`
	PublishAt *time.Time `json:"publish_at"`
}

type UpdateRequest struct {
	Title     string     `json:"title" binding:"required,max=200"`
	Body      string     `json:"body" binding:"required,max=20000"`
	GradeIDs  []string   `json:"grade_ids" binding:"required,min=1,max=100,unique,dive,uuid"`
	Status    string     `json:"status" binding:"required,oneof=draft published"`
	PublishAt *time.Time `json:"publish_at"`
}

type ListQuery struct {
	Page    int    `form:"page" binding:"omitempty,min=1"`
	PerPage int    `form:"per_page" binding:"omitempty,min=1,max=100"`
	Search  string `form:"search" binding:"omitempty,max=100"`
	Status  string `form:"status" binding:"omitempty,oneof=draft published"`
}

func (q *ListQuery) SetDefaults() {
	if q.Page == 0 {
		q.Page = 1
	}
	if q.PerPage == 0 {
		q.PerPage = 20
	}
}

type StudentListQuery struct {
	Page    int `form:"page" binding:"omitempty,min=1"`
	PerPage int `form:"per_page" binding:"omitempty,min=1,max=100"`
}

func (q *StudentListQuery) SetDefaults() {
	if q.Page == 0 {
		q.Page = 1
	}
	if q.PerPage == 0 {
		q.PerPage = 20
	}
}

type ActorResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type AnnouncementResponse struct {
	ID        string         `json:"id"`
	Title     string         `json:"title"`
	Body      string         `json:"body"`
	Status    string         `json:"status"`
	PublishAt *time.Time     `json:"publish_at"`
	GradeIDs  []string       `json:"grade_ids"`
	CreatedBy ActorResponse  `json:"created_by"`
	UpdatedBy *ActorResponse `json:"updated_by"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	IsRead    *bool          `json:"is_read,omitempty"`
	ReadAt    *time.Time     `json:"read_at,omitempty"`
}

type StudentAnnouncementResponse struct {
	ID        string     `json:"id"`
	Title     string     `json:"title"`
	Body      string     `json:"body"`
	PublishAt *time.Time `json:"publish_at"`
	IsRead    bool       `json:"is_read"`
	ReadAt    *time.Time `json:"read_at"`
}

type UnreadCountResponse struct {
	Count int64 `json:"count"`
}

type MarkAllReadResponse struct {
	MarkedCount int64 `json:"marked_count"`
}

func ToResponse(item announcementmodel.Announcement) AnnouncementResponse {
	createdByName := item.CreatedByName
	if createdByName == "" {
		createdByName = item.CreatedByID
	}
	result := AnnouncementResponse{
		ID:        item.ID,
		Title:     item.Title,
		Body:      item.Body,
		Status:    item.Status,
		PublishAt: item.PublishedAt,
		GradeIDs:  item.GradeIDs,
		CreatedBy: ActorResponse{ID: item.CreatedByID, Name: createdByName},
		CreatedAt: item.CreatedAt,
		UpdatedAt: item.UpdatedAt,
	}
	if item.UpdatedByID != nil {
		name := item.UpdatedByID
		if item.UpdatedByName != nil && *item.UpdatedByName != "" {
			name = item.UpdatedByName
		}
		result.UpdatedBy = &ActorResponse{ID: *item.UpdatedByID, Name: *name}
	}
	return result
}

func ToResponses(items []announcementmodel.Announcement) []AnnouncementResponse {
	responses := make([]AnnouncementResponse, 0, len(items))
	for _, item := range items {
		responses = append(responses, ToResponse(item))
	}
	return responses
}

func ToStudentResponses(items []announcementmodel.Announcement) []StudentAnnouncementResponse {
	responses := make([]StudentAnnouncementResponse, 0, len(items))
	for _, item := range items {
		responses = append(responses, StudentAnnouncementResponse{
			ID:        item.ID,
			Title:     item.Title,
			Body:      item.Body,
			PublishAt: item.PublishedAt,
			IsRead:    item.IsRead,
			ReadAt:    item.ReadAt,
		})
	}
	return responses
}
