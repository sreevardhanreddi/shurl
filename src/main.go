package main

import (
	"database/sql"
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

func main() {

	router := gin.Default()
	router.LoadHTMLGlob("src/templates/*")
	router.Static("/static", "./static")
	logger := config.GetLogger()
	db, err := db.Connect()
	if err != nil {
		logger.Fatal("failed to connect to database", zap.Error(err))
	}

	// router.Use(ginzap.Ginzap(logger, time.RFC3339, true))

	router.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", gin.H{
			"Title":          "Short URL Manager",
			"ShowBackButton": false,
		})
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
			logger.Error("failed to query row", zap.Error(err))
			c.JSON(http.StatusNotFound, gin.H{"status": "error", "message": "url not found"})
			return
		}
		logger.Info("found url", zap.String("url", url))

		logger.Info("updating click count")
		_, err = db.Exec("UPDATE links SET visits_count = visits_count + 1 WHERE id = $1", linkId)
		if err != nil {
			logger.Error("failed to update click count", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "failed to update click count"})
			return
		}
		logger.Info("click count updated")

		logger.Info("updating referrer, ip, user_agent")
		referrer := c.Request.Referer()
		ip := c.ClientIP()
		userAgent := c.Request.UserAgent()
		_, err = db.Exec("INSERT INTO visits (link_id, ip_address, user_agent, referrer) VALUES ($1, $2, $3, $4)", linkId, ip, userAgent, referrer)
		if err != nil {
			logger.Error("failed to update referrer, ip, user_agent", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "failed to update referrer, ip, user_agent"})
			return
		}
		logger.Info("referrer, ip, user_agent updated")
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
			var url string
			logger.Info("checking if custom alias is already in database", zap.String("customAlias", customAlias))
			if err := db.QueryRow("SELECT url FROM links WHERE code = $1", customAlias).Scan(&url); err != nil {
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
			customAlias = uuid.New().String()[:6]
			logger.Info("customAlias generated", zap.String("customAlias", customAlias))
			var url string
			// check if custom alias is already in database
			if err := db.QueryRow("SELECT url FROM links WHERE code = $1", customAlias).Scan(&url); err != nil {
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

			inputUrl.CustomAlias = customAlias
		}

		logger.Info("inserting url into database")
		var url models.InputUrl
		sqlStatement := `INSERT INTO links (url, code, expires_at) VALUES ($1, $2, $3) RETURNING url, code as custom_alias, expires_at`
		err = db.QueryRow(sqlStatement, inputUrl.URL, inputUrl.CustomAlias, inputUrl.ExpiresAt).Scan(&url.URL, &url.CustomAlias, &url.ExpiresAt)
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
				"data":    url,
			},
		)

	})

	router.GET("/api/links", func(c *gin.Context) {
		page, offset := c.DefaultQuery("page", "1"), c.DefaultQuery("offset", "100")
		pageInt, _ := strconv.Atoi(page)
		offsetInt, _ := strconv.Atoi(offset)

		rows, err := db.Query("SELECT id, url, code, visits_count, created_at, updated_at FROM links ORDER BY created_at DESC LIMIT $1 OFFSET $2", offsetInt, (pageInt-1)*offsetInt)
		if err != nil {
			logger.Error("failed to query rows", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "failed to query rows"})
			return
		}
		defer rows.Close()
		var links []models.Link
		for rows.Next() {
			var link models.Link
			err = rows.Scan(&link.ID, &link.URL, &link.Code, &link.VisitsCount, &link.CreatedAt, &link.UpdatedAt)
			if err != nil {
				logger.Error("failed to scan row", zap.Error(err))
				c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "failed to scan row"})
				return
			}
			links = append(links, link)
		}
		c.JSON(http.StatusOK, gin.H{"status": "success", "message": "links fetched successfully", "data": links})
	})

	router.GET("/api/links/visits/:id", func(c *gin.Context) {
		id := c.Param("id")

		rows, err := db.Query("SELECT id, link_id, ip_address, user_agent, referrer, created_at, updated_at FROM visits WHERE link_id = $1 ORDER BY created_at DESC", id)
		if err != nil {
			logger.Error("failed to query rows", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "failed to query rows"})
			return
		}
		defer rows.Close()
		var visits []models.Visit
		for rows.Next() {
			var visit models.Visit
			err = rows.Scan(&visit.ID, &visit.LinkID, &visit.IPAddress, &visit.UserAgent, &visit.Referrer, &visit.CreatedAt, &visit.UpdatedAt)
			if err != nil {
				logger.Error("failed to scan row", zap.Error(err))
				c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "failed to scan row"})
				return
			}
			visits = append(visits, visit)
		}
		c.JSON(http.StatusOK, gin.H{"status": "success", "message": "visits fetched successfully", "data": visits})
	})

	router.DELETE("/api/links/:id", func(c *gin.Context) {
		id := c.Param("id")

		_, err = db.Exec("DELETE FROM links WHERE id = $1", id)
		if err != nil {
			logger.Error("failed to delete link", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "failed to delete link"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "success", "message": "link deleted successfully"})
	})

	router.GET("/links/visits/:id", func(c *gin.Context) {
		id := c.Param("id")
		rows, err := db.Query("SELECT id, link_id, ip_address, user_agent, referrer, created_at, updated_at FROM visits WHERE link_id = $1 ORDER BY created_at DESC", id)
		if err != nil {
			logger.Error("failed to query rows", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "failed to query rows"})
			return
		}
		defer rows.Close()
		var visits []models.Visit
		for rows.Next() {
			var visit models.Visit
			err = rows.Scan(&visit.ID, &visit.LinkID, &visit.IPAddress, &visit.UserAgent, &visit.Referrer, &visit.CreatedAt, &visit.UpdatedAt)
			if err != nil {
				logger.Error("failed to scan row", zap.Error(err))
				c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "failed to scan row"})
				return
			}
			visits = append(visits, visit)
		}
		c.HTML(http.StatusOK, "visit_details.html", gin.H{
			"Title":          "Visit Details",
			"ShowBackButton": true,
			"data":           visits,
		})
	})

	logger.Info("starting server on port", zap.String("port", os.Getenv("PORT")))
	server := &http.Server{
		Addr:    ":" + os.Getenv("PORT"),
		Handler: router,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-quit
		logger.Info("received interrupt signal, shutting down...")
		if err := server.Close(); err != nil {
			logger.Fatal("server close: %v", zap.Error(err))
		}
	}()

	if err := server.ListenAndServe(); err != nil {
		if err == http.ErrServerClosed {
			logger.Info("Server closed under request")
		} else {
			logger.Fatal("Server closed unexpectdly: %v", zap.Error(err))
		}
	}

	server.ListenAndServe()
}
