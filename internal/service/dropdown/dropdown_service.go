// Package dropdown (service) berisi aturan dropdown: type apa saja yang ada, filter induk
// yang wajib/boleh dipakai, dan nilai (value) yang dikirim FE.
//
// Di package lain, import dengan alias dropdownservice:
//
//	dropdownservice "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/service/dropdown"
package dropdown

import (
	"context"
	"sort"

	dropdowndto "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/dto/dropdown"
	dropdownmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/dropdown"
	dropdownrepo "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/repository/dropdown"
)

// typeRule adalah aturan 1 type dropdown.
type typeRule struct {
	parentParam    string // nama query string induk, contoh "subject_id"; "" = type tanpa induk
	parentRequired bool   // true = induk wajib diisi (data anaknya bisa ribuan)
	gradeFilter    bool   // true = boleh difilter grade_id
	valueIsCode    bool   // true = value = code (disimpan sebagai teks), false = value = id
}

// rules adalah semua type dropdown. Tambah baris di sini kalau ada type master data baru.
var rules = map[string]typeRule{
	dropdownmodel.TypeEducationLevel:   {},
	dropdownmodel.TypeGrade:            {parentParam: "education_level_id"},
	dropdownmodel.TypeSubject:          {},
	dropdownmodel.TypeSubjectField:     {parentParam: "subject_id", parentRequired: true, gradeFilter: true},
	dropdownmodel.TypeChapter:          {parentParam: "subject_field_id", parentRequired: true},
	dropdownmodel.TypeSubchapter:       {parentParam: "chapter_id", parentRequired: true},
	dropdownmodel.TypeTopic:            {parentParam: "subchapter_id", parentRequired: true},
	dropdownmodel.TypeQuestionType:     {valueIsCode: true},
	dropdownmodel.TypeQuizType:         {valueIsCode: true},
	dropdownmodel.TypeDifficulty:       {valueIsCode: true},
	dropdownmodel.TypeSubscriptionPlan: {}, // dari tabel subscription_plans (lihat List)
}

// Types mengembalikan semua type dropdown (urut abjad), untuk pesan error.
func Types() []string {
	types := make([]string, 0, len(rules))
	for t := range rules {
		types = append(types, t)
	}
	sort.Strings(types)
	return types
}

// DropdownService berisi logika dropdown.
type DropdownService struct {
	repo *dropdownrepo.DropdownRepository
}

// NewDropdownService membuat DropdownService. Dipanggil sekali di main.go.
func NewDropdownService(repo *dropdownrepo.DropdownRepository) *DropdownService {
	return &DropdownService{repo: repo}
}

// List mengambil pilihan dropdown 1 type. Format UUID query string sudah dicek binding di handler.
// Error: ErrUnknownType (type tidak dikenal), ValidationError (induk wajib belum diisi).
func (s *DropdownService) List(ctx context.Context, itemType string, q dropdowndto.DropdownQuery) ([]dropdownmodel.Option, error) {
	rule, ok := rules[itemType]
	if !ok {
		return nil, dropdownmodel.ErrUnknownType
	}

	// Paket akses bukan master data: dibaca dari subscription_plans (paket aktif).
	if itemType == dropdownmodel.TypeSubscriptionPlan {
		options, err := s.repo.FindPlanOptions(ctx)
		for i := range options {
			options[i].Value = options[i].ID
		}
		return options, err
	}

	filter := dropdownmodel.Filter{Type: itemType, Root: rule.parentParam == ""}
	if rule.parentParam != "" {
		filter.ParentID = parentValue(q, rule.parentParam)
		if rule.parentRequired && filter.ParentID == "" {
			return nil, &dropdownmodel.ValidationError{Fields: []dropdownmodel.FieldError{
				{Field: rule.parentParam, Message: "wajib diisi"},
			}}
		}
	}
	if rule.gradeFilter {
		filter.GradeID = q.GradeID
	}

	options, err := s.repo.FindOptions(ctx, filter)
	if err != nil {
		return nil, err
	}
	for i := range options {
		options[i].Value = options[i].ID
		if rule.valueIsCode && options[i].Code != nil {
			options[i].Value = *options[i].Code
		}
	}
	return options, nil
}

// parentValue mengambil nilai query string induk berdasarkan namanya.
func parentValue(q dropdowndto.DropdownQuery, param string) string {
	switch param {
	case "education_level_id":
		return q.EducationLevelID
	case "subject_id":
		return q.SubjectID
	case "subject_field_id":
		return q.SubjectFieldID
	case "chapter_id":
		return q.ChapterID
	case "subchapter_id":
		return q.SubchapterID
	}
	return ""
}
