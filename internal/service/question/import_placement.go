package question

// File ini mengubah NAMA placement dari Excel (jenjang, kelas, mapel, ...) menjadi ID.
//
// Semua nama di file diambil dengan 1 query (FindMasterDataByNames), lalu dicocokkan per
// baris di sini mengikuti hierarki, contoh: kelas dicari di dalam jenjang yang dipilih,
// bidang materi dicari di dalam mapel + kelas yang dipilih, dan seterusnya.

import (
	"fmt"
	"strings"

	questiondto "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/dto/question"
	questionmodel "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/model/question"
)

// placementNames adalah nama placement 1 baris Excel (apa adanya dari sel).
type placementNames struct {
	EducationLevel, Grade, Subject, SubjectField, Chapter, Subchapter, Topic string
}

// resolvedPlacement adalah hasil pencocokan: ID untuk request create + nama resmi dari database.
type resolvedPlacement struct {
	Request questiondto.PlacementRequest
	Names   placementNames
}

// placementResolver menyimpan item master data dalam map supaya pencarian per baris cepat.
// Key: type|induk|nama-huruf-kecil (lihat fungsi key di bawah).
type placementResolver struct {
	items map[string]questionmodel.MasterDataItem
}

func newPlacementResolver(items []questionmodel.MasterDataItem) *placementResolver {
	r := &placementResolver{items: make(map[string]questionmodel.MasterDataItem, len(items))}
	for _, item := range items {
		r.items[r.keyOf(item)] = item
	}
	return r
}

// keyOf membuat key pencarian untuk 1 item, sesuai cara item itu dicari:
//   - jenjang & mapel: cukup nama (tidak punya induk)
//   - bidang materi: mapel + kelas + nama (nama sama boleh ada di kelas berbeda)
//   - kelas, bab, sub bab, topik: induk + nama
func (r *placementResolver) keyOf(item questionmodel.MasterDataItem) string {
	name := strings.ToLower(item.Name)
	switch item.Type {
	case "education_level", "subject":
		return key(item.Type, "", name)
	case "subject_field":
		return key(item.Type, deref(item.SubjectID)+"/"+deref(item.GradeID), name)
	default:
		return key(item.Type, deref(item.ParentID), name)
	}
}

func key(itemType, parent, lowerName string) string {
	return itemType + "|" + parent + "|" + lowerName
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// find mencari item; ok = false kalau tidak ada.
func (r *placementResolver) find(itemType, parent, name string) (questionmodel.MasterDataItem, bool) {
	item, ok := r.items[key(itemType, parent, strings.ToLower(strings.TrimSpace(name)))]
	return item, ok
}

// resolve mencocokkan nama placement 1 baris. Kesalahan dicatat lewat addErr(header, pesan).
// Kalau induknya tidak ketemu, anak-anaknya tidak dicek (supaya error tidak menumpuk).
// ok = true kalau semua placement berhasil dicocokkan.
func (r *placementResolver) resolve(n placementNames, addErr func(header, message string)) (resolvedPlacement, bool) {
	var res resolvedPlacement
	ok := true

	// lookup mencari 1 tingkat, mengecek wajib/ada/aktif, lalu mengembalikan item-nya.
	lookup := func(header, label, value, itemType, parent, notFoundMsg string, required bool) (questionmodel.MasterDataItem, bool) {
		value = strings.TrimSpace(value)
		if value == "" {
			if required {
				addErr(header, "wajib diisi")
				ok = false
			}
			return questionmodel.MasterDataItem{}, false
		}
		item, found := r.find(itemType, parent, value)
		switch {
		case !found:
			addErr(header, notFoundMsg)
			ok = false
			return item, false
		case item.Status != "active":
			addErr(header, fmt.Sprintf("%s %q tidak aktif", label, item.Name))
			ok = false
			return item, false
		}
		return item, true
	}

	level, levelOK := lookup("education_level_name", "jenjang", n.EducationLevel, "education_level", "",
		fmt.Sprintf("jenjang %q tidak ditemukan", n.EducationLevel), true)

	var grade questionmodel.MasterDataItem
	gradeOK := false
	if levelOK {
		grade, gradeOK = lookup("grade_name", "kelas", n.Grade, "grade", level.ID,
			fmt.Sprintf("kelas %q tidak ada di jenjang %s", n.Grade, level.Name), true)
	}

	subject, subjectOK := lookup("subject_name", "mapel", n.Subject, "subject", "",
		fmt.Sprintf("mapel %q tidak ditemukan", n.Subject), true)

	var field questionmodel.MasterDataItem
	fieldOK := false
	if subjectOK && gradeOK {
		field, fieldOK = lookup("subject_field_name", "bidang materi", n.SubjectField, "subject_field", subject.ID+"/"+grade.ID,
			fmt.Sprintf("bidang materi %q tidak ada di %s %s", n.SubjectField, subject.Name, grade.Name), true)
	}

	var chapter questionmodel.MasterDataItem
	chapterOK := false
	if fieldOK {
		chapter, chapterOK = lookup("chapter_name", "bab", n.Chapter, "chapter", field.ID,
			fmt.Sprintf("bab %q tidak ada di bidang materi %s (%s)", n.Chapter, field.Name, grade.Name), true)
	}

	// Sub bab & topik opsional, tapi topik butuh sub bab.
	if strings.TrimSpace(n.Topic) != "" && strings.TrimSpace(n.Subchapter) == "" {
		addErr("subchapter_name", "wajib diisi jika topic_name diisi")
		ok = false
	}
	var sub questionmodel.MasterDataItem
	subOK := false
	if chapterOK {
		sub, subOK = lookup("subchapter_name", "sub bab", n.Subchapter, "subchapter", chapter.ID,
			fmt.Sprintf("sub bab %q tidak ada di bab %s", n.Subchapter, chapter.Name), false)
	}
	var topic questionmodel.MasterDataItem
	topicOK := false
	if subOK {
		topic, topicOK = lookup("topic_name", "topik", n.Topic, "topic", sub.ID,
			fmt.Sprintf("topik %q tidak ada di sub bab %s", n.Topic, sub.Name), false)
	}

	if !ok || !chapterOK {
		return res, false
	}

	res.Request = questiondto.PlacementRequest{
		EducationLevelID: level.ID,
		GradeID:          grade.ID,
		SubjectID:        subject.ID,
		SubjectFieldID:   field.ID,
		ChapterID:        chapter.ID,
	}
	res.Names = placementNames{
		EducationLevel: level.Name, Grade: grade.Name, Subject: subject.Name,
		SubjectField: field.Name, Chapter: chapter.Name,
	}
	if subOK {
		res.Request.SubchapterID = &sub.ID
		res.Names.Subchapter = sub.Name
	}
	if topicOK {
		res.Request.TopicID = &topic.ID
		res.Names.Topic = topic.Name
	}
	return res, true
}
