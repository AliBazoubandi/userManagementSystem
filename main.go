package main

import (
	"context"
	"fmt"
	"main/config"
	"main/connection"
	"main/controller"
	"main/db"
	"main/routes"
	"main/utility"
	"main/ws"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"main/docs"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	// Load logger
	utility.Init()
	logger := utility.AppLogger.Logger

	// Load configuration
	config.LoadConfig()

	// Load database connection
	connection.InitDatabase()
	defer connection.CloseDB()

	// Load redis connection
	connection.InitRedis()
	defer connection.CloseRedis()

	logger.Info("Application started")

	// Map database
	queries := db.New(connection.DB.Conn)
	redisClient := connection.RDB.Conn

	// Load user controller
	uc := controller.NewUserController(queries, redisClient, logger, false)

	// Load wsc controller
	h := ws.NewHub()
	wsc := ws.NewWsController(queries, h, logger)
	go h.Run()

	// Load router
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()

	// Metrics
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	api := router.Group("/api")
	docs.SwaggerInfo.BasePath = "/api"
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Register Routes
	routes.RegisterUserRoutes(api, uc, wsc)

	// start server
	serverAddress := fmt.Sprintf("%s:%d", config.AppConfig.Server.Host, config.AppConfig.Server.Port)
	server := &http.Server{
		Addr:         serverAddress,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  30 * time.Second,
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("server failed", zap.Error(err))
		}
	}()
	logger.Info("Server started", zap.String("address", serverAddress))

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
	logger.Info("server shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		logger.Fatal("Server forced to shutdown", zap.Error(err))
	}

	wg.Wait()
	logger.Info("Server gracefully stopped")
}
