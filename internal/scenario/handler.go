package scenario

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/BygTTeKep/AiDreamsStudio/internal/auth"
	"github.com/BygTTeKep/AiDreamsStudio/internal/generation/claude"
	"github.com/BygTTeKep/AiDreamsStudio/internal/projects"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{
		service: service,
	}
}

func (h *Handler) RegisterRouters(rg *gin.RouterGroup, authMW gin.HandlerFunc) {
	g := rg.Group("/scenario", authMW)
	g.POST("/create", h.createScenario)
	g.POST("/generate", h.generateScenarios)
	g.GET("/:id", h.getScenarioById)
	g.POST("/:id/approve", h.approveScenario)
	g.POST("/search", h.searchScenarios)
}

// CreateScenario godoc
// @Summary Создание сценария руками
// @Description Сценарий сохраняется с source=user и status=pending_review. Кадры нумеруются по порядку position.
// @Tags scenario
// @Accept json
// @Produce json
// @Param projectId path string true "Project id"
// @Param request body CreateScenarioRequest true "Сценарий"
// @Success 201 {object} CreateScenarioResponse
// @Security BearerAuth
// @Router /projects/{projectId}/scenario/create [post]
func (h *Handler) createScenario(c *gin.Context) {
	userId, ok := auth.UserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{})
		return
	}
	var req CreateScenarioRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	projectId := c.Param("projectId")
	id, err := h.service.SaveScenario(c.Request.Context(), projectId, req, userId)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusCreated, CreateScenarioResponse{Id: id})
}

// GenerateScenarios godoc
// @Summary Генерация сценариев через Claude (асинхронно)
// @Description Ставит задачу в очередь. Готовые сценарии сохраняются со статусом pending_review и source=llm.
// @Tags scenario
// @Accept json
// @Produce json
// @Param projectId path string true "Project id"
// @Param request body GenerateScenarioRequest true "Параметры генерации"
// @Success 202 {object} GenerateScenarioResponse
// @Security BearerAuth
// @Router /projects/{projectId}/scenario/generate [post]
func (h *Handler) generateScenarios(c *gin.Context) {
	userId, ok := auth.UserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{})
		return
	}
	var req GenerateScenarioRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	projectId := c.Param("projectId")
	if err := h.service.GenerateScenarios(c.Request.Context(), projectId, req, userId); err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, GenerateScenarioResponse{Status: "queued"})
}

// GetScenarioById godoc
// @Summary Получение сценария по id
// @Tags scenario
// @Produce json
// @Param projectId path string true "Project id"
// @Param id path string true "Scenario id"
// @Success 200 {object} ScenarioResponse
// @Security BearerAuth
// @Router /projects/{projectId}/scenario/{id} [get]
func (h *Handler) getScenarioById(c *gin.Context) {
	userId, ok := auth.UserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{})
		return
	}
	sc, err := h.service.GetScenarioById(c.Request.Context(), c.Param("projectId"), c.Param("id"), userId)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, sc)
}

// ApproveScenario godoc
// @Summary Апрув сценария
// @Description Переводит сценарий из pending_review в approved. 409 — если сценарий уже не на ревью.
// @Tags scenario
// @Produce json
// @Param projectId path string true "Project id"
// @Param id path string true "Scenario id"
// @Success 200 {object} ScenarioResponse
// @Security BearerAuth
// @Router /projects/{projectId}/scenario/{id}/approve [post]
func (h *Handler) approveScenario(c *gin.Context) {
	userId, ok := auth.UserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{})
		return
	}
	sc, err := h.service.ApproveScenario(c.Request.Context(), c.Param("projectId"), c.Param("id"), userId)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, sc)
}

// SearchScenarios godoc
// @Summary Посик всех сценариев проекта
// @Description Поиск сценариев проекта с пагинацией
// @Tags scenario
// @Produce json
// @Param request body ScenarioSearchRequest true "Параметры поиска"
// @Success 200 {object} []ScenarioResponse
// @Security BearerAuth
// @Router /projects/{projectId}/scenario/search [post]
func (h *Handler) searchScenarios(c *gin.Context) {
	userId, ok := auth.UserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{})
		return
	}
	var req ScenarioSearchRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	scenarios, err := h.service.SearchScenarios(c.Request.Context(), userId, req.Pagination.Limit, req.Pagination.Offset, c.Param("projectId"))
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, scenarios)
}

func (h *Handler) handleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, projects.ErrProjectNotFound), errors.Is(err, ErrScenarioNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, ErrScenarioNotPending):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	case errors.Is(err, claude.ErrQueueFull), errors.Is(err, claude.ErrWorkerStopped):
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
	default:
		slog.Error("scenario handler", "path", c.FullPath(), "error", err)
		c.Status(http.StatusInternalServerError)
	}
}
