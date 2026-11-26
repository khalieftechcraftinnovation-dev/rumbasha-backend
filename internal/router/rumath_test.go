package router

import (
	"net/http"
	"testing"

	rumathdto "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/dto/quiz/rumath"
	loginmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/auth/login"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/quiz/rumath"
)

func TestRumathGenerateSessionPersistsTenQuestions(t *testing.T) {
	r, db := setupRouter(t)
	studentID := createUser(t, db, loginmodel.RoleStudent, "student.rumath", loginmodel.UserStatusActive)
	if err := db.Exec(`
		INSERT INTO student_feature_overrides (student_id, feature, is_enabled)
		VALUES (?, 'rumath', TRUE)`, studentID).Error; err != nil {
		t.Fatal(err)
	}
	studentToken := adminTaskToken(t, studentID, loginmodel.RoleStudent)

	res := sendRequest(r, http.MethodPost, "/api/v1/rumath/sessions",
		`{"challenge_type":"hitung_bersusun","operation":"+","level":"sedang"}`, studentToken)
	if res.Code != http.StatusCreated {
		t.Fatalf("generate RUMATH session: status %d, body: %s", res.Code, res.Body)
	}

	var generated rumathdto.GenerateSessionResponse
	decodeData(t, res, &generated)
	if generated.ChallengeType != rumath.ChallengeColumn ||
		generated.Operation != rumath.OperationAdd ||
		generated.Level != rumath.LevelMedium ||
		generated.TotalQuestions != rumath.QuestionsPerSession ||
		len(generated.Questions) != rumath.QuestionsPerSession {
		t.Fatalf("unexpected generated session: %+v", generated)
	}
	for _, question := range generated.Questions {
		if question.Answer != question.OperandA+question.OperandB {
			t.Errorf("incorrect answer in generated question: %+v", question)
		}
	}

	var persisted int64
	if err := db.Model(&rumath.Session{}).Where("id = ?", generated.ID).Count(&persisted).Error; err != nil {
		t.Fatal(err)
	}
	if persisted != 1 {
		t.Fatalf("session persistence count = %d, want 1", persisted)
	}
	if err := db.Model(&rumath.Question{}).Where("session_id = ?", generated.ID).Count(&persisted).Error; err != nil {
		t.Fatal(err)
	}
	if persisted != rumath.QuestionsPerSession {
		t.Fatalf("persisted question count = %d, want %d", persisted, rumath.QuestionsPerSession)
	}
}

func TestRumathGenerateSessionRequiresFeatureAccess(t *testing.T) {
	r, db := setupRouter(t)
	studentID := createUser(t, db, loginmodel.RoleStudent, "student.rumath.locked", loginmodel.UserStatusActive)
	studentToken := adminTaskToken(t, studentID, loginmodel.RoleStudent)

	res := sendRequest(r, http.MethodPost, "/api/v1/rumath/sessions",
		`{"challenge_type":"hitung_cepat","operation":"-","level":"mudah"}`, studentToken)
	if res.Code != http.StatusForbidden {
		t.Fatalf("student without RUMATH access: status %d, body: %s", res.Code, res.Body)
	}
}
