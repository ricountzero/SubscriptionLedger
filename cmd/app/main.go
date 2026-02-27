package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/pressly/goose/v3"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/zap"

	_ "github.com/jackc/pgx/v5/stdlib"
	_ "github.com/ricountzero/SubscriptionLedger/docs"
	"github.com/ricountzero/SubscriptionLedger/internal/config"
	"github.com/ricountzero/SubscriptionLedger/internal/handler"
	"github.com/ricountzero/SubscriptionLedger/internal/middleware"
	"github.com/ricountzero/SubscriptionLedger/internal/repository"
	"github.com/ricountzero/SubscriptionLedger/internal/service"
)

// @title           Subscription Ledger API
// @version         1.0
// @description     REST API for aggregating user online subscriptions
// @host            localhost:8080
// @BasePath        /
func main() {
	_ = godotenv.Load()

	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("failed to init logger: %v", err)
	}
	defer logger.Sync()

	cfg, err := config.Load("config.yaml")
	if err != nil {
		logger.Fatal("failed to load config", zap.Error(err))
	}

	// Connect to DB
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, cfg.Database.DSN())
	if err != nil {
		logger.Fatal("failed to connect to database", zap.Error(err))
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		logger.Fatal("database ping failed", zap.Error(err))
	}
	logger.Info("connected to database")

	// Run migrations
	db, err := goose.OpenDBWithDriver("postgres", cfg.Database.DSN())
	if err != nil {
		logger.Fatal("failed to open db for migrations", zap.Error(err))
	}
	if err := goose.Up(db, "migrations"); err != nil {
		logger.Fatal("failed to run migrations", zap.Error(err))
	}
	logger.Info("migrations applied")

	// Wire dependencies
	repo := repository.NewSubscriptionRepository(pool)
	svc := service.NewSubscriptionService(repo, logger)
	h := handler.NewSubscriptionHandler(svc, logger)

	// Setup router
	r := gin.New()
	r.Use(middleware.Logger(logger))
	r.Use(gin.Recovery())

	h.RegisterRoutes(r)

	// Swagger UI
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	addr := fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port)
	srv := &http.Server{Addr: addr, Handler: r}

	go func() {
		logger.Info("starting server", zap.String("addr", addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("server error", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down server")
	shutCtx, shutCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutCancel()
	if err := srv.Shutdown(shutCtx); err != nil {
		logger.Error("server shutdown error", zap.Error(err))
	}
	logger.Info("server stopped")
}
