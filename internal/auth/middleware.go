package auth

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	ctxUserIDKey = "auth.userId"
	ctxEmailKey  = "auth.email"
)

func (j *JWT) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		scheme, token, ok := strings.Cut(c.GetHeader("Authorization"), " ")
		if !ok || !strings.EqualFold(scheme, "Bearer") || token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing or invalid authorization header"})
			return
		}
		cl, err := j.ValidateToken(token)
		if err != nil {
			slog.Error("validate token", "error", err)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		c.Set(ctxUserIDKey, cl.UserID)
		c.Set(ctxEmailKey, cl.Email)
		c.Next()
	}
}

func UserID(c *gin.Context) (int, bool) {
	v, ok := c.Get(ctxUserIDKey)
	if !ok {
		return 0, false
	}
	id, ok := v.(int)
	return id, ok
}
