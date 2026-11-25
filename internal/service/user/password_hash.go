package user

import (
	"runtime"
	"sync"

	loginservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/login"
)

// hashPasswords membuat hash bcrypt banyak password secara paralel (worker = jumlah CPU).
// bcrypt sengaja lambat (±50-80 ms per password), jadi 100 password berurutan bisa ±7 detik;
// dengan 8 CPU menjadi ±1 detik. Hasil urut sesuai input.
func hashPasswords(passwords []string) ([]string, error) {
	hashes := make([]string, len(passwords))
	errs := make([]error, len(passwords))

	workers := min(runtime.NumCPU(), len(passwords))
	jobs := make(chan int)
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range jobs {
				hashes[i], errs[i] = loginservice.HashPassword(passwords[i])
			}
		}()
	}
	for i := range passwords {
		jobs <- i
	}
	close(jobs)
	wg.Wait()

	for _, err := range errs {
		if err != nil {
			return nil, err
		}
	}
	return hashes, nil
}
