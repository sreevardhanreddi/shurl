package routes

import (
	"database/sql"
	"shurl/src/handlers"
	"shurl/src/middlewares"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// SetupRoutes configures all the routes for the application
func SetupRoutes(router *gin.Engine, db *sql.DB, logger *zap.Logger) {
	// Set logger for handlers
	handlers.SetLogger(logger)

	// Static files
	router.Static("/static", "./src/static")

	// Protected routes group
	protected := router.Group("/")
	protected.Use(middlewares.BasicAuth())
	{
		// Page routes
		protected.GET("/", handlers.HandleIndex)
		protected.GET("/links/visits/:id", handlers.HandleVisitDetails(db))

		// API routes
		protected.POST("/api/generate", handlers.HandleGenerateLink(db))
		protected.GET("/api/links", handlers.HandleListLinks(db))
		protected.GET("/api/links/visits/:id", handlers.HandleLinkVisits(db))
		protected.DELETE("/api/links/:id", handlers.HandleDeleteLink(db))
	}

	// Redirect route - must be last to avoid conflicts with other routes
	// Not protected by authentication
	router.GET("/:code", handlers.HandleRedirect(db))
}
