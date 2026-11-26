package learningmodule

import (
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"

	learningmoduledto "github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/dto/learning/learningmodule"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/middleware"
	"github.com/khalieftechcraftinnovation-dev/rumbasha-backend/internal/response"
)

// StudentList menangani GET /api/v1/student/learning-modules: modul di jenjang siswa, yang di luar
// paketnya ikut tampil dengan locked = true.
func (h *LearningModuleHandler) StudentList(c *gin.Context) {
	var q learningmoduledto.StudentListQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.ValidationError(c, err)
		return
	}
	access, modules, f, total, err := h.service.StudentList(c.Request.Context(), middleware.GetUserID(c), q)
	if err != nil {
		h.handleError(c, err)
		return
	}
	items := make([]learningmoduledto.StudentModuleResponse, 0, len(modules))
	for _, m := range modules {
		items = append(items, learningmoduledto.ToStudentModuleResponse(m, h.service.StudentFileURL(m.ID)))
	}
	response.SuccessWithMeta(c, "Daftar modul",
		learningmoduledto.StudentListResponse{Access: learningmoduledto.ToStudentAccessResponse(access), Modules: items},
		response.Meta{Page: f.Page, PerPage: f.PerPage, Total: total})
}

// Open menangani POST /api/v1/student/learning-modules/:id/open: buka modul (cek jenjang & paket,
// catat riwayat buka). data = modul + file.url untuk membaca PDF.
func (h *LearningModuleHandler) Open(c *gin.Context) {
	module, err := h.service.Open(c.Request.Context(), middleware.GetUserID(c), c.Param("id"))
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Modul dibuka", learningmoduledto.ToStudentModuleResponse(module, h.service.StudentFileURL(module.ID)))
}

// StudentFile menangani GET /api/v1/student/learning-modules/:id/file: kirim PDF (akses dicek).
func (h *LearningModuleHandler) StudentFile(c *gin.Context) {
	filePath, fileName, err := h.service.StudentFilePath(c.Request.Context(), middleware.GetUserID(c), c.Param("id"))
	if err != nil {
		h.handleError(c, err)
		return
	}
	sendPDF(c, filePath, fileName)
}

// sendPDF mengirim file PDF untuk ditampilkan langsung di browser.
func sendPDF(c *gin.Context, filePath, fileName string) {
	c.Header("Content-Disposition", "inline; filename*=UTF-8''"+url.PathEscape(fileName))
	c.Header("Cache-Control", "private, max-age=0")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Content-Type", "application/pdf")
	c.File(filePath)
}
