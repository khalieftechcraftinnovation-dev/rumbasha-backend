// Test point per level yang diatur admin & hadiah coin juara 1-3 quiz (dibagikan scheduler).
package router

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"

	questionrepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/question"
	quizrepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/quiz"
	quizservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/quiz"
)

type quizPointsPrizes struct {
	Points struct {
		Easy   int `json:"easy"`
		Medium int `json:"medium"`
		Hard   int `json:"hard"`
	} `json:"points"`
	PrizeCoins struct {
		Rank1 int `json:"rank_1"`
		Rank2 int `json:"rank_2"`
		Rank3 int `json:"rank_3"`
	} `json:"prize_coins"`
	PrizesDistributedAt *string `json:"prizes_distributed_at"`
	PrizeAwards         []struct {
		Rank       int     `json:"rank"`
		Student    listRef `json:"student"`
		Points     int     `json:"points"`
		CoinAmount int     `json:"coin_amount"`
	} `json:"prize_awards"`
}

func getQuizPointsPrizes(t *testing.T, r *gin.Engine, accessToken, id string) quizPointsPrizes {
	t.Helper()
	res := sendRequest(r, http.MethodGet, "/api/v1/quizzes/"+id, "", accessToken)
	if res.Code != http.StatusOK {
		t.Fatalf("detail quiz: status %d. body: %s", res.Code, res.Body)
	}
	var q quizPointsPrizes
	decodeData(t, res, &q)
	return q
}

func TestCreateQuiz_PointPerLevelDanHadiah(t *testing.T) {
	r, db := setupRouter(t)
	p := seedPlacementInGrade(t, db, "GRADE_6")
	admin := adminToken(t, db)
	q := createQuestionID(t, r, admin, questionBody(p, "single_choice", singleChoiceAnswer, nil))

	// Tidak diisi → default point 10/15/20 & tanpa hadiah.
	plain := createQuiz(t, r, admin, quizBody(basePlacement(p), []string{q}, nil))
	got := getQuizPointsPrizes(t, r, admin, plain.ID)
	if got.Points.Easy != 10 || got.Points.Medium != 15 || got.Points.Hard != 20 ||
		got.PrizeCoins.Rank1 != 0 || got.PrizesDistributedAt != nil || got.PrizeAwards == nil || len(got.PrizeAwards) != 0 {
		t.Errorf("default: %+v", got)
	}

	// Diisi sebagian: level yang tidak diisi tetap default.
	custom := createQuiz(t, r, admin, quizBody(basePlacement(p), []string{q}, map[string]string{
		"points":      `{"easy": 5, "hard": 50}`,
		"prize_coins": `{"rank_1": 500, "rank_2": 300, "rank_3": 100}`,
	}))
	got = getQuizPointsPrizes(t, r, admin, custom.ID)
	if got.Points.Easy != 5 || got.Points.Medium != 15 || got.Points.Hard != 50 ||
		got.PrizeCoins.Rank1 != 500 || got.PrizeCoins.Rank2 != 300 || got.PrizeCoins.Rank3 != 100 {
		t.Errorf("custom: %+v", got)
	}

	// Update mengganti pengaturan (ganti penuh: tidak dikirim = kembali default).
	res := sendRequest(r, http.MethodPut, "/api/v1/quizzes/"+custom.ID, quizBody(basePlacement(p), []string{q}, map[string]string{
		"points": `{"medium": 30}`, "prize_coins": `{"rank_1": 1000}`,
	}), admin)
	if res.Code != http.StatusOK {
		t.Fatalf("update: status %d. body: %s", res.Code, res.Body)
	}
	got = getQuizPointsPrizes(t, r, admin, custom.ID)
	if got.Points.Easy != 10 || got.Points.Medium != 30 || got.PrizeCoins.Rank1 != 1000 || got.PrizeCoins.Rank2 != 0 {
		t.Errorf("update: %+v", got)
	}

	// Nilai di luar batas → 422 dengan nama field.
	res = sendRequest(r, http.MethodPost, "/api/v1/quizzes", quizBody(basePlacement(p), []string{q}, map[string]string{
		"points": `{"easy": -1, "hard": 1001}`, "prize_coins": `{"rank_2": -5}`,
	}), admin)
	fields := errorFields(t, res.Body.String())
	if res.Code != http.StatusUnprocessableEntity || fields["points.easy"] == "" || fields["points.hard"] == "" || fields["prize_coins.rank_2"] == "" {
		t.Errorf("validasi: status %d, errors %v", res.Code, fields)
	}
}

// Point saat submit memakai pengaturan quiz, bukan 10/15/20.
func TestSubmitQuiz_PointSesuaiPengaturanQuiz(t *testing.T) {
	r, db := setupRouter(t)
	p := seedPlacementInGrade(t, db, "GRADE_6")
	admin := adminToken(t, db)
	easy := createQuestionID(t, r, admin, questionBody(p, "single_choice", singleChoiceAnswer, nil))
	hard := createQuestionID(t, r, admin, questionBody(p, "single_choice", singleChoiceAnswer, map[string]string{"difficulty": `"hard"`}))
	quiz := createQuiz(t, r, admin, quizBody(basePlacement(p), []string{easy, hard}, map[string]string{
		"status": `"published"`, "points": `{"easy": 7, "hard": 40}`,
	}))
	db.Exec(`UPDATE quizzes SET start_at = NOW() - INTERVAL '1 hour', end_at = NOW() + INTERVAL '1 hour' WHERE id = ?`, quiz.ID)

	studentID, student := namedStudent(t, db, "Ani", "GRADE_6")
	if code, body := openQuiz(r, student, quiz.ID, fmt.Sprintf(`{"access_code":%q}`, quiz.AccessCode)); code != http.StatusOK {
		t.Fatalf("buka: status %d, body %s", code, body)
	}
	res := sendRequest(r, http.MethodPost, "/api/v1/quizzes/"+quiz.ID+"/submit", fmt.Sprintf(
		`{"answers":[{"question_id":%q,"answer":"B","is_correct":true},{"question_id":%q,"answer":"B","is_correct":true}]}`, easy, hard), student)
	if res.Code != http.StatusOK {
		t.Fatalf("submit: status %d. body: %s", res.Code, res.Body)
	}
	var review reviewData
	decodeData(t, res, &review)
	if review.Result.Points != 47 || review.Questions[0].Points != 7 || review.Questions[1].Points != 40 || review.Result.Score != 100 {
		t.Errorf("point: total %d, per soal %d & %d, nilai %v; seharusnya 47 (7 + 40), nilai 100",
			review.Result.Points, review.Questions[0].Points, review.Questions[1].Points, review.Result.Score)
	}
	var points int
	db.Raw(`SELECT points FROM leaderboard_entries WHERE quiz_id = ? AND student_id = ?`, quiz.ID, studentID).Scan(&points)
	if points != 47 {
		t.Errorf("leaderboard: point %d, seharusnya 47", points)
	}
}

func TestDistributePrizes_JuaraMendapatCoinSekali(t *testing.T) {
	r, db := setupRouter(t)
	p := seedPlacementInGrade(t, db, "GRADE_6")
	admin := adminToken(t, db)
	service := quizservice.NewQuizService(quizrepo.NewQuizRepository(db), questionrepo.NewQuestionRepository(db))
	q := createQuestionID(t, r, admin, questionBody(p, "single_choice", singleChoiceAnswer, map[string]string{"used_for": `["uji_kom","tryout"]`}))

	prizes := map[string]string{"status": `"published"`, "prize_coins": `{"rank_1": 500, "rank_2": 300, "rank_3": 100}`}
	ended := createQuiz(t, r, admin, quizBody(basePlacement(p), []string{q}, prizes))
	running := createQuiz(t, r, admin, quizBody(basePlacement(p), []string{q}, prizes))
	draft := createQuiz(t, r, admin, quizBody(basePlacement(p), []string{q}, map[string]string{"prize_coins": `{"rank_1": 500}`}))
	noPrize := createQuiz(t, r, admin, quizBody(basePlacement(p), []string{q}, map[string]string{"status": `"published"`, "type": `"tryout"`}))
	// Jadwal dibuat tidak beririsan (mapel & kelas sama).
	db.Exec(`UPDATE quizzes SET start_at = NOW() - INTERVAL '10 hours', end_at = NOW() - INTERVAL '8 hours' WHERE id = ?`, ended.ID)
	db.Exec(`UPDATE quizzes SET start_at = NOW() - INTERVAL '7 hours', end_at = NOW() - INTERVAL '5 hours' WHERE id = ?`, draft.ID)
	db.Exec(`UPDATE quizzes SET start_at = NOW() - INTERVAL '4 hours', end_at = NOW() - INTERVAL '2 hours' WHERE id = ?`, noPrize.ID)
	db.Exec(`UPDATE quizzes SET start_at = NOW() - INTERVAL '1 hour', end_at = NOW() + INTERVAL '1 hour' WHERE id = ?`, running.ID)

	// Peserta: Budi & Cici sama persis (juara 2 bersama, keduanya dapat hadiah juara 2),
	// lalu Dodi juara 4 (RANK lompat: 1, 2, 2, 4) → tidak dapat hadiah juara 3.
	ani, aniToken := namedStudent(t, db, "Ani", "GRADE_6")
	budi, _ := namedStudent(t, db, "Budi", "GRADE_6")
	cici, _ := namedStudent(t, db, "Cici", "GRADE_6")
	dodi, _ := namedStudent(t, db, "Dodi", "GRADE_6")
	db.Exec(`UPDATE students SET coin_balance = 20 WHERE user_id = ?`, ani)
	entry := func(quizID, studentID string, points, duration int) {
		t.Helper()
		err := db.Exec(`
			INSERT INTO leaderboard_entries (board, quiz_id, student_id, education_level_id, grade_id,
				points, duration_seconds, quiz_count, score, last_submitted_at)
			SELECT z.type, z.id, u.id, u.education_level_id, u.grade_id, ?, ?, 1, 100, NOW()
			FROM quizzes z, users u WHERE z.id = ? AND u.id = ?`, points, duration, quizID, studentID).Error
		if err != nil {
			t.Fatal(err)
		}
	}
	entry(ended.ID, ani, 50, 300)
	entry(ended.ID, budi, 40, 200)
	entry(ended.ID, cici, 40, 200)
	entry(ended.ID, dodi, 40, 250)
	entry(running.ID, ani, 50, 100)
	entry(noPrize.ID, ani, 50, 100)

	// Sebelum dibagikan: leaderboard sudah menampilkan hadiah per peringkat.
	b, _ := getBoard(t, r, aniToken, "/api/v1/leaderboards/quizzes/"+ended.ID)
	if got := ranking(b.Entries); got != "1:Ani,2:Budi,2:Cici,4:Dodi" {
		t.Fatalf("peringkat: %s", got)
	}

	result, err := service.DistributePrizes(context.Background())
	// Diproses: ended (3 penerima) & noPrize (tanpa hadiah, hanya ditandai). running & draft tidak.
	if err != nil || result.Quizzes != 2 || result.Awards != 3 || result.Coins != 1100 || result.Failed != 0 {
		t.Fatalf("bagi hadiah: %+v, err %v", result, err)
	}

	balance := func(id string) int {
		var b int
		db.Raw(`SELECT coin_balance FROM students WHERE user_id = ?`, id).Scan(&b)
		return b
	}
	for _, tc := range []struct {
		name, id string
		want     int
	}{{"Ani juara 1", ani, 520}, {"Budi juara 2", budi, 300}, {"Cici juara 2", cici, 300}, {"Dodi juara 4", dodi, 0}} {
		if got := balance(tc.id); got != tc.want {
			t.Errorf("%s: saldo %d, seharusnya %d", tc.name, got, tc.want)
		}
	}

	// Riwayat coin: type reward, saldo setelahnya, referensi quiz.
	var tx struct {
		Type, ReferenceType, ReferenceID string
		Amount, BalanceAfter             int
	}
	db.Raw(`SELECT type, reference_type, reference_id, amount, balance_after FROM coin_transactions WHERE student_id = ?`, ani).Scan(&tx)
	if tx.Type != "reward" || tx.ReferenceType != "quiz_prize" || tx.ReferenceID != ended.ID || tx.Amount != 500 || tx.BalanceAfter != 520 {
		t.Errorf("coin_transactions: %+v", tx)
	}

	// Detail quiz (admin): tanggal dibagikan + daftar penerima. Quiz lain belum/tidak dibagikan.
	got := getQuizPointsPrizes(t, r, admin, ended.ID)
	if got.PrizesDistributedAt == nil || len(got.PrizeAwards) != 3 || got.PrizeAwards[0].Rank != 1 ||
		got.PrizeAwards[0].Student.ID != ani || got.PrizeAwards[0].CoinAmount != 500 || got.PrizeAwards[2].Rank != 2 {
		t.Errorf("detail ended: %+v", got)
	}
	for name, id := range map[string]string{"running": running.ID, "draft": draft.ID} {
		if got := getQuizPointsPrizes(t, r, admin, id); got.PrizesDistributedAt != nil {
			t.Errorf("%s seharusnya belum dibagikan", name)
		}
	}
	if got := getQuizPointsPrizes(t, r, admin, noPrize.ID); got.PrizesDistributedAt == nil || len(got.PrizeAwards) != 0 {
		t.Errorf("tanpa hadiah: seharusnya ditandai selesai tanpa penerima: %+v", got)
	}

	// Leaderboard: hadiah per peringkat & tanggal dibagikan.
	var lb struct {
		Quiz struct {
			PrizesDistributedAt *string `json:"prizes_distributed_at"`
		} `json:"quiz"`
		Entries []struct {
			PrizeCoins *int `json:"prize_coins"`
		} `json:"entries"`
	}
	decodeData(t, sendRequest(r, http.MethodGet, "/api/v1/leaderboards/quizzes/"+ended.ID, "", aniToken), &lb)
	if lb.Quiz.PrizesDistributedAt == nil || len(lb.Entries) != 4 || *lb.Entries[0].PrizeCoins != 500 ||
		*lb.Entries[1].PrizeCoins != 300 || *lb.Entries[3].PrizeCoins != 0 {
		t.Errorf("leaderboard hadiah: %+v", lb)
	}

	// Dijalankan lagi (contoh 2 server / jadwal berikutnya) → tidak ada coin dobel.
	result, err = service.DistributePrizes(context.Background())
	if err != nil || result.Quizzes != 0 || result.Awards != 0 || balance(ani) != 520 {
		t.Errorf("jalan ke-2: %+v, err %v, saldo Ani %d", result, err, balance(ani))
	}
}
