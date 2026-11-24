// Test gambar dari link luar (image_url) di soal & opsi, serta input_type "text" lama ("string") ditolak.
package router

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
)

func TestCreateQuestion_GambarLinkLuar(t *testing.T) {
	r, db := setupRouter(t)
	p := seedPlacement(t, db)
	accessToken := adminToken(t, db)

	// Soal & opsi A memakai link gambar; opsi A tidak punya teks (cukup gambar).
	answer := `{"options":[{"key":"A","content":"","image_url":"https://cdn.example.com/a.png"},{"key":"B","content":"12"}],"correct_key":"B"}`
	id := createQuestionID(t, r, accessToken, questionBody(p, "single_choice", answer, map[string]string{
		"image_url":     `"https://cdn.example.com/soal.png"`,
		"display_order": `["question","image"]`,
	}))

	res := sendRequest(r, http.MethodGet, "/api/v1/questions/"+id, "", accessToken)
	body := res.Body.String()
	if res.Code != http.StatusOK ||
		!strings.Contains(body, `"image_url":"https://cdn.example.com/soal.png"`) ||
		!strings.Contains(body, `{"key":"A","content":"","image_id":null,"image_url":"https://cdn.example.com/a.png"}`) ||
		!strings.Contains(body, `{"key":"B","content":"12","image_id":null}`) { // opsi tanpa link: image_url tidak ditampilkan
		t.Fatalf("link gambar tidak tersimpan dengan benar. status %d, body: %s", res.Code, body)
	}
}

func TestCreateQuestion_GambarLinkLuarValidasi(t *testing.T) {
	r, db := setupRouter(t)
	p := seedPlacement(t, db)
	accessToken := adminToken(t, db)

	tests := []struct {
		name       string
		answer     string
		extra      map[string]string
		wantFields []string
	}{
		{
			name:       "bukan link http/https",
			answer:     `{"options":[{"key":"A","content":"1","image_url":"ftp://x.com/a.png"},{"key":"B","content":"2"}],"correct_key":"A"}`,
			extra:      map[string]string{"image_url": `"cdn.example.com/soal.png"`, "display_order": `["question","image"]`},
			wantFields: []string{"image_url", "answer.options[0].image_url"},
		},
		{
			name:   "image_id dan image_url sekaligus",
			answer: fmt.Sprintf(`{"options":[{"key":"A","content":"1","image_id":%q,"image_url":"https://x.com/a.png"},{"key":"B","content":"2"}],"correct_key":"A"}`, p.TopicID),
			extra: map[string]string{
				"image_id": fmt.Sprintf("%q", p.TopicID), "image_url": `"https://x.com/soal.png"`, "display_order": `["question","image"]`,
			},
			wantFields: []string{"image_url", "answer.options[0].image_url"},
		},
		{
			name:       "free_text input_type lama \"string\" ditolak",
			answer:     `{"input_type":"string","accepted_answers":["prima"]}`,
			wantFields: []string{"answer.input_type"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			questionType := "single_choice"
			if strings.Contains(tt.answer, "input_type") {
				questionType = "free_text"
			}
			res := sendRequest(r, http.MethodPost, "/api/v1/questions", questionBody(p, questionType, tt.answer, tt.extra), accessToken)
			if res.Code != http.StatusUnprocessableEntity {
				t.Fatalf("status = %d, seharusnya 422. body: %s", res.Code, res.Body)
			}
			got := errorFields(t, res.Body.String())
			for _, field := range tt.wantFields {
				if _, ok := got[firstQuestionField(field)]; !ok {
					t.Errorf("error untuk field %q tidak ada. errors: %v", field, got)
				}
			}
		})
	}
}
