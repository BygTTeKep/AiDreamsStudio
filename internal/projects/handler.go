package projects

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/BygTTeKep/AiDreamsStudio/internal/auth"
	"github.com/gin-gonic/gin"
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
	g := h.router.Group("/projects", authMW)
	g.POST("/create", h.create)
	g.GET("/:id", h.getByID)
	g.POST("/search", h.search)
}

// CreateProject godoc
// @Summary  Создать проект
// @Tags     projects
// @Accept   json
// @Produce  json
// @Param    input body     CreateProjectRequest true "Данные проекта"
// @Success  201   {object} CreateProjectResponse
// @Security BearerAuth
// @Router   /projects [post]
func (h *Handler) create(c *gin.Context) {
	var req CreateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	userID, ok := auth.UserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	projectId, err := h.service.CreateProject(c.Request.Context(), req.Name, userID)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusCreated, CreateProjectResponse{Id: projectId})
}

// GetProjectByID godoc
// @Summary Получить проект по id
// @Tags projects
// @Accept json
// @Param id path string true "Project id"
// @Success 200 {object} GetProjectByIdReponse
// @Security BearerAuth
// @Router /projects/:id [get]
func (h *Handler) getByID(c *gin.Context) {
	var req GetProjectByIdRequest
	rawId := c.Param("id")
	req.Id = rawId
	userId, ok := auth.UserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	project, err := h.service.FindById(c.Request.Context(), req.Id, userId)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, GetProjectByIdReponse{Id: project.ID, Name: project.Name, CreatedAt: project.CreatedAt})
}

// SearchProjects godoc
// @Summary Поиск проектов пользователя
// @Tags projects
// @Accept json
// @Param input body SearchProjectRequest true "Фильтр проектов"
// @Security BearerAuth
// @Router /projects/search [post]
func (h *Handler) search(c *gin.Context) {
	var req SearchProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	userId, ok := auth.UserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	projects, err := h.service.Search(c.Request.Context(), userId, req.Pagination.Limit, req.Pagination.Offset)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, SearchProjectResponse{Projects: toProjectResponses(projects)})
}

func (h *Handler) handleError(c *gin.Context, err error) {
	slog.Error("projects", "error", err)
	switch {
	case errors.Is(err, ErrProjectNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	default:
		c.Status(http.StatusInternalServerError)
	}
}
