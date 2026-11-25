// Package scoring berisi aturan nilai quiz yang dipakai bersama oleh Daily Quiz, Uji Kom, dan
// Tryout: menyusun jawaban yang dikumpulkan siswa, menghitung nilai, dan menyusun review.
//
// Aturan nilai: total 100, nilai per soal = 100 / total soal, nilai = jumlah benar x nilai per soal.
// Benar/salah per soal dinilai di FE (FE sudah menerima kunci jawaban saat quiz dibuka).
package scoring

import (
	"encoding/json"
	"fmt"
	"math"
	"time"

	questionmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/question"
)

// MaxScore adalah nilai total semua quiz.
const MaxScore = 100.0

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
	Points           float64         `json:"points"` // nilai yang didapat dari soal ini (0 kalau salah)
	AnsweredAt       *time.Time      `json:"answered_at"`
	TimeSpentSeconds *int            `json:"time_spent_seconds"`
}

// Result adalah hasil penilaian 1 pengerjaan quiz.
type Result struct {
	Answers           []Answer
	CorrectCount      int
	Score             float64 // 0-100, dibulatkan 2 angka di belakang koma
	PointsPerQuestion float64
	DurationSeconds   int // lama mengerjakan total (submit - mulai); diisi service
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

// Grade menyusun jawaban sesuai urutan soal quiz (questionIDs) lalu menghitung nilai.
// Soal yang tidak dikirim dianggap tidak dijawab (salah). Jawaban untuk soal yang bukan bagian
// quiz atau dikirim 2x dikembalikan sebagai kesalahan.
func Grade(questionIDs []string, submitted []SubmittedAnswer) (Result, []FieldError) {
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

	result := Result{PointsPerQuestion: PointsPerQuestion(len(questionIDs))}
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
			a.Points = round2(result.PointsPerQuestion)
			result.CorrectCount++
		}
		result.Answers = append(result.Answers, a)
	}
	// Nilai dihitung dari jumlah benar (bukan menjumlah poin yang sudah dibulatkan), supaya
	// semua benar selalu tepat 100.
	result.Score = round2(float64(result.CorrectCount) * MaxScore / float64(len(questionIDs)))
	return result, nil
}

// PointsPerQuestion = 100 / total soal (belum dibulatkan).
func PointsPerQuestion(total int) float64 {
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
	Points           float64
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
			AttemptQuestion: q, StudentAnswer: a.Answer, IsCorrect: a.IsCorrect, Points: a.Points,
			AnsweredAt: a.AnsweredAt, TimeSpentSeconds: a.TimeSpentSeconds,
		})
	}
	return items
}
