// Unit test untuk SubjectService. Jalankan dengan: go test ./internal/service/
//
// Catatan Go: file test selalu berakhiran _test.go, dan fungsi test
// selalu diawali "Test" dengan parameter (t *testing.T).
package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/dto"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository"
)

// newTestService membuat service baru dengan repository in-memory yang kosong.
func newTestService() *SubjectService {
	return NewSubjectService(repository.NewSubjectMemoryRepository())
}

func TestCreate_KodeDijadikanHurufBesar(t *testing.T) {
	svc := newTestService()

	subject, err := svc.Create(context.Background(), dto.CreateSubjectRequest{
		Code: "  mtk ",
		Name: " Matematika  ",
	})

	if err != nil {
		t.Fatalf("tidak boleh error, tapi dapat: %v", err)
	}
	if subject.Code != "MTK" {
		t.Errorf("Code = %q, seharusnya %q", subject.Code, "MTK")
	}
	if subject.Name != "Matematika" {
		t.Errorf("Name = %q, seharusnya %q", subject.Name, "Matematika")
	}
	if subject.ID == "" {
		t.Error("ID seharusnya terisi otomatis")
	}
}

func TestCreate_KodeKembarDitolak(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	_, _ = svc.Create(ctx, dto.CreateSubjectRequest{Code: "MTK", Name: "Matematika"})
	_, err := svc.Create(ctx, dto.CreateSubjectRequest{Code: "mtk", Name: "Matematika Lanjut"})

	if !errors.Is(err, model.ErrSubjectCodeTaken) {
		t.Errorf("seharusnya ErrSubjectCodeTaken, tapi dapat: %v", err)
	}
}

func TestGetByID_TidakDitemukan(t *testing.T) {
	svc := newTestService()

	_, err := svc.GetByID(context.Background(), "id-yang-tidak-ada")

	if !errors.Is(err, model.ErrSubjectNotFound) {
		t.Errorf("seharusnya ErrSubjectNotFound, tapi dapat: %v", err)
	}
}

// newServiceWithABC membuat service yang sudah berisi 3 data: A, B, C.
func newServiceWithABC() *SubjectService {
	svc := newTestService()
	for _, code := range []string{"A", "B", "C"} {
		_, _ = svc.Create(context.Background(), dto.CreateSubjectRequest{Code: code, Name: "Mapel " + code})
	}
	return svc
}

// codesOf mengambil kode dari setiap subject, misalnya [A B].
func codesOf(subjects []model.Subject) []string {
	codes := []string{}
	for _, subject := range subjects {
		codes = append(codes, subject.Code)
	}
	return codes
}

func TestList_Halaman1(t *testing.T) {
	subjects, total, _ := newServiceWithABC().List(context.Background(), 1, 2)

	if total != 3 {
		t.Errorf("total = %d, seharusnya 3", total)
	}
	if got := strings.Join(codesOf(subjects), ","); got != "A,B" {
		t.Errorf("halaman 1 = %s, seharusnya A,B", got)
	}
}

func TestList_Halaman2(t *testing.T) {
	subjects, _, _ := newServiceWithABC().List(context.Background(), 2, 2)

	if got := strings.Join(codesOf(subjects), ","); got != "C" {
		t.Errorf("halaman 2 = %s, seharusnya C", got)
	}
}

func TestList_HalamanMelebihiData(t *testing.T) {
	subjects, _, _ := newServiceWithABC().List(context.Background(), 3, 2)

	if len(subjects) != 0 {
		t.Errorf("halaman 3 seharusnya kosong, tapi dapat %v", codesOf(subjects))
	}
}
