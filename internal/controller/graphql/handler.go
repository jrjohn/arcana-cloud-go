package graphql

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/graphql-go/graphql"
	"go.uber.org/zap"

	"github.com/jrjohn/arcana-cloud-go/internal/security"
)

// GraphQLConfig holds GraphQL configuration
type GraphQLConfig struct {
	Enabled         bool   `mapstructure:"enabled"`
	Path            string `mapstructure:"path"`
	PlaygroundPath  string `mapstructure:"playground_path"`
	EnablePlayground bool   `mapstructure:"enable_playground"`
	MaxDepth        int    `mapstructure:"max_depth"`
	MaxComplexity   int    `mapstructure:"max_complexity"`
}

// DefaultGraphQLConfig returns default configuration
func DefaultGraphQLConfig() *GraphQLConfig {
	return &GraphQLConfig{
		Enabled:          false,
		Path:             "/graphql",
		PlaygroundPath:   "/playground",
		EnablePlayground: true,
		MaxDepth:         10,
		MaxComplexity:    100,
	}
}

// Handler handles GraphQL requests
type Handler struct {
	schema      *Schema
	config      *GraphQLConfig
	jwtProvider *security.JWTProvider
	logger      *zap.Logger
}

// NewHandler creates a new GraphQL handler
func NewHandler(
	schema *Schema,
	config *GraphQLConfig,
	jwtProvider *security.JWTProvider,
	logger *zap.Logger,
) *Handler {
	return &Handler{
		schema:      schema,
		config:      config,
		jwtProvider: jwtProvider,
		logger:      logger,
	}
}

// GraphQLRequest represents a GraphQL request
type GraphQLRequest struct {
	Query         string                 `json:"query"`
	OperationName string                 `json:"operationName"`
	Variables     map[string]interface{} `json:"variables"`
}

// RegisterRoutes registers GraphQL routes
func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	router.POST(h.config.Path, h.handleGraphQL)
	router.GET(h.config.Path, h.handleGraphQL) // For query via GET

	if h.config.EnablePlayground {
		router.GET(h.config.PlaygroundPath, h.handlePlayground)
	}
}

// handleGraphQL handles GraphQL requests
func (h *Handler) handleGraphQL(c *gin.Context) {
	var req GraphQLRequest

	// Parse request based on method
	if c.Request.Method == http.MethodPost {
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"errors": []map[string]string{
					{"message": "Invalid request body"},
				},
			})
			return
		}
	} else {
		req.Query = c.Query("query")
		req.OperationName = c.Query("operationName")
		if variables := c.Query("variables"); variables != "" {
			json.Unmarshal([]byte(variables), &req.Variables)
		}
	}

	// Extract authentication context
	ctx := h.extractAuthContext(c)

	// Execute GraphQL query
	result := graphql.Do(graphql.Params{
		Schema:         h.schema.Schema(),
		RequestString:  req.Query,
		OperationName:  req.OperationName,
		VariableValues: req.Variables,
		Context:        ctx,
	})

	// Return response
	c.JSON(http.StatusOK, result)
}

// extractAuthContext extracts authentication info from request headers
func (h *Handler) extractAuthContext(c *gin.Context) context.Context {
	ctx := c.Request.Context()

	// Get Authorization header
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return ctx
	}

	// Parse Bearer token
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return ctx
	}

	token := parts[1]

	// Validate token
	claims, err := h.jwtProvider.ValidateAccessToken(token)
	if err != nil {
		return ctx
	}

	// Add claims to context
	ctx = context.WithValue(ctx, ContextKeyUserID, claims.UserID)
	ctx = context.WithValue(ctx, ContextKeyUsername, claims.Username)
	ctx = context.WithValue(ctx, ContextKeyToken, token)

	return ctx
}

// handlePlayground serves the GraphQL playground
func (h *Handler) handlePlayground(c *gin.Context) {
	html := `<!DOCTYPE html>
<html>
<head>
    <meta charset=utf-8/>
    <meta name="viewport" content="user-scalable=no, initial-scale=1.0, minimum-scale=1.0, maximum-scale=1.0, minimal-ui">
    <title>Arcana Cloud GraphQL Playground</title>
    <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/graphql-playground-react/build/static/css/index.css" />
    <link rel="shortcut icon" href="https://cdn.jsdelivr.net/npm/graphql-playground-react/build/favicon.png" />
    <script src="https://cdn.jsdelivr.net/npm/graphql-playground-react/build/static/js/middleware.js"></script>
</head>
<body>
    <div id="root">
        <style>
            body {
                background-color: rgb(23, 42, 58);
                font-family: Open Sans, sans-serif;
                height: 90vh;
            }
            #root {
                height: 100%;
                width: 100%;
                display: flex;
                align-items: center;
                justify-content: center;
            }
            .loading {
                font-size: 32px;
                font-weight: 200;
                color: rgba(255, 255, 255, .6);
                margin-left: 28px;
            }
            img {
                width: 78px;
                height: 78px;
            }
            .title {
                font-weight: 400;
            }
        </style>
        <img src='https://cdn.jsdelivr.net/npm/graphql-playground-react/build/logo.png' alt=''>
        <div class="loading"> Loading
            <span class="title">Arcana Cloud GraphQL Playground</span>
        </div>
    </div>
    <script>window.addEventListener('load', function (event) {
        GraphQLPlayground.init(document.getElementById('root'), {
            endpoint: '` + h.config.Path + `',
            settings: {
                'request.credentials': 'include',
            },
            tabs: [
                {
                    endpoint: '` + h.config.Path + `',
                    query: '# Welcome to Arcana Cloud GraphQL Playground\n#\n# Example queries:\n#\n# query {\n#   me {\n#     id\n#     username\n#     email\n#   }\n# }\n#\n# mutation {\n#   login(input: {usernameOrEmail: "admin", password: "password"}) {\n#     accessToken\n#     user {\n#       id\n#       username\n#     }\n#   }\n# }\n'
                }
            ]
        })
    })</script>
</body>
</html>`

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, html)
}
