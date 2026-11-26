package rumath

import (
	"testing"
	"time"

	rumathmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/quiz/rumath"
)

func TestGenerateQuestionsAlwaysCreatesTenValidQuestions(t *testing.T) {
	tests := []struct {
		name      string
		level     string
		operation string
		min       int
		max       int
	}{
		{name: "easy addition", level: rumathmodel.LevelEasy, operation: rumathmodel.OperationAdd, min: 1, max: 10},
		{name: "easy subtraction", level: rumathmodel.LevelEasy, operation: rumathmodel.OperationSubtract, min: 1, max: 10},
		{name: "medium addition", level: rumathmodel.LevelMedium, operation: rumathmodel.OperationAdd, min: 10, max: 100},
		{name: "medium subtraction", level: rumathmodel.LevelMedium, operation: rumathmodel.OperationSubtract, min: 10, max: 100},
		{name: "hard addition", level: rumathmodel.LevelHard, operation: rumathmodel.OperationAdd, min: 100, max: 1000},
		{name: "hard subtraction", level: rumathmodel.LevelHard, operation: rumathmodel.OperationSubtract, min: 100, max: 1000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := rumathmodel.Session{
				ID:        "session-id",
				Level:     tt.level,
				Operation: tt.operation,
			}
			questions := generateQuestions(session, time.Now())
			if len(questions) != rumathmodel.QuestionsPerSession {
				t.Fatalf("got %d questions, want %d", len(questions), rumathmodel.QuestionsPerSession)
			}

			seen := make(map[[2]int]bool, len(questions))
			for i, question := range questions {
				if question.QuestionNumber != i+1 {
					t.Errorf("question number = %d, want %d", question.QuestionNumber, i+1)
				}
				if question.OperandA < tt.min || question.OperandA > tt.max ||
					question.OperandB < tt.min || question.OperandB > tt.max {
					t.Errorf("operands outside [%d, %d]: %d %s %d", tt.min, tt.max, question.OperandA, tt.operation, question.OperandB)
				}
				if tt.operation == rumathmodel.OperationSubtract {
					if question.OperandA < question.OperandB || question.Answer != question.OperandA-question.OperandB {
						t.Errorf("invalid subtraction question: %+v", question)
					}
				} else if question.Answer != question.OperandA+question.OperandB {
					t.Errorf("invalid addition question: %+v", question)
				}
				pair := [2]int{question.OperandA, question.OperandB}
				if seen[pair] {
					t.Errorf("duplicate question pair: %v", pair)
				}
				seen[pair] = true
			}
		})
	}
}
