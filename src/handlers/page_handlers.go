package handlers

import (
	"database/sql"
	"fmt"
	"html/template"
	"net/http"
	"shurl/src/models"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Logger instance
var logger *zap.Logger

// SetLogger sets the logger for handlers
func SetLogger(l *zap.Logger) {
	logger = l
}

// HandleIndex handles the index page request
func HandleIndex(c *gin.Context) {
	tmpl, err := template.ParseFiles("src/templates/base.html", "src/templates/index.html")
	if err != nil {
		logger.Error("failed to parse templates", zap.Error(err))
		c.String(http.StatusInternalServerError, "Error rendering page")
		return
	}

	err = tmpl.ExecuteTemplate(c.Writer, "base", gin.H{
		"Title":          "Shurl - Short URL Manager",
		"ShowBackButton": false,
	})
	if err != nil {
		logger.Error("failed to execute index template", zap.Error(err))
	}
}

// HandleVisitDetails handles the visit details page
func HandleVisitDetails(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		idInt, err := strconv.Atoi(id)
		if err != nil {
			logger.Error("invalid id format", zap.String("id", id), zap.Error(err))
			c.String(http.StatusBadRequest, "Invalid Link ID")
			return
		}

		var link models.Link
		err = db.QueryRow("SELECT id, url, code, visits_count, created_at, updated_at, expires_at FROM links WHERE id = $1", idInt).Scan(
			&link.ID, &link.URL, &link.Code, &link.VisitsCount, &link.CreatedAt, &link.UpdatedAt, &link.ExpiresAt)
		if err != nil {
			if err == sql.ErrNoRows {
				logger.Warn("link not found for visit details", zap.Int("id", idInt))
				c.String(http.StatusNotFound, "Link not found")
			} else {
				logger.Error("failed to query link for visit details", zap.Int("id", idInt), zap.Error(err))
				c.String(http.StatusInternalServerError, "Error fetching link details")
			}
			return
		}

		rows, err := db.Query("SELECT id, link_id, ip_address, user_agent, referrer, created_at, updated_at FROM visits WHERE link_id = $1 ORDER BY created_at DESC", id)
		if err != nil {
			logger.Error("failed to query visit rows", zap.Error(err))
			c.String(http.StatusInternalServerError, "Error fetching visits")
			return
		}
		defer rows.Close()

		var visits []models.Visit
		for rows.Next() {
			var visit models.Visit
			err = rows.Scan(&visit.ID, &visit.LinkID, &visit.IPAddress, &visit.UserAgent, &visit.Referrer, &visit.CreatedAt, &visit.UpdatedAt)
			if err != nil {
				logger.Error("failed to scan visit row", zap.Error(err))
				c.String(http.StatusInternalServerError, "Error reading visit data")
				return
			}
			visits = append(visits, visit)
		}
		if err = rows.Err(); err != nil {
			logger.Error("error iterating visit rows", zap.Error(err))
			c.String(http.StatusInternalServerError, "Error reading visits")
			return
		}

		tmpl, err := template.ParseFiles("src/templates/base.html", "src/templates/visit_details.html")
		if err != nil {
			logger.Error("failed to parse visit_details template", zap.Error(err))
			c.String(http.StatusInternalServerError, "Error rendering page")
			return
		}

		err = tmpl.ExecuteTemplate(c.Writer, "base", gin.H{
			"Title":          fmt.Sprintf("Visit Details for - %s", link.URL),
			"ShowBackButton": true,
			"Visits":         visits,
			"Link":           link,
		})
		if err != nil {
			logger.Error("failed to execute visit_details template", zap.Error(err))
		}
	}
}
