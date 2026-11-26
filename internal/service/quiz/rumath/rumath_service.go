// Package rumath generates and stores student arithmetic practice sessions.
package rumath

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/google/uuid"

	rumathdto "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/dto/quiz/rumath"
	subscriptionmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/billing/subscription"
	rumathmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/quiz/rumath"
	rumathrepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/quiz/rumath"
	subscriptionservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/billing/subscription"
)

type RumathService struct {
	repo          *rumathrepo.RumathRepository
	subscriptions *subscriptionservice.SubscriptionService
}

func NewRumathService(repo *rumathrepo.RumathRepository, subscriptions *subscriptionservice.SubscriptionService) *RumathService {
	return &RumathService{repo: repo, subscriptions: subscriptions}
}

func (s *RumathService) GenerateSession(ctx context.Context, studentID string, req rumathdto.GenerateSessionRequest) (rumathmodel.GeneratedSession, error) {
	session := rumathmodel.Session{
		StudentID:      studentID,
		ChallengeType:  req.ChallengeType,
		Operation:      req.Operation,
		Level:          req.Level,
		TotalQuestions: rumathmodel.QuestionsPerSession,
	}
	if err := session.Validate(); err != nil {
		return rumathmodel.GeneratedSession{}, err
	}

	access, err := s.subscriptions.GetTodayAccess(ctx, studentID)
	if err != nil {
		return rumathmodel.GeneratedSession{}, fmt.Errorf("check RUMATH access: %w", err)
	}
	if !access.Features[subscriptionmodel.FeatureRumath] {
		return rumathmodel.GeneratedSession{}, rumathmodel.ErrAccessRequired
	}

	now := time.Now()
	session.ID = uuid.NewString()
	session.CreatedAt = now
	questions := generateQuestions(session, now)
	if err := s.repo.CreateSession(ctx, session, questions); err != nil {
		return rumathmodel.GeneratedSession{}, err
	}

	return rumathmodel.GeneratedSession{Session: session, Questions: questions}, nil
}

func generateQuestions(session rumathmodel.Session, now time.Time) []rumathmodel.Question {
	min, max := levelBounds(session.Level)
	questions := make([]rumathmodel.Question, 0, rumathmodel.QuestionsPerSession)
	seen := make(map[[2]int]struct{}, rumathmodel.QuestionsPerSession)
	for len(questions) < rumathmodel.QuestionsPerSession {
		a := min + rand.Intn(max-min+1)
		b := min + rand.Intn(max-min+1)
		if session.Operation == rumathmodel.OperationSubtract && a < b {
			a, b = b, a
		}
		if session.Operation == rumathmodel.OperationAdd && a > b {
			a, b = b, a
		}
		pair := [2]int{a, b}
		if _, exists := seen[pair]; exists {
			continue
		}
		seen[pair] = struct{}{}

		answer := a + b
		if session.Operation == rumathmodel.OperationSubtract {
			answer = a - b
		}
		questions = append(questions, rumathmodel.Question{
			ID:             uuid.NewString(),
			SessionID:      session.ID,
			QuestionNumber: len(questions) + 1,
			OperandA:       a,
			OperandB:       b,
			Answer:         answer,
			CreatedAt:      now,
		})
	}
	return questions
}

func levelBounds(level string) (int, int) {
	switch level {
	case rumathmodel.LevelEasy:
		return 1, 10
	case rumathmodel.LevelMedium:
		return 10, 100
	default:
		return 100, 1000
	}
}
