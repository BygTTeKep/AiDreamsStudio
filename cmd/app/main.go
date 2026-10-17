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
	"github.com/BygTTeKep/AiDreamsStudio/internal/generation/claude"
	"github.com/BygTTeKep/AiDreamsStudio/internal/images"
	"github.com/BygTTeKep/AiDreamsStudio/internal/projects"
	"github.com/BygTTeKep/AiDreamsStudio/internal/scenario"
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
	projectsGroup := router.Group("/projects", jwt.Middleware())
	projectGroup := projectsGroup.Group("/:projectId")

	projectRepo := projects.NewRepository(db)
	projectService := projects.NewService(projectRepo)
	projectHandler := projects.NewHandler(projectService)
	projectHandler.RegisterRouters(projectsGroup, jwt.Middleware())

	//Images модуль
	imageRepo := images.NewRepository(db)
	imageStore, err := images.NewStorage(cfg.Storage.URL)
	if err != nil {
		slog.Error("image store", "error", err)
		os.Exit(1)
	}

	imageService := images.NewService(imageStore, imageRepo)
	imageHandler := images.NewHandler(imageService)
	imageHandler.RegisterRouters(projectGroup, jwt.Middleware())

	//Scenario
	claudeClient := claude.NewClient(claude.Config{
		APIKey:    cfg.Claude.APIKey,
		Model:     cfg.Claude.Model,
		MaxTokens: cfg.Claude.MaxTokens,
		Effort:    cfg.Claude.Effort,
	})
	scenarioWorker := claude.NewWorker(claudeClient, nil, claude.WorkerConfig{
		Workers:   cfg.Claude.Workers,
		QueueSize: cfg.Claude.QueueSize,
		Timeout:   cfg.Claude.Timeout,
	})

	scenarioRepo := scenario.NewRepository(db)
	scenarioService := scenario.NewService(scenarioRepo, scenarioWorker, imageService)
	scenarioWorker.SetSaver(scenarioService)
	// воркер живёт дольше ctx запроса, поэтому получает свой контекст и останавливается после HTTP-сервера
	workerCtx, cancelWorker := context.WithCancel(context.Background())
	defer cancelWorker()
	scenarioWorker.Start(workerCtx)
	scenarioHanler := scenario.NewHandler(scenarioService)
	scenarioHanler.RegisterRouters(projectGroup, jwt.Middleware())

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

	// дожидаемся задач, уже взятых в работу; по таймауту — отменяем их
	go func() {
		<-ctxShutdown.Done()
		cancelWorker()
	}()
	scenarioWorker.Stop()
}
