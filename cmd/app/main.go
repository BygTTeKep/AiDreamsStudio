package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	_ "github.com/BygTTeKep/AiDreamsStudio/docs"

	_ "github.com/BygTTeKep/AiDreamsStudio"
	"github.com/BygTTeKep/AiDreamsStudio/internal/auth"
	"github.com/BygTTeKep/AiDreamsStudio/internal/config"
	"github.com/BygTTeKep/AiDreamsStudio/internal/database"
	"github.com/BygTTeKep/AiDreamsStudio/internal/projects"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @title           Video Service API
// @version         1.0
// @host            localhost:3000
// @BasePath        /
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		slog.Error("config init", "error", err)
		os.Exit(1)
	}

	db, err := database.NewDB(ctx, cfg.DB)
	if err != nil {
		slog.Error("database init", "error", err)
		os.Exit(1)
	}

	router := gin.Default()

	// AUTH модуль
	authRepo := auth.NewRepository(db)
	jwt := auth.NewJWT(&cfg.JWT)
	authService := auth.NewService(authRepo, jwt)
	authHandler := auth.NewHandler(authService, router)
	authHandler.RegisterRouters()

	// Projects модуль
	projectRepo := projects.NewRepository(db)
	projectService := projects.NewService(projectRepo)
	projectHandler := projects.NewHandler(projectService, router)
	projectHandler.RegisterRouters(jwt.Middleware())

	srv := &http.Server{
		Addr:           ":" + strconv.Itoa(cfg.App.Port),
		Handler:        router,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server start", "error", err)
			stop()
		}
	}()

	<-ctx.Done()
	ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancelShutdown()
	if err := srv.Shutdown(ctxShutdown); err != nil {
		slog.Error("server shutdown", "error", err)
	}
}
