package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
)

// CORSConfig holds CORS configuration
type CORSConfig struct {
	AllowOrigins     []string
	AllowMethods     []string
	AllowHeaders     []string
	ExposeHeaders    []string
	AllowCredentials bool
	MaxAge           time.Duration
}

// DefaultCORSConfig returns the default CORS configuration
func DefaultCORSConfig() CORSConfig {
	return CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{
			"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS", "HEAD",
		},
		AllowHeaders: []string{
			"Origin", "Content-Type", "Accept", "Authorization",
			"X-Requested-With", "X-Request-ID",
		},
		ExposeHeaders: []string{
			"Content-Length", "X-Request-ID",
		},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}
}

// resolveAllowedOrigin returns the origin to reflect, or "" if origin is not allowed
func resolveAllowedOrigin(origin string, allowOrigins []string) string {
	for _, o := range allowOrigins {
		if o == "*" || o == origin {
			return origin
		}
	}
	if len(allowOrigins) > 0 && allowOrigins[0] == "*" {
		return "*"
	}
	return ""
}

// applyPreflightHeaders sets the CORS preflight response headers
func applyPreflightHeaders(c *gin.Context, cfg CORSConfig) {
	c.Header("Access-Control-Allow-Methods", joinStrings(cfg.AllowMethods))
	c.Header("Access-Control-Allow-Headers", joinStrings(cfg.AllowHeaders))
	c.Header("Access-Control-Max-Age", formatMaxAge(cfg.MaxAge))
	c.AbortWithStatus(204)
}

// CORS returns a CORS middleware with the given configuration
func CORS(config CORSConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		allowOrigin := resolveAllowedOrigin(origin, config.AllowOrigins)

		if allowOrigin != "" {
			c.Header("Access-Control-Allow-Origin", allowOrigin)
		}
		if config.AllowCredentials {
			c.Header("Access-Control-Allow-Credentials", "true")
		}

		if c.Request.Method == "OPTIONS" {
			applyPreflightHeaders(c, config)
			return
		}

		if len(config.ExposeHeaders) > 0 {
			c.Header("Access-Control-Expose-Headers", joinStrings(config.ExposeHeaders))
		}

		c.Next()
	}
}

func joinStrings(strs []string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += ", " + strs[i]
	}
	return result
}

func formatMaxAge(d time.Duration) string {
	return string(rune(int(d.Seconds())))
}
