package service

import "time"

// jakarta adalah zona waktu bisnis Rumbasha: WIB (UTC+7), sesuai BR-014.
//
// Dibuat dengan FixedZone (bukan time.LoadLocation("Asia/Jakarta")) karena
// Indonesia tidak memakai daylight saving, dan cara ini tidak butuh database
// zona waktu di sistem operasi (Windows sering tidak punya).
var jakarta = time.FixedZone("WIB", 7*60*60)

// startOfNextMonth mengembalikan tanggal 1 bulan berikutnya, jam 00:00 WIB.
// Inilah batas akhir semua sesi login: setiap awal bulan semua user login ulang.
//
// Contoh: 22 Nov 2026 10:00 WIB → 1 Des 2026 00:00 WIB
//
//	31 Des 2026 23:59 WIB → 1 Jan 2027 00:00 WIB
func startOfNextMonth(now time.Time) time.Time {
	local := now.In(jakarta)
	// time.Date otomatis menggeser bulan 13 menjadi Januari tahun berikutnya.
	return time.Date(local.Year(), local.Month()+1, 1, 0, 0, 0, 0, jakarta)
}

// todayInJakarta mengembalikan tanggal hari ini di WIB, format "2026-11-22".
func todayInJakarta(now time.Time) string {
	return now.In(jakarta).Format(time.DateOnly)
}

// earliest mengembalikan waktu yang paling awal di antara a dan b.
func earliest(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}
