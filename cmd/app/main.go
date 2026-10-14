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

	"github.com/BygTTeKep/AiDreamsStudio/internal/auth"
	"github.com/BygTTeKep/AiDreamsStudio/internal/config"
	"github.com/BygTTeKep/AiDreamsStudio/internal/database"
	"github.com/gin-gonic/gin"
)

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

	srv := &http.Server{
		Addr:           ":" + strconv.Itoa(cfg.App.Port),
		Handler:        router,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

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
