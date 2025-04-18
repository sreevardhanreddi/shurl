package validation

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"reflect"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// ValidationError wraps the validator error to customize the response
type ValidationError struct {
	Location string      `json:"location"`
	Message  string      `json:"message"`
	Field    string      `json:"field"`
	Value    interface{} `json:"value"`
}

// getFieldLocation determines where a field is located in the request
func getFieldLocation(c *gin.Context, fieldName string) string {
	// Check URL params first
	if _, exists := c.Params.Get(fieldName); exists {
		return "params"
	}

	// Check query params
	if _, exists := c.GetQuery(fieldName); exists {
		return "query"
	}

	// Check form data
	if _, exists := c.GetPostForm(fieldName); exists {
		return "form"
	}

	// Default to body for JSON requests
	contentType := c.ContentType()
	if strings.Contains(contentType, "application/json") {
		return "body"
	}

	// Fallback to request
	return "request"
}

// formatValidationErrors converts validator errors into a more readable format
func formatValidationErrors(c *gin.Context, err error) []ValidationError {
	var validationErrors []ValidationError

	// Try to convert to validator.ValidationErrors
	var ve validator.ValidationErrors
	if errors.As(err, &ve) {
		for _, e := range ve {
			// Debug logging for validation error
			log.Println("Validation Error Details:")
			log.Printf("Type: %T\n", e)
			log.Printf("Field: %s\n", e.Field())
			log.Printf("Tag: %s\n", e.Tag())
			log.Printf("Value: %v\n", e.Value())
			log.Printf("Param: %s\n", e.Param())
			log.Printf("Error: %s\n", e.Error())
			log.Printf("Namespace: %s\n", e.Namespace())
			log.Printf("StructNamespace: %s\n", e.StructNamespace())
			log.Printf("StructField: %s\n", e.StructField())
			log.Printf("ActualTag: %s\n", e.ActualTag())
			log.Printf("Kind: %v\n", e.Kind())
			log.Printf("Type: %v\n", e.Type())

			// Print all available methods using reflection
			t := reflect.TypeOf(e)
			log.Println("\nAvailable Methods:")
			for i := 0; i < t.NumMethod(); i++ {
				method := t.Method(i)
				log.Printf("Method %d: %s\n", i, method.Name)
			}

			fieldName := e.Tag()
			location := getFieldLocation(c, fieldName)

			// Create a custom error message based on the validation tag
			var errorMsg string
			switch e.Tag() {
			case "required":
				errorMsg = "This field is required"
			case "url":
				errorMsg = "Must be a valid URL"
			case "min":
				errorMsg = fmt.Sprintf("Must be at least %s characters long", e.Param())
			case "max":
				errorMsg = fmt.Sprintf("Must be at most %s characters long", e.Param())
			case "oneof":
				errorMsg = fmt.Sprintf("Must be one of: %s", e.Param())
			default:
				errorMsg = fmt.Sprintf("Failed on the '%s' validation rule", e.Tag())
			}

			validationErrors = append(validationErrors, ValidationError{
				Location: location,
				Message:  errorMsg,
				Field:    fieldName,
				Value:    e.Value(),
			})
		}
		return validationErrors
	}

	// If it's not a validator.ValidationErrors, return a generic error
	validationErrors = append(validationErrors, ValidationError{
		Location: "request",
		Message:  err.Error(),
		Field:    "request",
		Value:    nil,
	})

	return validationErrors
}

// HandleValidationErrors processes validation errors and returns a formatted response
func HandleValidationErrors(c *gin.Context, err error) {
	validationErrors := formatValidationErrors(c, err)
	c.JSON(http.StatusBadRequest, gin.H{
		"status": "error",
		"errors": validationErrors,
	})
}
