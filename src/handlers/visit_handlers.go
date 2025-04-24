package handlers

import (
	"database/sql"
	"net/http"
	"shurl/src/models"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// HandleLinkVisits returns the visits for a specific link
func HandleLinkVisits(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		_, err := strconv.Atoi(id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "invalid link ID format"})
			return
		}

		rows, err := db.Query("SELECT id, link_id, ip_address, user_agent, referrer, created_at, updated_at FROM visits WHERE link_id = $1 ORDER BY created_at DESC", id)
		if err != nil {
			logger.Error("failed to query visit rows", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "failed to query visits"})
			return
		}
		defer rows.Close()

		var visits []models.Visit
		for rows.Next() {
			var visit models.Visit
			err = rows.Scan(&visit.ID, &visit.LinkID, &visit.IPAddress, &visit.UserAgent, &visit.Referrer, &visit.CreatedAt, &visit.UpdatedAt)
			if err != nil {
				logger.Error("failed to scan visit row", zap.Error(err))
				c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "failed to scan visit row"})
				return
			}
			visits = append(visits, visit)
		}
		if err = rows.Err(); err != nil {
			logger.Error("error iterating visit rows", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "error reading visits"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "success", "message": "visits fetched successfully", "data": visits})
	}
}
