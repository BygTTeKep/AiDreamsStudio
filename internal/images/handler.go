package images

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/BygTTeKep/AiDreamsStudio/internal/auth"
	"github.com/BygTTeKep/AiDreamsStudio/internal/projects"
	"github.com/gin-gonic/gin"
)

const (
	maxFileSize = 10 << 20 // 10 MB
	maxFiles    = 10
)

type Handler struct {
	service *Service
	router  *gin.Engine
}

func NewHandler(service *Service, router *gin.Engine) *Handler {
	return &Handler{
		service: service,
		router:  router,
	}
}

func (h *Handler) RegisterRouters(authMW gin.HandlerFunc) {
	g := h.router.Group("/images", authMW)
	g.POST("/:projectId/uploads", h.uploadImages)
	g.POST("/:projectId/list", h.listImages)
}

// UploadImages godoc
// @Summary  Загрузить картинки продукта
// @Tags     images
// @Accept   multipart/form-data
// @Produce  json
// @Param    id     path     string true "ID проекта"
// @Param    images formData file   true "Картинки"
// @Success  201 {object} UploadImagesResponse
// @Security BearerAuth
// @Router   /images/:projectId/uploads [post]
func (h *Handler) uploadImages(c *gin.Context) {
	userID, ok := auth.UserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// жёсткий лимит на тело запроса, до разбора формы
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxFiles*maxFileSize)

	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid multipart form"})
		return
	}
	files := form.File["images"]
	if len(files) == 0 || len(files) > maxFiles {
		c.JSON(http.StatusBadRequest, gin.H{"error": "1..10 files required"})
		return
	}

	images, err := h.service.UploadImages(c.Request.Context(), userID, c.Param("projectId"), files)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusCreated, UploadImagesResponse{Images: toImageResponses(images)})
}

// ListImages godoc
// @Summary ПОлучение картинок проекта
// @Tags images
// @Accept application/json
// @Produce json
// @Param input body ListImagesRequest true "Фильтрация с пагинацией"
// @Param projectId path string true "Id проекта"
// @Security BearerAuth
// @Router /images/:projectId/list [post]
func (h *Handler) listImages(c *gin.Context) {
	var req ListImagesRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err})
		return
	}
	userID, _ := auth.UserID(c)
	images, err := h.service.ListImages(c.Request.Context(), userID, c.Param("projectId"), req.Pagination.Limit, req.Pagination.Offset)
	if err != nil {
		h.handleError(c, err)
		return
	}
	resp := make([]ImageResponse, 0, len(images))
	for _, img := range images {
		resp = append(resp, ImageResponse{
			Id:          img.Id,
			ContentType: img.ContentType,
			URL:         "/api/v1/images/" + img.Id + "/file",
		})
	}
	c.JSON(http.StatusOK, gin.H{"images": resp})
}

func (h *Handler) getImageFile(c *gin.Context) {
	userID, ok := auth.UserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	img, rc, err := h.service.OpenImage(c.Request.Context(), userID, c.Param("id"))
	if err != nil {
		h.handleError(c, err)
		return
	}
	defer rc.Close()

	c.Header("Cache-Control", "private, max-age=3600")
	c.Header("X-Content-Type-Options", "nosniff")
	c.DataFromReader(http.StatusOK, int64(img.SizeBytes), img.ContentType, rc, nil)
}

func (h *Handler) handleError(c *gin.Context, err error) {
	slog.Error("files module", "error", err)
	switch {
	case errors.Is(err, projects.ErrProjectNotFound):
		c.JSON(http.StatusNotFound, err)
	case errors.Is(err, ErrImageNotFound):
		c.JSON(http.StatusNotFound, err)
	default:
		c.Status(http.StatusInternalServerError)
	}
}
