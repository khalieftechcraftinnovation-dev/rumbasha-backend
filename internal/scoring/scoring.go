// Package scoring berisi aturan nilai & point quiz yang dipakai bersama oleh Daily Quiz, Uji Kom,
// dan Tryout: menyusun jawaban yang dikumpulkan siswa, menghitung nilai & point, dan menyusun review.
//
// Nilai != point:
//   - Nilai (score): total 100, per soal = 100 / total soal, nilai = jumlah benar x nilai per soal.
//   - Point (points): untuk leaderboard, per soal BENAR sesuai level soal: easy 10, medium 15,
//     hard 20 (default; diatur admin: per quiz untuk Uji Kom & Tryout, global untuk Daily Quiz);
//     salah / tidak dijawab = 0.
//
// Benar/salah per soal dinilai di FE (FE sudah menerima kunci jawaban saat quiz dibuka).
package scoring

import (
	"encoding/json"
	"fmt"
	"math"
	"time"

	questionmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/quiz/question"
)

// MaxScore adalah nilai total semua quiz.
const MaxScore = 100.0

// PointsByDifficulty adalah point 1 soal yang dijawab benar, menurut level (difficulty) soal.
var PointsByDifficulty = map[string]int{
	questionmodel.DifficultyEasy:   10,
	questionmodel.DifficultyMedium: 15,
	questionmodel.DifficultyHard:   20,
}

// SubmittedAnswer adalah 1 jawaban yang dikirim FE.
type SubmittedAnswer struct {
	QuestionID       string
	Answer           json.RawMessage // jawaban siswa apa adanya (contoh "B", ["A","C"], "0.75"); null = tidak dijawab
	IsCorrect        bool            // hasil penilaian FE
	AnsweredAt       *time.Time      // jam soal terakhir dijawab (dari FE); nil = tidak dikirim
	TimeSpentSeconds *int            // lama mengerjakan soal ini, detik (dari FE); nil = tidak dikirim
}

// Answer adalah 1 jawaban yang disimpan (urut nomor soal). Tag json = isi kolom JSONB answers.
type Answer struct {
	QuestionID       string          `json:"question_id"`
	Answer           json.RawMessage `json:"answer"`
	IsCorrect        bool            `json:"is_correct"`
	Score            float64         `json:"score"`  // nilai yang didapat dari soal ini (0 kalau salah)
	Points           int             `json:"points"` // point leaderboard dari soal ini (0 kalau salah)
	AnsweredAt       *time.Time      `json:"answered_at"`
	TimeSpentSeconds *int            `json:"time_spent_seconds"`
}

// Result adalah hasil penilaian 1 pengerjaan quiz.
type Result struct {
	Answers          []Answer
	CorrectCount     int
	Score            float64 // nilai 0-100, dibulatkan 2 angka di belakang koma
	ScorePerQuestion float64
	Points           int // total point leaderboard
	DurationSeconds  int // lama mengerjakan total (submit - mulai); diisi service
}

// DurationSeconds = lama mengerjakan dalam detik (submit - mulai), tidak pernah negatif.
func DurationSeconds(startedAt, submittedAt time.Time) int {
	d := int(submittedAt.Sub(startedAt).Seconds())
	if d < 0 {
		return 0
	}
	return d
}

// FieldError adalah 1 kesalahan isi jawaban, contoh field "answers[2].question_id".
type FieldError struct {
	Field, Message string
}

// Grade menyusun jawaban sesuai urutan soal quiz (questionIDs) lalu menghitung nilai & point.
// difficulties = level setiap soal (id → easy/medium/hard); pointTable = point 1 soal benar per level
// (Daily Quiz: pengaturan global daily_quiz_settings; Uji Kom/Tryout: diatur admin per quiz).
// Soal yang tidak dikirim dianggap tidak dijawab (salah). Jawaban untuk soal yang bukan bagian
// quiz atau dikirim 2x dikembalikan sebagai kesalahan.
func Grade(questionIDs []string, difficulties map[string]string, pointTable map[string]int, submitted []SubmittedAnswer) (Result, []FieldError) {
	inQuiz := make(map[string]bool, len(questionIDs))
	for _, id := range questionIDs {
		inQuiz[id] = true
	}

	errs := []FieldError{}
	byID := make(map[string]SubmittedAnswer, len(submitted))
	for i, a := range submitted {
		field := fmt.Sprintf("answers[%d].question_id", i)
		switch {
		case !inQuiz[a.QuestionID]:
			errs = append(errs, FieldError{field, "soal ini bukan bagian dari quiz"})
		case hasKey(byID, a.QuestionID):
			errs = append(errs, FieldError{field, "soal ini dikirim lebih dari sekali"})
		default:
			byID[a.QuestionID] = a
		}
	}
	if len(errs) > 0 {
		return Result{}, errs
	}

	result := Result{ScorePerQuestion: ScorePerQuestion(len(questionIDs))}
	result.Answers = make([]Answer, 0, len(questionIDs))
	for _, id := range questionIDs {
		a := Answer{QuestionID: id, Answer: json.RawMessage("null")}
		if s, ok := byID[id]; ok {
			if len(s.Answer) > 0 {
				a.Answer = s.Answer
			}
			a.IsCorrect = s.IsCorrect
			a.AnsweredAt = s.AnsweredAt
			a.TimeSpentSeconds = s.TimeSpentSeconds
		}
		if a.IsCorrect {
			a.Score = round2(result.ScorePerQuestion)
			a.Points = pointTable[difficulties[id]]
			result.CorrectCount++
			result.Points += a.Points
		}
		result.Answers = append(result.Answers, a)
	}
	// Nilai dihitung dari jumlah benar (bukan menjumlah nilai per soal yang sudah dibulatkan),
	// supaya semua benar selalu tepat 100.
	result.Score = round2(float64(result.CorrectCount) * MaxScore / float64(len(questionIDs)))
	return result, nil
}

// ScorePerQuestion = 100 / total soal (belum dibulatkan).
func ScorePerQuestion(total int) float64 {
	if total == 0 {
		return 0
	}
	return MaxScore / float64(total)
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}

func hasKey(m map[string]SubmittedAnswer, key string) bool {
	_, ok := m[key]
	return ok
}

// ReviewItem adalah 1 soal di review setelah submit: isi soal + kunci jawaban + pembahasan +
// jawaban siswa & hasilnya.
type ReviewItem struct {
	questionmodel.AttemptQuestion
	StudentAnswer    json.RawMessage
	AnsweredAt       *time.Time
	TimeSpentSeconds *int
	IsCorrect        bool
	Score            float64
	Points           int
}

// Review menggabungkan isi soal (urut nomor soal, sudah berisi pembahasan) dengan jawaban yang
// disimpan. Urutan review = urutan soal saat dikerjakan.
func Review(content []questionmodel.AttemptQuestion, answers []Answer) []ReviewItem {
	byID := make(map[string]Answer, len(answers))
	for _, a := range answers {
		byID[a.QuestionID] = a
	}
	items := make([]ReviewItem, 0, len(content))
	for _, q := range content {
		a, ok := byID[q.ID]
		if !ok {
			a = Answer{Answer: json.RawMessage("null")}
		}
		items = append(items, ReviewItem{
			AttemptQuestion: q, StudentAnswer: a.Answer, IsCorrect: a.IsCorrect,
			Score: a.Score, Points: a.Points,
			AnsweredAt: a.AnsweredAt, TimeSpentSeconds: a.TimeSpentSeconds,
		})
	}
	return items
}
