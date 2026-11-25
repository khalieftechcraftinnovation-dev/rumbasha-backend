// Package coin (repository) berisi operasi saldo coin siswa yang dipakai beberapa fitur.
//
// Di package lain, import dengan alias coinrepo:
//
//	coinrepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/coin"
package coin

import (
	"time"

	"gorm.io/gorm"
)

// Reward adalah coin hadiah yang ditambahkan ke 1 siswa.
type Reward struct {
	StudentID      string
	Amount         int    // > 0
	ReferenceType  string // contoh quiz_prize, daily_quiz_prize
	ReferenceID    string // ID data asalnya (UUID)
	IdempotencyKey string // unik per hadiah, mencegah coin masuk 2x
	Description    string // maks 255 karakter (dipotong otomatis)
	At             time.Time
}

// AddReward menambah saldo coin siswa + mencatat coin_transactions type 'reward' di dalam
// transaksi tx (BR-012: perubahan saldo selalu bersama riwayatnya). [Native SQL, 2 query]
//
// UPDATE ... RETURNING sekaligus mengunci baris saldo siswa sampai transaksi selesai.
// ok = false kalau akun bukan siswa (tidak ada baris students): tidak ada yang berubah.
func AddReward(tx *gorm.DB, r Reward) (transactionID string, ok bool, err error) {
	var balances []int
	err = tx.Raw(`
		UPDATE students SET coin_balance = coin_balance + ?, updated_at = NOW()
		WHERE user_id = ?
		RETURNING coin_balance`, r.Amount, r.StudentID).
		Scan(&balances).Error
	if err != nil || len(balances) == 0 {
		return "", false, err
	}

	err = tx.Raw(`
		INSERT INTO coin_transactions
			(student_id, type, amount, balance_after, reference_type, reference_id, idempotency_key, description, created_at)
		VALUES (?, 'reward', ?, ?, ?, ?, ?, ?, ?)
		RETURNING id`,
		r.StudentID, r.Amount, balances[0], r.ReferenceType, r.ReferenceID, r.IdempotencyKey,
		truncate(r.Description, 255), r.At).
		Scan(&transactionID).Error
	if err != nil {
		return "", false, err
	}
	return transactionID, true, nil
}

// truncate memotong teks (per huruf, bukan byte) supaya muat di kolom VARCHAR(n).
func truncate(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n])
}
