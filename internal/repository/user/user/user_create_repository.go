package user

import (
	"context"
	"strings"
	"time"

	"gorm.io/gorm"

	usermodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/user/user"
)

// ActiveBranches mengambil cabang aktif (belum dihapus) dari daftar ID, 1 query. Hasilnya set ID.
func (r *UserRepository) ActiveBranches(ctx context.Context, ids []string) (map[string]bool, error) {
	result := map[string]bool{}
	if len(ids) == 0 {
		return result, nil
	}
	var found []string
	err := r.db.WithContext(ctx).
		Raw(`SELECT id FROM branches WHERE id IN ? AND deleted_at IS NULL AND status = 'active'`, ids).
		Scan(&found).Error
	for _, id := range found {
		result[id] = true
	}
	return result, err
}

// TakenUsernames mengambil username yang sudah dipakai (TERMASUK akun yang sudah dihapus, karena
// username unik selamanya), 1 query lewat index unik users_username_unique. Hasilnya set username.
func (r *UserRepository) TakenUsernames(ctx context.Context, usernames []string) (map[string]bool, error) {
	result := map[string]bool{}
	if len(usernames) == 0 {
		return result, nil
	}
	var found []string
	err := r.db.WithContext(ctx).Raw(`SELECT username FROM users WHERE username IN ?`, usernames).Scan(&found).Error
	for _, u := range found {
		result[u] = true
	}
	return result, err
}

// Create menyimpan banyak akun sekaligus dalam 1 transaksi (semua atau tidak sama sekali), lalu
// mengembalikan ID-nya sesuai urutan users. [Native SQL, 2 query]
//
//	INSERT INTO users (...) VALUES (...), (...), ... RETURNING id, username
//	INSERT INTO students (user_id, ...) VALUES (...), ...      -- hanya akun siswa
//
// Kode siswa (STU000001, ...) dibuat otomatis oleh database, urut sesuai urutan siswa di request.
// Username kembar (2 request bersamaan) → ErrUsernameTaken.
func (r *UserRepository) Create(ctx context.Context, users []usermodel.NewUser, createdBy string, now time.Time) ([]string, error) {
	ids := make([]string, len(users))
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		rows := make([]string, 0, len(users))
		args := make([]any, 0, len(users)*17)
		for _, u := range users {
			rows = append(rows, "(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")
			args = append(args, u.Role, u.Username, u.PasswordHash, u.RealName, u.Email, u.Phone, u.ParentPhone,
				u.SchoolOrigin, u.BranchID, u.EducationLevelID, u.GradeID, u.Status, u.JoinDate, u.EndDate, createdBy, now, now)
		}
		var saved []struct{ ID, Username string }
		err := tx.Raw(`
			INSERT INTO users (role, username, password_hash, real_name, email, phone, parent_phone,
			                   school_origin, branch_id, education_level_id, grade_id, status, join_date, end_date,
			                   created_by, created_at, updated_at)
			VALUES `+strings.Join(rows, ", ")+`
			RETURNING id, username`, args...).
			Scan(&saved).Error
		if isUniqueViolation(err, "users_username_unique") {
			return usermodel.ErrUsernameTaken
		}
		if err != nil {
			return err
		}
		idByUsername := make(map[string]string, len(saved))
		for _, s := range saved {
			idByUsername[s.Username] = s.ID
		}
		for i, u := range users {
			ids[i] = idByUsername[u.Username]
		}

		// Data siswa (kode siswa dari sequence, urut sesuai request).
		studentRows := []string{}
		studentArgs := []any{}
		for i, u := range users {
			if u.Role != "student" {
				continue
			}
			studentRows = append(studentRows, "(?, ?, ?)")
			studentArgs = append(studentArgs, ids[i], u.DateOfBirth, u.TargetSchool)
		}
		if len(studentRows) == 0 {
			return nil
		}
		return tx.Exec(`INSERT INTO students (user_id, date_of_birth, target_school) VALUES `+
			strings.Join(studentRows, ", "), studentArgs...).Error
	})
	return ids, err
}

// FindByIDs mengambil banyak user (bentuk baris list), urutan sesuai ids. [Native SQL, primary key]
func (r *UserRepository) FindByIDs(ctx context.Context, ids []string) ([]usermodel.ListItem, error) {
	var rows []userRow
	err := r.db.WithContext(ctx).
		Raw(`SELECT `+userColumns+` FROM users u`+userJoins+` WHERE u.id IN ? AND u.deleted_at IS NULL`, ids).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	byID := make(map[string]usermodel.ListItem, len(rows))
	for _, row := range rows {
		byID[row.ID] = row.toListItem()
	}
	items := make([]usermodel.ListItem, 0, len(ids))
	for _, id := range ids {
		if item, ok := byID[id]; ok {
			items = append(items, item)
		}
	}
	return items, nil
}

// BranchRef adalah cabang (dicocokkan dari nama di file Excel).
type BranchRef struct {
	ID, Name, Status string
}

// FindBranchesByNames mengambil cabang (belum dihapus) yang namanya ada di daftar, tidak peka huruf
// besar/kecil, 1 query (index unik branches_name_unique memakai LOWER(name)). Hasil: map nama kecil → cabang.
func (r *UserRepository) FindBranchesByNames(ctx context.Context, names []string) (map[string]BranchRef, error) {
	result := map[string]BranchRef{}
	if len(names) == 0 {
		return result, nil
	}
	lower := make([]string, 0, len(names))
	for _, n := range names {
		lower = append(lower, strings.ToLower(n))
	}
	var rows []BranchRef
	err := r.db.WithContext(ctx).
		Raw(`SELECT id, name, status FROM branches WHERE LOWER(name) IN ? AND deleted_at IS NULL`, lower).
		Scan(&rows).Error
	for _, b := range rows {
		result[strings.ToLower(b.Name)] = b
	}
	return result, err
}

// FindLevelsAndGrades mengambil semua jenjang & kelas yang belum dihapus (data kecil: ±20 baris),
// untuk mencocokkan nama jenjang/kelas di file Excel. 1 query.
func (r *UserRepository) FindLevelsAndGrades(ctx context.Context) ([]usermodel.MasterDataItem, error) {
	var rows []usermodel.MasterDataItem
	err := r.db.WithContext(ctx).Raw(`
		SELECT id, type, name, parent_id, status FROM master_data_items
		WHERE type IN ('education_level', 'grade') AND deleted_at IS NULL
		ORDER BY sort_order`).Scan(&rows).Error
	return rows, err
}
