package handlers

import (
	"database/sql"
	"net/http"
	"shurl/src/models"
	"shurl/src/validation"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// HandleGenerateLink handles the request to generate a short URL
func HandleGenerateLink(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var inputUrl models.InputUrl
		logger.Info("binding inputUrl")
		if err := c.ShouldBindJSON(&inputUrl); err != nil {
			logger.Error("failed to bind inputUrl", zap.Error(err))
			validation.HandleValidationErrors(c, err, inputUrl)
			return
		}
		logger.Info("inputUrl bound")

		customAlias := inputUrl.CustomAlias
		if customAlias != "" {
			var existingUrl string
			logger.Info("checking if custom alias is already in database", zap.String("customAlias", customAlias))
			if err := db.QueryRow("SELECT url FROM links WHERE code = $1", customAlias).Scan(&existingUrl); err != nil {
				if err == sql.ErrNoRows {
					logger.Info("customAlias is not in database", zap.String("customAlias", customAlias))
				} else {
					logger.Error("failed to check if custom alias is already in database", zap.Error(err))
					c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "failed to check if custom alias is already in database"})
					return
				}
			} else {
				logger.Info("customAlias is already in database", zap.String("customAlias", customAlias))
				c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "custom alias already exists"})
				return
			}
		}

		if customAlias == "" {
			logger.Info("generating custom alias")
			for i := 0; i < 5; i++ {
				customAlias = uuid.New().String()[:6]
				logger.Info("customAlias generated", zap.String("customAlias", customAlias))
				var existingUrl string
				err := db.QueryRow("SELECT url FROM links WHERE code = $1", customAlias).Scan(&existingUrl)
				if err == sql.ErrNoRows {
					logger.Info("generated customAlias is unique", zap.String("customAlias", customAlias))
					break
				} else if err != nil {
					logger.Error("failed to check if generated custom alias is already in database", zap.Error(err))
					c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "failed to check alias uniqueness"})
					return
				}
				if i == 4 {
					logger.Error("failed to generate a unique alias after 5 attempts")
					c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "failed to generate unique alias"})
					return
				}
				logger.Warn("generated customAlias collision, retrying", zap.String("customAlias", customAlias))
			}
			inputUrl.CustomAlias = customAlias
		}

		logger.Info("inserting url into database")
		var createdLink models.Link
		sqlStatement := `INSERT INTO links (url, code, expires_at) VALUES ($1, $2, $3) RETURNING id, url, code, visits_count, created_at, updated_at, expires_at`
		err := db.QueryRow(sqlStatement, inputUrl.URL, inputUrl.CustomAlias, inputUrl.ExpiresAt).Scan(
			&createdLink.ID, &createdLink.URL, &createdLink.Code, &createdLink.VisitsCount,
			&createdLink.CreatedAt, &createdLink.UpdatedAt, &createdLink.ExpiresAt,
		)
		if err != nil {
			logger.Error("failed to insert url into database", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "failed to insert url into database"})
			return
		}
		logger.Info("url inserted into database")
		c.JSON(
			http.StatusCreated,
			gin.H{
				"status":  "success",
				"message": "URL created successfully",
				"data":    createdLink,
			},
		)
	}
}

// HandleListLinks handles the request to list all links
func HandleListLinks(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		page, offset := c.DefaultQuery("page", "1"), c.DefaultQuery("offset", "100")
		pageInt, _ := strconv.Atoi(page)
		offsetInt, _ := strconv.Atoi(offset)
		if pageInt < 1 {
			pageInt = 1
		}
		if offsetInt < 1 {
			offsetInt = 10
		}

		rows, err := db.Query("SELECT id, url, code, visits_count, created_at, updated_at, expires_at FROM links ORDER BY created_at DESC LIMIT $1 OFFSET $2", offsetInt, (pageInt-1)*offsetInt)
		if err != nil {
			logger.Error("failed to query rows", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "failed to query rows"})
			return
		}
		defer rows.Close()

		var links []models.Link
		for rows.Next() {
			var link models.Link
			err = rows.Scan(&link.ID, &link.URL, &link.Code, &link.VisitsCount, &link.CreatedAt, &link.UpdatedAt, &link.ExpiresAt)
			if err != nil {
				logger.Error("failed to scan row", zap.Error(err))
				c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "failed to scan row"})
				return
			}
			links = append(links, link)
		}
		if err = rows.Err(); err != nil {
			logger.Error("error iterating link rows", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "error reading links"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "success", "message": "links fetched successfully", "data": links})
	}
}

// HandleDeleteLink handles the request to delete a link
func HandleDeleteLink(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		idInt, err := strconv.Atoi(id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "invalid link ID format"})
			return
		}

		_, err = db.Exec("DELETE FROM visits WHERE link_id = $1", idInt)
		if err != nil {
			logger.Error("failed to delete associated visits", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "failed to delete associated visits"})
			return
		}

		result, err := db.Exec("DELETE FROM links WHERE id = $1", idInt)
		if err != nil {
			logger.Error("failed to delete link", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "failed to delete link"})
			return
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			c.JSON(http.StatusNotFound, gin.H{"status": "error", "message": "link not found"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "success", "message": "link deleted successfully"})
	}
}
