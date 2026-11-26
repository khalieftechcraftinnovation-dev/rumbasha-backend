// Test pengaturan global Daily Quiz (point per level & hadiah juara bulanan) dan pembagian hadiah
// juara 1-3 per kelas.
package router

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/clock"
	loginmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/auth/login"
	dailyquizrepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/quiz/dailyquiz"
	questionrepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/quiz/question"
	dailyquizservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/quiz/dailyquiz"
)

type dailySettings struct {
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
	FreePerDay int      `json:"free_per_day"`
	CoinCost   int      `json:"coin_cost"`
	UpdatedBy  *listRef `json:"updated_by"`
}

const settingsURL = "/api/v1/daily-quiz-settings"

func TestDailyQuizSettings_PointGlobalDipakaiSaatSubmit(t *testing.T) {
	r, db := setupRouter(t)
	p := seedPlacementInGrade(t, db, "GRADE_5")
	admin := adminToken(t, db)
	for i := 0; i < 5; i++ { // 5 soal easy
		createQuestionID(t, r, admin, questionBody(p, "single_choice", singleChoiceAnswer, nil))
	}
	_, student := namedStudent(t, db, "Ani", "GRADE_5")

	// Nilai awal: 10/15/20, tanpa hadiah. Semua role yang login boleh melihat.
	var s dailySettings
	decodeData(t, sendRequest(r, http.MethodGet, settingsURL, "", student), &s)
	if s.Points.Easy != 10 || s.Points.Medium != 15 || s.Points.Hard != 20 || s.PrizeCoins.Rank1 != 0 ||
		s.FreePerDay != 2 || s.CoinCost != 10 || s.UpdatedBy != nil {
		t.Errorf("nilai awal: %+v", s)
	}

	body := `{"points":{"easy":7,"medium":12,"hard":30},"prize_coins":{"rank_1":300,"rank_2":200,"rank_3":100},"free_per_day":2,"coin_cost":10}`
	// Hanya admin & super admin yang boleh mengubah.
	for _, tok := range []string{student, tokenFor(t, loginmodel.RoleTutor)} {
		if res := sendRequest(r, http.MethodPut, settingsURL, body, tok); res.Code != http.StatusForbidden {
			t.Errorf("ubah oleh bukan admin: status %d, seharusnya 403", res.Code)
		}
	}
	// Ganti penuh: semua field wajib & dalam batas.
	res := sendRequest(r, http.MethodPut, settingsURL, `{"points":{"easy":-1,"medium":12},"prize_coins":{"rank_1":300,"rank_2":200,"rank_3":100},"free_per_day":101}`, admin)
	fields := errorFields(t, res.Body.String())
	if res.Code != http.StatusUnprocessableEntity || fields["points.easy"] == "" || fields["points.hard"] == "" ||
		fields["free_per_day"] == "" || fields["coin_cost"] == "" {
		t.Errorf("validasi: status %d, errors %v", res.Code, fields)
	}

	res = sendRequest(r, http.MethodPut, settingsURL, body, admin)
	if res.Code != http.StatusOK {
		t.Fatalf("ubah pengaturan: status %d. body: %s", res.Code, res.Body)
	}
	decodeData(t, res, &s)
	if s.Points.Easy != 7 || s.Points.Hard != 30 || s.PrizeCoins.Rank1 != 300 || s.UpdatedBy == nil || s.UpdatedBy.ID == "" {
		t.Errorf("setelah diubah: %+v", s)
	}

	// Daily Quiz yang di-submit setelahnya memakai point baru: 5 soal easy benar semua = 5 x 7.
	placement := `{"subject_id":"` + p.SubjectID + `","subject_field_id":"` + p.FieldID + `"}`
	code, generatedBody := generateDaily(r, student, dailyBody(placement, "easy", 5))
	if code != http.StatusCreated {
		t.Fatalf("generate: status %d", code)
	}
	var generated struct {
		ID        string `json:"id"`
		Questions []struct {
			ID string `json:"id"`
		} `json:"questions"`
	}
	decodeJSONData(t, generatedBody, &generated)
	sendRequest(r, http.MethodPost, "/api/v1/daily-quizzes/"+generated.ID+"/open", "", student)
	items := []string{}
	for _, q := range generated.Questions {
		items = append(items, `{"question_id":"`+q.ID+`","answer":"B","is_correct":true}`)
	}
	res = sendRequest(r, http.MethodPost, "/api/v1/daily-quizzes/"+generated.ID+"/submit", `{"answers":[`+strings.Join(items, ",")+`]}`, student)
	var review reviewData
	decodeData(t, res, &review)
	if review.Result.Points != 35 || review.Questions[0].Points != 7 {
		t.Errorf("point Daily Quiz: total %d, per soal %d; seharusnya 35 & 7", review.Result.Points, review.Questions[0].Points)
	}
}

func TestDailyQuizMonthlyPrizes_JuaraPerKelas(t *testing.T) {
	r, db := setupRouter(t)
	admin := adminToken(t, db)
	service := dailyquizservice.NewDailyQuizService(dailyquizrepo.NewDailyQuizRepository(db), questionrepo.NewQuestionRepository(db))
	if res := sendRequest(r, http.MethodPut, settingsURL,
		`{"points":{"easy":10,"medium":15,"hard":20},"prize_coins":{"rank_1":300,"rank_2":200,"rank_3":100},"free_per_day":2,"coin_cost":10}`, admin); res.Code != http.StatusOK {
		t.Fatalf("atur hadiah: status %d", res.Code)
	}

	// Bulan lalu (sudah ditutup): Kelas 5 → Ani 1, Budi & Cici 2 (sama persis), Dodi 4.
	// Kelas 6 → Eko juara 1 kelasnya walau point-nya kecil. Bulan ini: belum dibagikan.
	lastMonth := `(NOW() - INTERVAL '1 month')`
	ani, aniToken := namedStudent(t, db, "Ani", "GRADE_5")
	budi, _ := namedStudent(t, db, "Budi", "GRADE_5")
	cici, _ := namedStudent(t, db, "Cici", "GRADE_5")
	dodi, _ := namedStudent(t, db, "Dodi", "GRADE_5")
	eko, _ := namedStudent(t, db, "Eko", "GRADE_6")
	submittedDaily(t, db, ani, 100, 500, lastMonth)
	submittedDaily(t, db, budi, 80, 300, lastMonth)
	submittedDaily(t, db, cici, 80, 300, lastMonth)
	submittedDaily(t, db, dodi, 50, 100, lastMonth)
	submittedDaily(t, db, eko, 30, 100, lastMonth)
	submittedDaily(t, db, ani, 70, 100, "NOW()")

	result, err := service.DistributeMonthlyPrizes(context.Background())
	// 1 bulan (bulan lalu): Ani 300, Budi 200, Cici 200, Eko 300 = 1000 coin. Dodi peringkat 4.
	if err != nil || result.Periods != 1 || result.Awards != 4 || result.Coins != 1000 || result.Failed != 0 {
		t.Fatalf("bagi hadiah: %+v, err %v", result, err)
	}
	balance := func(id string) int {
		var b int
		db.Raw(`SELECT coin_balance FROM students WHERE user_id = ?`, id).Scan(&b)
		return b
	}
	for name, want := range map[string]struct {
		id    string
		coins int
	}{"Ani": {ani, 300}, "Budi": {budi, 200}, "Cici": {cici, 200}, "Dodi": {dodi, 0}, "Eko": {eko, 300}} {
		if got := balance(want.id); got != want.coins {
			t.Errorf("%s: saldo %d, seharusnya %d", name, got, want.coins)
		}
	}
	var refType string
	db.Raw(`SELECT reference_type FROM coin_transactions WHERE student_id = ? AND type = 'reward'`, eko).Scan(&refType)
	if refType != "daily_quiz_prize" {
		t.Errorf("coin_transactions reference_type = %q", refType)
	}

	// Admin: daftar penerima bulan lalu (urut kelas, lalu peringkat). Bulan ini belum dibagikan.
	month := time.Now().In(clock.Jakarta).AddDate(0, -1, 0).Format("2006-01")
	if now := time.Now().In(clock.Jakarta); now.Day() > 28 { // AddDate di akhir bulan bisa melompat
		month = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, clock.Jakarta).AddDate(0, -1, 0).Format("2006-01")
	}
	var prizes struct {
		DistributedAt *string `json:"distributed_at"`
		PrizeCoins    *struct {
			Rank1 int `json:"rank_1"`
		} `json:"prize_coins"`
		Awards []struct {
			Grade   listRef `json:"grade"`
			Rank    int     `json:"rank"`
			Student struct {
				Name string `json:"name"`
			} `json:"student"`
			CoinAmount int `json:"coin_amount"`
		} `json:"awards"`
	}
	decodeData(t, sendRequest(r, http.MethodGet, "/api/v1/daily-quiz-prizes?month="+month, "", admin), &prizes)
	got := []string{}
	for _, a := range prizes.Awards {
		got = append(got, a.Grade.Name+":"+strconv.Itoa(a.Rank)+":"+a.Student.Name+":"+strconv.Itoa(a.CoinAmount))
	}
	if prizes.DistributedAt == nil || prizes.PrizeCoins == nil || prizes.PrizeCoins.Rank1 != 300 ||
		strings.Join(got, ",") != "Kelas 5:1:Ani:300,Kelas 5:2:Budi:200,Kelas 5:2:Cici:200,Kelas 6:1:Eko:300" {
		t.Errorf("daftar penerima: %v (distributed %v)", got, prizes.DistributedAt)
	}
	var current struct {
		DistributedAt *string `json:"distributed_at"`
		Awards        []any   `json:"awards"`
	}
	decodeData(t, sendRequest(r, http.MethodGet, "/api/v1/daily-quiz-prizes", "", admin), &current)
	if current.DistributedAt != nil || len(current.Awards) != 0 {
		t.Errorf("bulan ini seharusnya belum dibagikan: %+v", current)
	}
	if res := sendRequest(r, http.MethodGet, "/api/v1/daily-quiz-prizes", "", aniToken); res.Code != http.StatusForbidden {
		t.Errorf("siswa melihat daftar penerima: status %d, seharusnya 403", res.Code)
	}

	// Hadiah diubah setelah dibagikan → bulan lalu tetap memakai snapshot saat dibagikan.
	sendRequest(r, http.MethodPut, settingsURL, `{"points":{"easy":10,"medium":15,"hard":20},"prize_coins":{"rank_1":999,"rank_2":0,"rank_3":0},"free_per_day":2,"coin_cost":10}`, admin)

	// Leaderboard difilter kelas: hadiah per peringkat + tanggal dibagikan.
	var lb struct {
		PrizeCoins struct {
			Rank1 int `json:"rank_1"`
		} `json:"prize_coins"`
		PrizesDistributedAt *string `json:"prizes_distributed_at"`
		Entries             []struct {
			PrizeCoins *int `json:"prize_coins"`
		} `json:"entries"`
	}
	var gradeID string
	db.Raw(`SELECT id FROM master_data_items WHERE type = 'grade' AND code = 'GRADE_5'`).Scan(&gradeID)
	decodeData(t, sendRequest(r, http.MethodGet, "/api/v1/leaderboards/daily-quiz?month="+month+"&grade_id="+gradeID, "", aniToken), &lb)
	if lb.PrizeCoins.Rank1 != 300 || lb.PrizesDistributedAt == nil || len(lb.Entries) != 4 ||
		lb.Entries[0].PrizeCoins == nil || *lb.Entries[0].PrizeCoins != 300 || *lb.Entries[3].PrizeCoins != 0 {
		t.Errorf("leaderboard kelas 5: %+v", lb)
	}
	// Bulan ini: belum dibagikan, hadiah = pengaturan terbaru; tanpa filter kelas tidak ada hadiah per baris.
	var lbNow struct {
		PrizeCoins struct {
			Rank1 int `json:"rank_1"`
		} `json:"prize_coins"`
		PrizesDistributedAt *string `json:"prizes_distributed_at"`
		Entries             []struct {
			PrizeCoins *int `json:"prize_coins"`
		} `json:"entries"`
	}
	decodeData(t, sendRequest(r, http.MethodGet, "/api/v1/leaderboards/daily-quiz", "", aniToken), &lbNow)
	if lbNow.PrizeCoins.Rank1 != 999 || lbNow.PrizesDistributedAt != nil || len(lbNow.Entries) != 1 || lbNow.Entries[0].PrizeCoins != nil {
		t.Errorf("leaderboard bulan ini: %+v", lbNow)
	}

	// Dijalankan lagi → tidak ada coin dobel.
	result, err = service.DistributeMonthlyPrizes(context.Background())
	if err != nil || result.Periods != 0 || balance(ani) != 300 {
		t.Errorf("jalan ke-2: %+v, err %v, saldo Ani %d", result, err, balance(ani))
	}
}

func TestPrizeCloseAt_HariTerakhirBulan2359(t *testing.T) {
	for period, want := range map[string]string{
		"2026-11": "2026-11-30 23:59",
		"2026-12": "2026-12-31 23:59",
		"2028-02": "2028-02-29 23:59", // kabisat
	} {
		got, ok := dailyquizservice.PrizeCloseAt(period)
		if !ok || got.In(clock.Jakarta).Format("2006-01-02 15:04") != want {
			t.Errorf("%s: %v, seharusnya %s WIB", period, got, want)
		}
	}
	if _, ok := dailyquizservice.PrizeCloseAt("2026-13"); ok {
		t.Error("period rusak seharusnya ditolak")
	}
}
