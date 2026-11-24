package question

// File ini berisi pengecekan field "answer" per tipe soal.
// Setiap tipe punya fungsi sendiri (validateSingleChoice, dst.) supaya mudah dicari & di-debug.
//
// Jawaban yang lolos disimpan dalam bentuk RAPI (canonical): field yang tidak dikenal
// ditolak, nilai default diisi, dan spasi di ujung dihapus.

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"

	questionmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/question"
)

// Batas jumlah opsi per tipe soal.
const (
	minChoiceOptions    = 2  // pilihan ganda minimal A-B
	maxChoiceOptions    = 5  // pilihan ganda maksimal A-E
	minTrueFalseOptions = 1  // benar/salah minimal 1 pernyataan
	maxTrueFalseOptions = 10 // benar/salah maksimal 10 pernyataan
	maxOptionContent    = 1000
	maxAcceptedAnswers  = 20
	maxAcceptedAnswer   = 200
)

// validateAnswer membaca "answer" sesuai type, mengecek isinya, lalu mengembalikan JSON rapinya.
// Kesalahan isi dicatat di errs. Error yang dikembalikan hanya untuk kegagalan tak terduga.
// imageFields diisi dengan image_id di opsi, untuk dicek keberadaannya di database.
func validateAnswer(questionType string, raw json.RawMessage, errs *fieldErrors, imageFields map[string]string) (json.RawMessage, error) {
	var answer any

	switch questionType {
	case questionmodel.TypeSingleChoice:
		var a questionmodel.SingleChoiceAnswer
		if !decodeAnswer(raw, &a, questionType, errs) {
			return nil, nil
		}
		validateSingleChoice(&a, errs, imageFields)
		answer = a

	case questionmodel.TypeMultipleChoice:
		var a questionmodel.MultipleChoiceAnswer
		if !decodeAnswer(raw, &a, questionType, errs) {
			return nil, nil
		}
		validateMultipleChoice(&a, errs, imageFields)
		answer = a

	case questionmodel.TypeTrueFalse:
		var a questionmodel.TrueFalseAnswer
		if !decodeAnswer(raw, &a, questionType, errs) {
			return nil, nil
		}
		validateTrueFalse(&a, errs, imageFields)
		answer = a

	case questionmodel.TypeFreeText:
		var a questionmodel.FreeTextAnswer
		if !decodeAnswer(raw, &a, questionType, errs) {
			return nil, nil
		}
		validateFreeText(&a, errs)
		answer = a

	default:
		// Tidak mungkin terjadi: type sudah dicek di binding (oneof).
		return nil, fmt.Errorf("type soal tidak dikenal: %q", questionType)
	}

	result, err := json.Marshal(answer)
	if err != nil {
		return nil, fmt.Errorf("gagal mengubah answer ke JSON: %w", err)
	}
	return result, nil
}

// decodeAnswer membaca JSON answer ke struct target.
// Field yang tidak dikenal DITOLAK, supaya salah ketik (contoh "correct_key" di soal
// multiple_choice yang seharusnya "correct_keys") langsung ketahuan.
func decodeAnswer(raw json.RawMessage, target any, questionType string, errs *fieldErrors) bool {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || trimmed[0] != '{' {
		errs.add("answer", "harus berupa object JSON")
		return false
	}

	decoder := json.NewDecoder(bytes.NewReader(trimmed))
	decoder.DisallowUnknownFields()
	err := decoder.Decode(target)
	if err == nil {
		return true
	}

	var typeErr *json.UnmarshalTypeError
	switch {
	case errors.As(err, &typeErr):
		errs.add("answer."+typeErr.Field, "tipe data tidak sesuai (seharusnya "+typeErr.Type.String()+")")
	case strings.HasPrefix(err.Error(), "json: unknown field "):
		field := strings.TrimPrefix(err.Error(), "json: unknown field ")
		errs.add("answer", fmt.Sprintf("field %s tidak dikenal untuk tipe soal %s", field, questionType))
	default:
		errs.add("answer", "format JSON tidak valid")
	}
	return false
}

// ----------------------------------------------------------------------------
// single_choice
// ----------------------------------------------------------------------------

func validateSingleChoice(a *questionmodel.SingleChoiceAnswer, errs *fieldErrors, imageFields map[string]string) {
	keys := validateOptions(a.Options, minChoiceOptions, maxChoiceOptions, errs, imageFields)

	a.CorrectKey = strings.TrimSpace(a.CorrectKey)
	switch {
	case a.CorrectKey == "":
		errs.add("answer.correct_key", "wajib diisi")
	case !keys[a.CorrectKey]:
		errs.add("answer.correct_key", "harus salah satu key di options")
	}
}

// ----------------------------------------------------------------------------
// multiple_choice
// ----------------------------------------------------------------------------

func validateMultipleChoice(a *questionmodel.MultipleChoiceAnswer, errs *fieldErrors, imageFields map[string]string) {
	keys := validateOptions(a.Options, minChoiceOptions, maxChoiceOptions, errs, imageFields)

	if len(a.CorrectKeys) == 0 {
		errs.add("answer.correct_keys", "wajib diisi minimal 1")
	}
	seen := map[string]bool{}
	for i, key := range a.CorrectKeys {
		key = strings.TrimSpace(key)
		a.CorrectKeys[i] = key
		field := fmt.Sprintf("answer.correct_keys[%d]", i)
		switch {
		case !keys[key]:
			errs.add(field, "harus salah satu key di options")
		case seen[key]:
			errs.add(field, "tidak boleh ada yang sama")
		}
		seen[key] = true
	}

	if a.Scoring == "" {
		a.Scoring = questionmodel.ScoringAllOrNothing
	}
	if a.Scoring != questionmodel.ScoringAllOrNothing && a.Scoring != questionmodel.ScoringPartial {
		errs.add("answer.scoring", "harus salah satu dari: all_or_nothing, partial")
	}
}

// ----------------------------------------------------------------------------
// true_false
// ----------------------------------------------------------------------------

func validateTrueFalse(a *questionmodel.TrueFalseAnswer, errs *fieldErrors, imageFields map[string]string) {
	keys := validateOptions(a.Options, minTrueFalseOptions, maxTrueFalseOptions, errs, imageFields)

	// Setiap pernyataan wajib punya tepat 1 kunci benar/salah.
	byKey := map[string]questionmodel.TrueFalseKey{}
	for i, ck := range a.CorrectKeys {
		ck.Key = strings.TrimSpace(ck.Key)
		field := fmt.Sprintf("answer.correct_keys[%d]", i)
		switch {
		case !keys[ck.Key]:
			errs.add(field+".key", "harus salah satu key di options")
		case byKey[ck.Key].Key != "":
			errs.add(field+".key", "kunci untuk "+ck.Key+" sudah ada")
		case ck.Correct == nil:
			errs.add(field+".correct", "wajib diisi (true/false)")
		default:
			byKey[ck.Key] = ck
		}
	}

	// Susun ulang mengikuti urutan options, sekaligus cek tidak ada yang terlewat.
	ordered := make([]questionmodel.TrueFalseKey, 0, len(a.Options))
	for _, opt := range a.Options {
		ck, ok := byKey[opt.Key]
		if !ok {
			if keys[opt.Key] {
				errs.add("answer.correct_keys", "kunci untuk pernyataan "+opt.Key+" belum diisi")
			}
			continue
		}
		ordered = append(ordered, ck)
	}
	a.CorrectKeys = ordered
}

// ----------------------------------------------------------------------------
// free_text
// ----------------------------------------------------------------------------

func validateFreeText(a *questionmodel.FreeTextAnswer, errs *fieldErrors) {
	switch a.InputType {
	case "":
		errs.add("answer.input_type", "wajib diisi")
	case questionmodel.InputTypeNumber, questionmodel.InputTypeString:
	default:
		errs.add("answer.input_type", "harus salah satu dari: number, string")
	}

	switch {
	case len(a.AcceptedAnswers) == 0:
		errs.add("answer.accepted_answers", "wajib diisi minimal 1")
	case len(a.AcceptedAnswers) > maxAcceptedAnswers:
		errs.add("answer.accepted_answers", fmt.Sprintf("maksimal %d item", maxAcceptedAnswers))
	}

	caseSensitive := a.CaseSensitive != nil && *a.CaseSensitive
	seen := map[string]bool{}
	for i, ans := range a.AcceptedAnswers {
		ans = strings.TrimSpace(ans)
		a.AcceptedAnswers[i] = ans
		field := fmt.Sprintf("answer.accepted_answers[%d]", i)

		compareKey := ans
		if !caseSensitive {
			compareKey = strings.ToLower(ans)
		}

		switch {
		case ans == "":
			errs.add(field, "wajib diisi")
		case len([]rune(ans)) > maxAcceptedAnswer:
			errs.add(field, fmt.Sprintf("maksimal %d karakter", maxAcceptedAnswer))
		case a.InputType == questionmodel.InputTypeNumber && !isNumberAnswer(ans):
			errs.add(field, "harus berupa angka, desimal (0.75 / 0,75), atau pecahan (3/4)")
		case seen[compareKey]:
			errs.add(field, "tidak boleh ada yang sama")
		}
		seen[compareKey] = true
	}

	// Simpan hanya pengaturan yang berlaku untuk input_type-nya.
	if a.InputType == questionmodel.InputTypeNumber {
		if a.Tolerance == nil {
			zero := 0.0
			a.Tolerance = &zero
		}
		if *a.Tolerance < 0 {
			errs.add("answer.tolerance", "tidak boleh negatif")
		}
		a.CaseSensitive = nil // tidak berlaku untuk angka
	}
	if a.InputType == questionmodel.InputTypeString {
		if a.Tolerance != nil && *a.Tolerance != 0 {
			errs.add("answer.tolerance", "hanya berlaku untuk input_type number")
		}
		a.Tolerance = nil
		if a.CaseSensitive == nil {
			no := false
			a.CaseSensitive = &no
		}
	}
}

// isNumberAnswer: "12", "-3", "0.75", "0,75", "3/4" dianggap angka. Penyebut pecahan tidak boleh 0.
func isNumberAnswer(s string) bool {
	s = strings.ReplaceAll(s, ",", ".")
	if top, bottom, isFraction := strings.Cut(s, "/"); isFraction {
		_, err1 := strconv.ParseFloat(strings.TrimSpace(top), 64)
		d, err2 := strconv.ParseFloat(strings.TrimSpace(bottom), 64)
		return err1 == nil && err2 == nil && d != 0
	}
	_, err := strconv.ParseFloat(s, 64)
	return err == nil
}

// ----------------------------------------------------------------------------
// Opsi (dipakai single_choice, multiple_choice, true_false)
// ----------------------------------------------------------------------------

// validateOptions mengecek daftar opsi dan mengembalikan set key yang valid.
//   - Jumlah opsi antara minCount dan maxCount.
//   - Key urut: A, B, C, ... (opsi ke-1 = A, ke-2 = B, dst.).
//   - Setiap opsi wajib punya content atau image_id.
func validateOptions(options []questionmodel.ChoiceOption, minCount, maxCount int, errs *fieldErrors, imageFields map[string]string) map[string]bool {
	keys := map[string]bool{}

	if len(options) < minCount || len(options) > maxCount {
		errs.add("answer.options", fmt.Sprintf("jumlah opsi harus %d sampai %d", minCount, maxCount))
		if len(options) > maxCount {
			return keys
		}
	}

	for i := range options {
		opt := &options[i]
		field := fmt.Sprintf("answer.options[%d]", i)

		opt.Key = strings.TrimSpace(opt.Key)
		expectedKey := string(rune('A' + i))
		if opt.Key != expectedKey {
			errs.add(field+".key", fmt.Sprintf("harus %q (urut A, B, C, ...)", expectedKey))
		} else {
			keys[opt.Key] = true
		}

		opt.Content = strings.TrimSpace(opt.Content)
		if opt.Content == "" && opt.ImageID == nil {
			errs.add(field+".content", "wajib diisi jika image_id kosong")
		}
		if len([]rune(opt.Content)) > maxOptionContent {
			errs.add(field+".content", fmt.Sprintf("maksimal %d karakter", maxOptionContent))
		}

		if opt.ImageID != nil {
			if uuid.Validate(*opt.ImageID) != nil {
				errs.add(field+".image_id", "format UUID tidak valid")
			} else {
				imageFields[field+".image_id"] = *opt.ImageID
			}
		}
	}
	return keys
}
