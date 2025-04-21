package validation

import (
	"errors"
	"fmt"
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

// getJSONFieldName gets the JSON field name from the struct field
func getJSONFieldName(structType reflect.Type, fieldName string) string {
	field, found := structType.FieldByName(fieldName)
	if !found {
		return fieldName
	}

	jsonTag := field.Tag.Get("json")
	if jsonTag == "" {
		return fieldName
	}

	// Handle cases where json tag might have options like `json:"field,omitempty"`
	if commaIdx := strings.Index(jsonTag, ","); commaIdx != -1 {
		return jsonTag[:commaIdx]
	}

	return jsonTag
}

// formatValidationErrors converts validator errors into a more readable format
func formatValidationErrors(c *gin.Context, err error, inputType interface{}) []ValidationError {
	var validationErrors []ValidationError
	structType := reflect.TypeOf(inputType)

	// Try to convert to validator.ValidationErrors
	var ve validator.ValidationErrors
	if errors.As(err, &ve) {
		for _, e := range ve {

			// fieldName := e.Field()
			fieldName := getJSONFieldName(structType, e.Field())
			location := getFieldLocation(c, fieldName)

			// Create a custom error message based on the validation tag
			var errorMsg string
			switch e.Tag() {
			case "required":
				errorMsg = "This field is required"
			case "url":
				errorMsg = "Must be a valid URL"
			case "alphanum":
				errorMsg = "Must contain only alphanumeric characters"
			case "min":
				errorMsg = fmt.Sprintf("Must be at least %s characters long", e.Param())
			case "max":
				errorMsg = fmt.Sprintf("Must be at most %s characters long", e.Param())
			case "gt":
				errorMsg = fmt.Sprintf("Must be greater than %s", e.Param())
			case "omitempty":
				errorMsg = "Invalid value"
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
func HandleValidationErrors(c *gin.Context, err error, inputType interface{}) {
	validationErrors := formatValidationErrors(c, err, inputType)
	c.JSON(http.StatusBadRequest, gin.H{
		"status":  "error",
		"message": "validation failed",
		"data":    validationErrors,
	})
}
