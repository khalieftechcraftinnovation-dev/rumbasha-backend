// Package rumath defines RUMATH request and response types.
package rumath

import (
	"time"

	rumathmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/quiz/rumath"
)

type GenerateSessionRequest struct {
	ChallengeType string `json:"challenge_type" binding:"required,oneof=hitung_cepat hitung_bersusun"`
	Operation     string `json:"operation" binding:"required,oneof=+ -"`
	Level         string `json:"level" binding:"required,oneof=mudah sedang sulit"`
}

type GenerateSessionResponse struct {
	ID             string             `json:"id"`
	ChallengeType  string             `json:"challenge_type"`
	Operation      string             `json:"operation"`
	Level          string             `json:"level"`
	TotalQuestions int                `json:"total_questions"`
	CreatedAt      time.Time          `json:"created_at"`
	Questions      []QuestionResponse `json:"questions"`
}

type QuestionResponse struct {
	ID             string `json:"id"`
	QuestionNumber int    `json:"question_number"`
	OperandA       int    `json:"operand_a"`
	Operation      string `json:"operation"`
	OperandB       int    `json:"operand_b"`
	Answer         int    `json:"answer"`
}

func ToGenerateSessionResponse(generated rumathmodel.GeneratedSession) GenerateSessionResponse {
	session := generated.Session
	response := GenerateSessionResponse{
		ID:             session.ID,
		ChallengeType:  session.ChallengeType,
		Operation:      session.Operation,
		Level:          session.Level,
		TotalQuestions: session.TotalQuestions,
		CreatedAt:      session.CreatedAt,
		Questions:      make([]QuestionResponse, 0, len(generated.Questions)),
	}
	for _, question := range generated.Questions {
		response.Questions = append(response.Questions, QuestionResponse{
			ID:             question.ID,
			QuestionNumber: question.QuestionNumber,
			OperandA:       question.OperandA,
			Operation:      session.Operation,
			OperandB:       question.OperandB,
			Answer:         question.Answer,
		})
	}
	return response
}
