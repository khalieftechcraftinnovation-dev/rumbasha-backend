package question

import (
	"context"
	"encoding/json"
	"fmt"

	questionmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/question"
)

// FindForAttempt mengambil isi soal untuk quiz yang dibuka siswa, urut sesuai ids:
// teks, rumus, gambar, tabel, urutan tampil, level, dan answer (opsi + KUNCI JAWABAN).
// Pembahasan hanya diisi kalau withExplanation = true (review setelah submit); saat quiz dibuka
// pembahasan TIDAK dikirim. Soal yang sudah dihapus tetap diambil, karena
// sudah menjadi bagian quiz. [Native SQL, 2 query]
//
//  1. Soal + link gambar soal (Image Library di-JOIN ke media_assets, atau link luar).
//  2. Link gambar opsi jawaban dari Image Library (semua opsi semua soal sekaligus).
func (r *QuestionRepository) FindForAttempt(ctx context.Context, ids []string, withExplanation bool) ([]questionmodel.AttemptQuestion, error) {
	if len(ids) == 0 {
		return []questionmodel.AttemptQuestion{}, nil
	}
	var rows []struct {
		ID, Code, Type, QuestionText string
		Formula, ImageURL, TableData *string
		DisplayOrder, Difficulty     string
		Answer, Explanation          string
	}
	err := r.db.WithContext(ctx).
		Raw(`
			SELECT q.id, q.code, q.type, q.question_text, q.formula,
			       COALESCE(m.url, q.image_url) AS image_url,
			       q.table_data::text AS table_data, q.display_order::text AS display_order,
			       q.difficulty, q.answer::text AS answer, q.explanation
			FROM questions q
			LEFT JOIN media_assets m ON m.id = q.image_id
			WHERE q.id IN ?`, ids).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	// Kumpulkan image_id opsi jawaban semua soal → 1 query link gambar.
	imageIDs := []string{}
	for _, row := range rows {
		var a struct {
			Options []questionmodel.ChoiceOption `json:"options"`
		}
		if err := json.Unmarshal([]byte(row.Answer), &a); err != nil {
			return nil, fmt.Errorf("soal %s: answer rusak: %w", row.Code, err)
		}
		for _, o := range a.Options {
			if o.ImageID != nil {
				imageIDs = append(imageIDs, *o.ImageID)
			}
		}
	}
	imageURLs, err := r.findImageURLs(ctx, imageIDs)
	if err != nil {
		return nil, err
	}

	byID := make(map[string]questionmodel.AttemptQuestion, len(rows))
	for _, row := range rows {
		q := questionmodel.AttemptQuestion{
			ID: row.ID, Code: row.Code, Type: row.Type, QuestionText: row.QuestionText,
			Formula: row.Formula, ImageURL: row.ImageURL, Difficulty: row.Difficulty,
		}
		if withExplanation {
			q.Explanation = row.Explanation
		}
		if row.TableData != nil {
			q.Table = &questionmodel.Table{}
			if err := json.Unmarshal([]byte(*row.TableData), q.Table); err != nil {
				return nil, fmt.Errorf("soal %s: table_data rusak: %w", row.Code, err)
			}
		}
		if err := json.Unmarshal([]byte(row.DisplayOrder), &q.DisplayOrder); err != nil {
			return nil, fmt.Errorf("soal %s: display_order rusak: %w", row.Code, err)
		}
		if q.Answer, err = answerWithImageURLs(row.Type, []byte(row.Answer), imageURLs); err != nil {
			return nil, fmt.Errorf("soal %s: answer rusak: %w", row.Code, err)
		}
		byID[row.ID] = q
	}

	result := make([]questionmodel.AttemptQuestion, 0, len(ids))
	for _, id := range ids {
		q, ok := byID[id]
		if !ok {
			return nil, fmt.Errorf("soal %s tidak ditemukan", id)
		}
		result = append(result, q)
	}
	return result, nil
}

// answerWithImageURLs merapikan answer (bentuk sama dengan create soal) dan mengisi image_url
// setiap opsi yang memakai gambar Image Library (image_id), supaya FE bisa langsung menampilkan.
func answerWithImageURLs(questionType string, raw []byte, imageURLs map[string]string) (json.RawMessage, error) {
	fill := func(options []questionmodel.ChoiceOption) {
		for i, o := range options {
			if o.ImageID != nil {
				if url, ok := imageURLs[*o.ImageID]; ok {
					options[i].ImageURL = &url
				}
			}
		}
	}

	var target any
	switch questionType {
	case questionmodel.TypeSingleChoice:
		a := &questionmodel.SingleChoiceAnswer{}
		if err := json.Unmarshal(raw, a); err != nil {
			return nil, err
		}
		fill(a.Options)
		target = a
	case questionmodel.TypeMultipleChoice:
		a := &questionmodel.MultipleChoiceAnswer{}
		if err := json.Unmarshal(raw, a); err != nil {
			return nil, err
		}
		fill(a.Options)
		target = a
	case questionmodel.TypeTrueFalse:
		a := &questionmodel.TrueFalseAnswer{}
		if err := json.Unmarshal(raw, a); err != nil {
			return nil, err
		}
		fill(a.Options)
		target = a
	default: // isian: tidak ada opsi
		return questionmodel.CanonicalAnswer(questionType, raw)
	}
	return json.Marshal(target)
}

// findImageURLs mengambil link gambar Image Library (map id → url). [Native SQL, primary key]
func (r *QuestionRepository) findImageURLs(ctx context.Context, ids []string) (map[string]string, error) {
	result := map[string]string{}
	if len(ids) == 0 {
		return result, nil
	}
	var rows []struct{ ID, URL string }
	if err := r.db.WithContext(ctx).Raw(`SELECT id, url FROM media_assets WHERE id IN ?`, ids).Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		result[row.ID] = row.URL
	}
	return result, nil
}
