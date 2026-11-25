// Unit test aturan nilai quiz. Tidak butuh database.
// Jalankan dengan: go test ./internal/scoring/
package scoring

import (
	"encoding/json"
	"testing"
	"time"
)

func TestGrade_NilaiTotal100(t *testing.T) {
	ids := []string{"q1", "q2", "q3"}
	cases := []struct {
		name      string
		submitted []SubmittedAnswer
		correct   int
		score     float64
	}{
		{"semua benar = tepat 100", []SubmittedAnswer{ans("q1", nil, true), ans("q2", nil, true), ans("q3", nil, true)}, 3, 100},
		{"2 dari 3 = 66.67", []SubmittedAnswer{ans("q3", nil, true), ans("q1", nil, true), ans("q2", nil, false)}, 2, 66.67},
		{"tidak dikirim = salah", []SubmittedAnswer{ans("q2", json.RawMessage(`"B"`), true)}, 1, 33.33},
		{"tidak ada jawaban", nil, 0, 0},
	}
	for _, c := range cases {
		r, errs := Grade(ids, nil, PointsByDifficulty, c.submitted)
		if len(errs) > 0 || r.CorrectCount != c.correct || r.Score != c.score {
			t.Errorf("%s: benar %d nilai %v (errs %v), seharusnya %d / %v", c.name, r.CorrectCount, r.Score, errs, c.correct, c.score)
		}
		// Urutan jawaban = urutan soal; soal tanpa jawaban berisi null.
		for i, a := range r.Answers {
			if a.QuestionID != ids[i] || len(a.Answer) == 0 {
				t.Errorf("%s: jawaban ke-%d = %+v", c.name, i, a)
			}
			if a.IsCorrect && a.Score != 33.33 || !a.IsCorrect && a.Score != 0 {
				t.Errorf("%s: nilai soal %s = %v", c.name, a.QuestionID, a.Score)
			}
		}
	}
}

func TestGrade_SoalAsingDanKembar(t *testing.T) {
	_, errs := Grade([]string{"q1"}, nil, PointsByDifficulty, []SubmittedAnswer{ans("q1", nil, true), ans("q9", nil, true), ans("q1", nil, false)})
	if len(errs) != 2 || errs[0].Field != "answers[1].question_id" || errs[1].Field != "answers[2].question_id" {
		t.Errorf("errors = %+v", errs)
	}
}

// ans membuat 1 jawaban yang dikirim FE (tanpa data waktu).
func ans(questionID string, answer json.RawMessage, isCorrect bool) SubmittedAnswer {
	return SubmittedAnswer{QuestionID: questionID, Answer: answer, IsCorrect: isCorrect}
}

func TestGrade_WaktuPerSoalDanTotal(t *testing.T) {
	at := time.Date(2026, 12, 1, 8, 5, 0, 0, time.UTC)
	spent := 42
	r, _ := Grade([]string{"q1", "q2"}, nil, PointsByDifficulty, []SubmittedAnswer{{QuestionID: "q1", IsCorrect: true, AnsweredAt: &at, TimeSpentSeconds: &spent}})
	if r.Answers[0].AnsweredAt == nil || !r.Answers[0].AnsweredAt.Equal(at) || *r.Answers[0].TimeSpentSeconds != 42 ||
		r.Answers[1].AnsweredAt != nil || r.Answers[1].TimeSpentSeconds != nil {
		t.Errorf("waktu per soal: %+v", r.Answers)
	}
	if got := DurationSeconds(at, at.Add(90*time.Second)); got != 90 {
		t.Errorf("durasi = %d, seharusnya 90", got)
	}
	if got := DurationSeconds(at, at.Add(-time.Second)); got != 0 {
		t.Errorf("durasi negatif = %d, seharusnya 0", got)
	}
}

func TestGrade_PointPerLevelBukanNilai(t *testing.T) {
	ids := []string{"easy", "medium", "hard", "salah"}
	levels := map[string]string{"easy": "easy", "medium": "medium", "hard": "hard", "salah": "hard"}
	r, _ := Grade(ids, levels, PointsByDifficulty, []SubmittedAnswer{
		ans("easy", nil, true), ans("medium", nil, true), ans("hard", nil, true), ans("salah", nil, false),
	})
	// Point: 10 + 15 + 20 + 0 = 45. Nilai: 3 dari 4 benar = 75 (nilai != point).
	if r.Points != 45 || r.Score != 75 {
		t.Errorf("point %d nilai %v, seharusnya point 45 & nilai 75", r.Points, r.Score)
	}
	want := []int{10, 15, 20, 0}
	for i, a := range r.Answers {
		if a.Points != want[i] {
			t.Errorf("point soal %s = %d, seharusnya %d", a.QuestionID, a.Points, want[i])
		}
	}
}

func TestGrade_PointDiaturPerQuiz(t *testing.T) {
	ids := []string{"easy", "hard"}
	levels := map[string]string{"easy": "easy", "hard": "hard"}
	table := map[string]int{"easy": 5, "medium": 25, "hard": 50} // contoh pengaturan admin
	r, _ := Grade(ids, levels, table, []SubmittedAnswer{ans("easy", nil, true), ans("hard", nil, true)})
	if r.Points != 55 {
		t.Errorf("point = %d, seharusnya 55 (5 + 50)", r.Points)
	}
}
