package auth

import (
	"errors"
	"log/slog"
	"net/http"

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

func (h *Handler) RegisterRouters() {
	g := h.router.Group("/auth")
	g.POST("/register", h.register)
	g.POST("/login", h.login)
}

func (h *Handler) register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.handleError(c, err)
		return
	}
	jwt, err := h.service.Register(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, RegisterResponse{JWT: jwt})
}

func (h *Handler) login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	jwt, err := h.service.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, LoginResponse{JWT: jwt})
}

func (h *Handler) handleError(c *gin.Context, err error) {
	slog.Error("Auth", "error", err)
	switch {
	case errors.Is(err, ErrCreateUser):
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	case errors.Is(err, ErrUserNotFound):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	case errors.Is(err, ErrUserAlredyExists):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	default:
		c.Status(http.StatusInternalServerError)
	}
}
