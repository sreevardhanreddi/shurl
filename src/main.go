package main

import (
	"errors"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"shurl/src/validation"

	"github.com/gin-gonic/gin"
	_ "github.com/joho/godotenv/autoload"
)

type InputUrl struct {
	URL         string     `json:"url" binding:"required,url"`
	CustomAlias string     `json:"custom_alias" binding:"omitempty,alphanum,min=3,max=6"`
	ExpiresAt   *time.Time `json:"expires_at" binding:"omitempty,gt=now"`
}

func isValidURL(urlStr string) bool {
	_, err := url.ParseRequestURI(urlStr)
	return err == nil
}

func main() {

	router := gin.Default()

	router.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "Hello, World"})
	})

	router.GET("/:url", func(c *gin.Context) {
		url := c.Param("url")
		if !isValidURL(url) {
			c.JSON(http.StatusBadRequest, gin.H{"status": "error", "errors": errors.New("invalid url")})
			return
		}
		// check if url is in database

		c.JSON(http.StatusOK, gin.H{"url": url})
	})

	router.POST("/generate", func(c *gin.Context) {
		var inputUrl InputUrl
		if err := c.ShouldBindJSON(&inputUrl); err != nil {
			validation.HandleValidationErrors(c, err, inputUrl)
			return
		}

		c.JSON(http.StatusOK, gin.H{"status": "success", "url": inputUrl.URL})
	})

	server := &http.Server{
		Addr:    ":" + os.Getenv("PORT"),
		Handler: router,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-quit
		log.Println("received interrupt signal, shutting down...")
		if err := server.Close(); err != nil {
			log.Fatalf("server close: %v", err)
		}
	}()

	if err := server.ListenAndServe(); err != nil {
		if err == http.ErrServerClosed {
			log.Println("Server closed under request")
		} else {
			log.Fatalf("Server closed unexpectdly: %v", err)
		}
	}

	server.ListenAndServe()
}
