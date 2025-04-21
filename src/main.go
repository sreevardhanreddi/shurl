package main

import (
	"database/sql"
	"html/template"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"shurl/src/config"
	"shurl/src/db"
	"shurl/src/models"
	"shurl/src/validation"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	_ "github.com/joho/godotenv/autoload"
	"go.uber.org/zap"
)

var baseTmpl *template.Template

func main() {
	var err error

	router := gin.Default()
	router.Static("/static", "./src/static")
	logger := config.GetLogger()
	db, err := db.Connect()
	if err != nil {
		logger.Fatal("failed to connect to database", zap.Error(err))
	}

	baseTmpl, err = template.ParseFiles("src/templates/base.html")
	if err != nil {
		config.GetLogger().Fatal("failed to parse base template", zap.Error(err))
	}

	router.GET("/", func(c *gin.Context) {
		tmpl, err := template.Must(baseTmpl.Clone()).ParseFiles("src/templates/index.html")
		if err != nil {
			logger.Error("failed to parse index template", zap.Error(err))
			c.String(http.StatusInternalServerError, "Error rendering page")
			return
		}

		err = tmpl.ExecuteTemplate(c.Writer, "base", gin.H{
			"Title":          "Short URL Manager",
			"ShowBackButton": false,
		})
		if err != nil {
			logger.Error("failed to execute index template", zap.Error(err))
		}
	})

	router.GET("/:code", func(c *gin.Context) {
		code := c.Param("code")
		if code == "" {
			logger.Error("code is required")
			c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "code is required"})
			return
		}
		logger.Info("searching for code", zap.String("code", code))

		var url string
		var linkId int
		err = db.QueryRow("SELECT url, id FROM links WHERE code = $1", code).Scan(&url, &linkId)
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
	})

	router.POST("/api/generate", func(c *gin.Context) {
		var inputUrl models.InputUrl
		logger.Info("binding inputUrl")
		if err := c.ShouldBindJSON(&inputUrl); err != nil {
			logger.Error("failed to bind inputUrl", zap.Error(err))
			validation.HandleValidationErrors(c, err, inputUrl)
			return
		}
		logger.Info("inputUrl bound")

		logger.Info("database connected")
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
		err = db.QueryRow(sqlStatement, inputUrl.URL, inputUrl.CustomAlias, inputUrl.ExpiresAt).Scan(
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
	})

	router.GET("/api/links", func(c *gin.Context) {
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
	})

	router.GET("/api/links/visits/:id", func(c *gin.Context) {
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
	})

	router.DELETE("/api/links/:id", func(c *gin.Context) {
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
	})

	router.GET("/links/visits/:id", func(c *gin.Context) {
		id := c.Param("id")
		idInt, err := strconv.Atoi(id)
		if err != nil {
			logger.Error("invalid id format", zap.String("id", id), zap.Error(err))
			c.String(http.StatusBadRequest, "Invalid Link ID")
			return
		}

		var link models.Link
		err = db.QueryRow("SELECT id, url, code FROM links WHERE id = $1", idInt).Scan(&link.ID, &link.URL, &link.Code)
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

		tmpl, err := template.Must(baseTmpl.Clone()).ParseFiles("src/templates/visit_details.html")
		if err != nil {
			logger.Error("failed to parse visit_details template", zap.Error(err))
			c.String(http.StatusInternalServerError, "Error rendering page")
			return
		}

		err = tmpl.ExecuteTemplate(c.Writer, "base", gin.H{
			"Title":          "Visit Details",
			"ShowBackButton": true,
			"Visits":         visits,
			"Link":           link,
		})
		if err != nil {
			logger.Error("failed to execute visit_details template", zap.Error(err))
		}
	})

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
