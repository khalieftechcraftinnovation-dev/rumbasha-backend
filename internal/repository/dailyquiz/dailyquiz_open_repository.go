package dailyquiz

import (
	"context"
	"time"

	dailyquizmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/dailyquiz"
)

// Open membuka Daily Quiz milik siswa dalam 1 transaksi (aman dari 2 request bersamaan):
//
//  1. Kunci baris saldo coin siswa (students FOR UPDATE): request buka berikutnya dari siswa
//     yang sama menunggu sampai transaksi ini selesai, jadi jatah gratis tidak bisa terpakai 2x.
//  2. Ambil Daily Quiz milik siswa. Sudah pernah dibuka → selesai (tidak dihitung & dipotong lagi).
//  3. Hitung Daily Quiz yang sudah dibuka hari ini (sejak dayStart), lalu decideCost (service)
//     menentukan biaya: 0 kalau masih ada jatah gratis, atau error kalau coin tidak cukup.
//  4. Kalau ada biaya: potong saldo + catat coin_transactions (type usage, referensi daily_quiz).
//  5. Tandai Daily Quiz sudah dibuka (opened_at, coin_charged).
//
// [Native SQL]
func (r *DailyQuizRepository) Open(ctx context.Context, quizID, studentID string, now, dayStart time.Time,
	decideCost func(openedToday, balance int) (int, error)) (dailyquizmodel.OpenResult, error) {
	var result dailyquizmodel.OpenResult
	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return result, tx.Error
	}
	defer tx.Rollback() // tidak berpengaruh kalau sudah Commit

	var balances []int
	if err := tx.Raw(`SELECT coin_balance FROM students WHERE user_id = ? FOR UPDATE`, studentID).Scan(&balances).Error; err != nil {
		return result, err
	}
	if len(balances) == 0 {
		return result, dailyquizmodel.ErrDailyQuizNotFound // akun bukan siswa
	}
	result.CoinBalance = balances[0]

	var quizzes []struct {
		OpenedAt    *time.Time
		CoinCharged int
	}
	err := tx.Raw(`SELECT opened_at, coin_charged FROM daily_quizzes WHERE id = ? AND student_id = ? FOR UPDATE`,
		quizID, studentID).Scan(&quizzes).Error
	if err != nil {
		return result, err
	}
	if len(quizzes) == 0 {
		return result, dailyquizmodel.ErrDailyQuizNotFound
	}

	// Pakai index daily_quizzes_student_opened_idx.
	err = tx.Raw(`SELECT COUNT(*) FROM daily_quizzes WHERE student_id = ? AND opened_at >= ?`, studentID, dayStart).
		Scan(&result.OpenedToday).Error
	if err != nil {
		return result, err
	}

	if quizzes[0].OpenedAt != nil { // dibuka ulang
		result.AlreadyOpened = true
		result.OpenedAt = *quizzes[0].OpenedAt
		result.CoinCharged = quizzes[0].CoinCharged
		return result, tx.Commit().Error
	}

	cost, err := decideCost(result.OpenedToday, result.CoinBalance)
	if err != nil {
		return result, err // ROLLBACK lewat defer
	}
	if cost > 0 {
		result.CoinBalance -= cost
		err := tx.Exec(`UPDATE students SET coin_balance = ?, updated_at = NOW() WHERE user_id = ?`,
			result.CoinBalance, studentID).Error
		if err != nil {
			return result, err
		}
		// Setiap pemakaian coin wajib tercatat dengan referensi (BR-012). idempotency_key
		// mencegah 1 Daily Quiz memotong coin 2x.
		err = tx.Exec(`
			INSERT INTO coin_transactions
				(student_id, type, amount, balance_after, reference_type, reference_id, idempotency_key, description)
			VALUES (?, 'usage', ?, ?, 'daily_quiz', ?, ?, 'Daily Quiz (jatah gratis hari ini habis)')`,
			studentID, -cost, result.CoinBalance, quizID, "daily_quiz_open:"+quizID).Error
		if err != nil {
			return result, err
		}
	}

	err = tx.Exec(`UPDATE daily_quizzes SET opened_at = ?, coin_charged = ? WHERE id = ?`, now, cost, quizID).Error
	if err != nil {
		return result, err
	}
	if err := tx.Commit().Error; err != nil {
		return result, err
	}
	result.OpenedAt = now
	result.CoinCharged = cost
	result.OpenedToday++
	return result, nil
}

// FindDetail mengambil Daily Quiz milik siswa beserta nama letak & ID soal (urut nomor).
// Bukan milik siswa / tidak ada → ErrDailyQuizNotFound. [Native SQL, 2 query]
func (r *DailyQuizRepository) FindDetail(ctx context.Context, quizID, studentID string) (dailyquizmodel.DailyQuizDetail, error) {
	var rows []struct {
		ID, StudentID                                        string
		EducationLevelID, GradeID, SubjectID, SubjectFieldID string
		ChapterID, SubchapterID, TopicID                     *string
		Difficulty                                           string
		TotalQuestions                                       int
		CreatedAt                                            time.Time
		EducationLevelName, GradeName                        string
		SubjectName, SubjectFieldName                        string
		ChapterName, SubchapterName, TopicName               *string
	}
	err := r.db.WithContext(ctx).
		Raw(`
			SELECT d.id, d.student_id, d.education_level_id, d.grade_id, d.subject_id, d.subject_field_id,
			       d.chapter_id, d.subchapter_id, d.topic_id, d.difficulty, d.total_questions, d.created_at,
			       level.name   AS education_level_name, grade.name AS grade_name,
			       subject.name AS subject_name,         field.name AS subject_field_name,
			       chapter.name AS chapter_name, sub.name AS subchapter_name, topic.name AS topic_name
			FROM daily_quizzes d
			JOIN      master_data_items AS level   ON level.id   = d.education_level_id
			JOIN      master_data_items AS grade   ON grade.id   = d.grade_id
			JOIN      master_data_items AS subject ON subject.id = d.subject_id
			JOIN      master_data_items AS field   ON field.id   = d.subject_field_id
			LEFT JOIN master_data_items AS chapter ON chapter.id = d.chapter_id
			LEFT JOIN master_data_items AS sub     ON sub.id     = d.subchapter_id
			LEFT JOIN master_data_items AS topic   ON topic.id   = d.topic_id
			WHERE d.id = ? AND d.student_id = ?`, quizID, studentID).
		Scan(&rows).Error
	if err != nil {
		return dailyquizmodel.DailyQuizDetail{}, err
	}
	if len(rows) == 0 {
		return dailyquizmodel.DailyQuizDetail{}, dailyquizmodel.ErrDailyQuizNotFound
	}
	row := rows[0]

	questionIDs := []string{}
	err = r.db.WithContext(ctx).
		Raw(`SELECT question_id FROM daily_quiz_questions WHERE daily_quiz_id = ? ORDER BY sort_order`, quizID).
		Scan(&questionIDs).Error
	if err != nil {
		return dailyquizmodel.DailyQuizDetail{}, err
	}

	return dailyquizmodel.DailyQuizDetail{
		DailyQuiz: dailyquizmodel.DailyQuiz{
			ID: row.ID, StudentID: row.StudentID,
			EducationLevelID: row.EducationLevelID, GradeID: row.GradeID,
			SubjectID: row.SubjectID, SubjectFieldID: row.SubjectFieldID,
			ChapterID: row.ChapterID, SubchapterID: row.SubchapterID, TopicID: row.TopicID,
			Difficulty: row.Difficulty, TotalQuestions: row.TotalQuestions,
			QuestionIDs: questionIDs, CreatedAt: row.CreatedAt,
		},
		EducationLevelName: row.EducationLevelName, GradeName: row.GradeName,
		SubjectName: row.SubjectName, SubjectFieldName: row.SubjectFieldName,
		ChapterName: row.ChapterName, SubchapterName: row.SubchapterName, TopicName: row.TopicName,
	}, nil
}
