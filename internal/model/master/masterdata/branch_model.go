package masterdata

import "time"

// Branch adalah 1 cabang Rumbasha (tabel branches) + jumlah akun yang memakainya.
type Branch struct {
	ID        string
	Name      string
	Status    string
	Users     int // akun (admin, tutor, siswa) yang belum dihapus di cabang ini
	CreatedBy *Ref
	UpdatedBy *Ref
	CreatedAt time.Time
	UpdatedAt time.Time
}

// BranchFilter adalah filter list cabang.
type BranchFilter struct {
	Search string
	Status string
}
