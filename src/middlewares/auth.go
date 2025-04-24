package middlewares

import (
	"crypto/subtle"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

// BasicAuth is a middleware that performs basic HTTP authentication.
// It reads username and password from environment variables.
func BasicAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get credentials from environment variables
		username := os.Getenv("BASIC_AUTH_USERNAME")
		password := os.Getenv("BASIC_AUTH_PASSWORD")

		// Skip authentication if credentials are not set
		if username == "" || password == "" {
			c.Next()
			return
		}

		// Get authentication header
		user, pass, hasAuth := c.Request.BasicAuth()

		// Check if authentication is provided and credentials are correct
		if !hasAuth ||
			subtle.ConstantTimeCompare([]byte(user), []byte(username)) != 1 ||
			subtle.ConstantTimeCompare([]byte(pass), []byte(password)) != 1 {

			// Unauthorized response
			c.Header("WWW-Authenticate", "Basic realm=Authorization Required")
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		c.Next()
	}
}
