package masterdata

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	masterdatadto "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/dto/master/masterdata"
	masterdatamodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/master/masterdata"
)

var branchRule = masterdatamodel.TypeRule{Label: "Cabang"}

// ListBranches mengambil semua cabang (filter nama & status) + jumlah akun.
func (s *MasterDataService) ListBranches(ctx context.Context, q masterdatadto.BranchListQuery) ([]masterdatamodel.Branch, error) {
	return s.repo.ListBranches(ctx, masterdatamodel.BranchFilter{Search: strings.TrimSpace(q.Search), Status: q.Status})
}

// GetBranch mengambil 1 cabang. Bukan UUID / tidak ada → ErrBranchNotFound.
func (s *MasterDataService) GetBranch(ctx context.Context, id string) (masterdatamodel.Branch, error) {
	if uuid.Validate(id) != nil {
		return masterdatamodel.Branch{}, masterdatamodel.ErrBranchNotFound
	}
	return s.repo.FindBranch(ctx, id)
}

// CreateBranch menambah cabang. Nama unik (tanpa peduli huruf besar/kecil).
func (s *MasterDataService) CreateBranch(ctx context.Context, req masterdatadto.BranchRequest, adminID string) (masterdatamodel.Branch, error) {
	name, status, err := cleanBranch(req)
	if err != nil {
		return masterdatamodel.Branch{}, err
	}
	id, err := s.repo.CreateBranch(ctx, name, status, adminID)
	if err != nil {
		return masterdatamodel.Branch{}, duplicateError(err, branchRule)
	}
	return s.repo.FindBranch(ctx, id)
}

// UpdateBranch mengganti nama & status cabang (ganti penuh). Cabang nonaktif tidak bisa dipilih
// untuk akun baru / diubah, akun yang sudah ada tetap.
func (s *MasterDataService) UpdateBranch(ctx context.Context, id string, req masterdatadto.BranchRequest, adminID string) (masterdatamodel.Branch, error) {
	if uuid.Validate(id) != nil {
		return masterdatamodel.Branch{}, masterdatamodel.ErrBranchNotFound
	}
	name, status, err := cleanBranch(req)
	if err != nil {
		return masterdatamodel.Branch{}, err
	}
	if err := s.repo.UpdateBranch(ctx, id, name, status, adminID); err != nil {
		return masterdatamodel.Branch{}, duplicateError(err, branchRule)
	}
	return s.repo.FindBranch(ctx, id)
}

// DeleteBranches menghapus 1-100 cabang (soft delete), semua atau tidak sama sekali. Cabang yang
// masih dipakai akun (belum dihapus) tidak bisa dihapus.
func (s *MasterDataService) DeleteBranches(ctx context.Context, ids []string, adminID string) ([]string, error) {
	missing, used, err := s.repo.SoftDeleteBranches(ctx, ids, adminID)
	if err != nil {
		return nil, err
	}
	if len(missing) == 0 && len(used) == 0 {
		return ids, nil
	}
	isMissing := map[string]bool{}
	for _, id := range missing {
		isMissing[id] = true
	}
	errs := fieldErrors{}
	for i, id := range ids {
		field := fmt.Sprintf("ids[%d]", i)
		switch {
		case isMissing[id]:
			errs.add(field, "cabang tidak ditemukan atau sudah dihapus")
		case used[id] > 0:
			errs.add(field, fmt.Sprintf("masih dipakai %d akun; pindahkan akun atau nonaktifkan cabang", used[id]))
		}
	}
	return nil, &masterdatamodel.ValidationError{Fields: errs}
}

func cleanBranch(req masterdatadto.BranchRequest) (string, string, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return "", "", &masterdatamodel.ValidationError{Fields: []masterdatamodel.FieldError{{Field: "name", Message: "wajib diisi"}}}
	}
	status := req.Status
	if status == "" {
		status = masterdatamodel.StatusActive
	}
	return name, status, nil
}
