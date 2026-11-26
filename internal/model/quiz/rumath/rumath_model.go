// Package rumath defines RUMATH practice sessions and generated questions.
package rumath

import (
	"errors"
	"time"
)

const (
	ChallengeQuick      = "hitung_cepat"
	ChallengeColumn     = "hitung_bersusun"
	OperationAdd        = "+"
	OperationSubtract   = "-"
	LevelEasy           = "mudah"
	LevelMedium         = "sedang"
	LevelHard           = "sulit"
	QuestionsPerSession = 10
)

var ErrAccessRequired = errors.New("RUMATH access is required")

type FieldError struct {
	Field   string
	Message string
}

type ValidationError struct {
	Fields []FieldError
}

func (e *ValidationError) Error() string {
	return "input sesi RUMATH tidak valid"
}

type Session struct {
	ID             string    `gorm:"column:id;primaryKey"`
	StudentID      string    `gorm:"column:student_id"`
	ChallengeType  string    `gorm:"column:challenge_type"`
	Operation      string    `gorm:"column:operation"`
	Level          string    `gorm:"column:level"`
	TotalQuestions int       `gorm:"column:total_questions"`
	CreatedAt      time.Time `gorm:"column:created_at"`
}

func (Session) TableName() string {
	return "rumath_sessions"
}

type Question struct {
	ID             string    `gorm:"column:id;primaryKey"`
	SessionID      string    `gorm:"column:session_id"`
	QuestionNumber int       `gorm:"column:question_number"`
	OperandA       int       `gorm:"column:operand_a"`
	OperandB       int       `gorm:"column:operand_b"`
	Answer         int       `gorm:"column:answer"`
	CreatedAt      time.Time `gorm:"column:created_at"`
}

func (Question) TableName() string {
	return "rumath_questions"
}

type GeneratedSession struct {
	Session   Session
	Questions []Question
}

func (s Session) Validate() error {
	var fields []FieldError
	if s.ChallengeType != ChallengeQuick && s.ChallengeType != ChallengeColumn {
		fields = append(fields, FieldError{Field: "challenge_type", Message: "pilih hitung_cepat atau hitung_bersusun"})
	}
	if s.Operation != OperationAdd && s.Operation != OperationSubtract {
		fields = append(fields, FieldError{Field: "operation", Message: "pilih operasi + atau -"})
	}
	switch s.Level {
	case LevelEasy, LevelMedium, LevelHard:
	default:
		fields = append(fields, FieldError{Field: "level", Message: "pilih mudah, sedang, atau sulit"})
	}
	if len(fields) > 0 {
		return &ValidationError{Fields: fields}
	}
	return nil
}
