package handlers

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// HandleRedirect redirects to the original URL when a short code is accessed
func HandleRedirect(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		code := c.Param("code")
		if code == "" {
			logger.Error("code is required")
			c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "code is required"})
			return
		}
		logger.Info("searching for code", zap.String("code", code))

		var url string
		var linkId int
		err := db.QueryRow("SELECT url, id FROM links WHERE code = $1", code).Scan(&url, &linkId)
		if err != nil {
			if err == sql.ErrNoRows {
				c.JSON(http.StatusNotFound, gin.H{"status": "error", "message": "url not found"})
			} else {
				logger.Error("failed to query row", zap.Error(err))
				c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "internal server error"})
			}
			return
		}
		logger.Info("found url", zap.String("url", url))

		logger.Info("updating click count")
		_, err = db.Exec("UPDATE links SET visits_count = visits_count + 1 WHERE id = $1", linkId)
		if err != nil {
			logger.Error("failed to update click count", zap.Error(err))
		} else {
			logger.Info("click count updated")
		}

		logger.Info("recording visit details")
		referrer := c.Request.Referer()
		ip := c.ClientIP()
		userAgent := c.Request.UserAgent()
		_, err = db.Exec("INSERT INTO visits (link_id, ip_address, user_agent, referrer) VALUES ($1, $2, $3, $4)", linkId, ip, userAgent, referrer)
		if err != nil {
			logger.Error("failed to record visit", zap.Error(err))
		} else {
			logger.Info("visit recorded")
		}

		c.Redirect(http.StatusTemporaryRedirect, url)
	}
}
