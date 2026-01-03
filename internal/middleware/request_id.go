package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	// RequestIDHeader is the header name for request ID
	RequestIDHeader = "X-Request-ID"
	// RequestIDKey is the context key for request ID
	RequestIDKey = "request_id"
)

// RequestID adds a unique request ID to each request
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if request ID is provided in header
		requestID := c.GetHeader(RequestIDHeader)
		if requestID == "" {
			requestID = uuid.New().String()
		}

		// Set in context and response header
		c.Set(RequestIDKey, requestID)
		c.Header(RequestIDHeader, requestID)

		c.Next()
	}
}

// GetRequestID retrieves the request ID from context
func GetRequestID(c *gin.Context) string {
	if id, exists := c.Get(RequestIDKey); exists {
		if str, ok := id.(string); ok {
			return str
		}
	}
	return ""
}
