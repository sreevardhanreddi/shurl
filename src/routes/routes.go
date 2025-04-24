package routes

import (
	"database/sql"
	"shurl/src/handlers"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// SetupRoutes configures all the routes for the application
func SetupRoutes(router *gin.Engine, db *sql.DB, logger *zap.Logger) {
	// Set logger for handlers
	handlers.SetLogger(logger)

	// Static files
	router.Static("/static", "./src/static")

	// Page routes
	router.GET("/", handlers.HandleIndex)
	router.GET("/links/visits/:id", handlers.HandleVisitDetails(db))

	// API routes
	router.POST("/api/generate", handlers.HandleGenerateLink(db))
	router.GET("/api/links", handlers.HandleListLinks(db))
	router.GET("/api/links/visits/:id", handlers.HandleLinkVisits(db))
	router.DELETE("/api/links/:id", handlers.HandleDeleteLink(db))

	// Redirect route - must be last to avoid conflicts with other routes
	router.GET("/:code", handlers.HandleRedirect(db))
}
