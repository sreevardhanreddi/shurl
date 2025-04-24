package main

import (
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"shurl/src/config"
	"shurl/src/db"
	"shurl/src/routes"

	"github.com/gin-gonic/gin"
	_ "github.com/joho/godotenv/autoload"
	"go.uber.org/zap"
)

func main() {
	var err error

	router := gin.Default()
	logger := config.GetLogger()
	db, err := db.Connect()
	if err != nil {
		logger.Fatal("failed to connect to database", zap.Error(err))
	}

	// Set up all routes
	routes.SetupRoutes(router, db, logger)

	logger.Info("starting server on port", zap.String("port", os.Getenv("PORT")))
	server := &http.Server{
		Addr:    ":" + os.Getenv("PORT"),
		Handler: router,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("listen: %s\n", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("Shutting down server...")

	if db != nil {
		if err := db.Close(); err != nil {
			logger.Error("Error closing database connection", zap.Error(err))
		} else {
			logger.Info("Database connection closed.")
		}
	}

	logger.Info("Server exiting")
}
