// Package clock berisi aturan waktu bisnis Rumbasha (zona WIB).
// Dipakai bersama oleh beberapa fitur, jadi tidak ditaruh di folder fitur tertentu.
package clock

import "time"

// Jakarta adalah zona waktu bisnis Rumbasha: WIB (UTC+7), sesuai BR-014.
//
// Dibuat dengan FixedZone (bukan time.LoadLocation("Asia/Jakarta")) karena
// Indonesia tidak memakai daylight saving, dan cara ini tidak butuh database
// zona waktu di sistem operasi (Windows sering tidak punya).
var Jakarta = time.FixedZone("WIB", 7*60*60)

// StartOfNextMonth mengembalikan tanggal 1 bulan berikutnya, jam 00:00 WIB.
// Inilah batas akhir semua sesi login: setiap awal bulan semua user login ulang.
//
// Contoh: 22 Nov 2026 10:00 WIB → 1 Des 2026 00:00 WIB
//
//	31 Des 2026 23:59 WIB → 1 Jan 2027 00:00 WIB
func StartOfNextMonth(now time.Time) time.Time {
	local := now.In(Jakarta)
	// time.Date otomatis menggeser bulan 13 menjadi Januari tahun berikutnya.
	return time.Date(local.Year(), local.Month()+1, 1, 0, 0, 0, 0, Jakarta)
}

// TodayInJakarta mengembalikan tanggal hari ini di WIB, format "2026-11-22".
func TodayInJakarta(now time.Time) string {
	return now.In(Jakarta).Format(time.DateOnly)
}

// Earliest mengembalikan waktu yang paling awal di antara a dan b.
func Earliest(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}
