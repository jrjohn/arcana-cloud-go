package graphql

import (
	"context"
	"errors"
	"strconv"

	"github.com/graphql-go/graphql"

	"github.com/jrjohn/arcana-cloud-go/internal/domain/service"
	"github.com/jrjohn/arcana-cloud-go/internal/dto/request"
	"github.com/jrjohn/arcana-cloud-go/internal/dto/response"
	pluginmgr "github.com/jrjohn/arcana-cloud-go/internal/plugin/manager"
)

// ContextKey type for context keys
type ContextKey string

const (
	ContextKeyUserID   ContextKey = "userID"
	ContextKeyUsername ContextKey = "username"
	ContextKeyToken    ContextKey = "token"
)

// Resolver handles GraphQL resolvers
type Resolver struct {
	authService   service.AuthService
	userService   service.UserService
	pluginManager *pluginmgr.Manager
}

// NewResolver creates a new resolver
func NewResolver(
	authService service.AuthService,
	userService service.UserService,
	pluginManager *pluginmgr.Manager,
) *Resolver {
	return &Resolver{
		authService:   authService,
		userService:   userService,
		pluginManager: pluginManager,
	}
}

// Auth Resolvers

// Register handles user registration
func (r *Resolver) Register(p graphql.ResolveParams) (interface{}, error) {
	input, ok := p.Args["input"].(map[string]interface{})
	if !ok {
		return nil, errors.New("invalid input")
	}

	req := &request.RegisterRequest{
		Username: input["username"].(string),
		Email:    input["email"].(string),
		Password: input["password"].(string),
	}

	if firstName, ok := input["firstName"].(string); ok {
		req.FirstName = firstName
	}
	if lastName, ok := input["lastName"].(string); ok {
		req.LastName = lastName
	}

	return r.authService.Register(p.Context, req)
}

// Login handles user login
func (r *Resolver) Login(p graphql.ResolveParams) (interface{}, error) {
	input, ok := p.Args["input"].(map[string]interface{})
	if !ok {
		return nil, errors.New("invalid input")
	}

	req := &request.LoginRequest{
		UsernameOrEmail: input["usernameOrEmail"].(string),
		Password:        input["password"].(string),
	}

	return r.authService.Login(p.Context, req)
}

// RefreshToken handles token refresh
func (r *Resolver) RefreshToken(p graphql.ResolveParams) (interface{}, error) {
	refreshToken, ok := p.Args["refreshToken"].(string)
	if !ok {
		return nil, errors.New("invalid refresh token")
	}

	req := &request.RefreshTokenRequest{
		RefreshToken: refreshToken,
	}

	return r.authService.RefreshToken(p.Context, req)
}

// Logout handles user logout
func (r *Resolver) Logout(p graphql.ResolveParams) (interface{}, error) {
	token := getTokenFromContext(p.Context)
	if token == "" {
		return false, errors.New("not authenticated")
	}

	err := r.authService.Logout(p.Context, token)
	return err == nil, err
}

// User Resolvers

// Me returns the current authenticated user
func (r *Resolver) Me(p graphql.ResolveParams) (interface{}, error) {
	userID := getUserIDFromContext(p.Context)
	if userID == 0 {
		return nil, errors.New("not authenticated")
	}

	return r.userService.GetByID(p.Context, userID)
}

// User returns a user by ID
func (r *Resolver) User(p graphql.ResolveParams) (interface{}, error) {
	id, ok := p.Args["id"].(string)
	if !ok {
		return nil, errors.New("invalid user ID")
	}

	userID, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		return nil, errors.New("invalid user ID format")
	}

	return r.userService.GetByID(p.Context, uint(userID))
}

// Users returns a paginated list of users
func (r *Resolver) Users(p graphql.ResolveParams) (interface{}, error) {
	page := 1
	size := 10

	if pageArg, ok := p.Args["page"].(int); ok {
		page = pageArg
	}
	if sizeArg, ok := p.Args["size"].(int); ok {
		size = sizeArg
	}

	result, err := r.userService.List(p.Context, page, size)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"items":    result.Items,
		"pageInfo": result.PageInfo,
	}, nil
}

// UpdateProfile updates the current user's profile
func (r *Resolver) UpdateProfile(p graphql.ResolveParams) (interface{}, error) {
	userID := getUserIDFromContext(p.Context)
	if userID == 0 {
		return nil, errors.New("not authenticated")
	}

	input, ok := p.Args["input"].(map[string]interface{})
	if !ok {
		return nil, errors.New("invalid input")
	}

	req := &request.UpdateProfileRequest{}
	if firstName, ok := input["firstName"].(string); ok {
		req.FirstName = firstName
	}
	if lastName, ok := input["lastName"].(string); ok {
		req.LastName = lastName
	}
	if email, ok := input["email"].(string); ok {
		req.Email = email
	}

	return r.userService.Update(p.Context, userID, req)
}

// ChangePassword changes the current user's password
func (r *Resolver) ChangePassword(p graphql.ResolveParams) (interface{}, error) {
	userID := getUserIDFromContext(p.Context)
	if userID == 0 {
		return false, errors.New("not authenticated")
	}

	currentPassword, _ := p.Args["currentPassword"].(string)
	newPassword, _ := p.Args["newPassword"].(string)

	req := &request.ChangePasswordRequest{
		OldPassword: currentPassword,
		NewPassword: newPassword,
	}

	err := r.userService.ChangePassword(p.Context, userID, req)
	return err == nil, err
}

// Plugin Resolvers

// Plugins returns all plugins
func (r *Resolver) Plugins(p graphql.ResolveParams) (interface{}, error) {
	if r.pluginManager == nil {
		return []interface{}{}, nil
	}

	plugins := r.pluginManager.ListPlugins()
	result := make([]map[string]interface{}, len(plugins))

	for i, plugin := range plugins {
		result[i] = map[string]interface{}{
			"id":          plugin.Info.Key,
			"key":         plugin.Info.Key,
			"name":        plugin.Info.Name,
			"version":     plugin.Info.Version,
			"description": plugin.Info.Description,
			"state":       string(plugin.State),
			"enabled":     plugin.State == pluginmgr.StateStarted,
		}
	}

	return result, nil
}

// Plugin returns a plugin by key
func (r *Resolver) Plugin(p graphql.ResolveParams) (interface{}, error) {
	if r.pluginManager == nil {
		return nil, errors.New("plugin system not enabled")
	}

	key, ok := p.Args["key"].(string)
	if !ok {
		return nil, errors.New("invalid plugin key")
	}

	plugin, exists := r.pluginManager.GetPlugin(key)
	if !exists {
		return nil, errors.New("plugin not found")
	}

	return map[string]interface{}{
		"id":          plugin.Info.Key,
		"key":         plugin.Info.Key,
		"name":        plugin.Info.Name,
		"version":     plugin.Info.Version,
		"description": plugin.Info.Description,
		"state":       string(plugin.State),
		"enabled":     plugin.State == pluginmgr.StateStarted,
	}, nil
}

// EnablePlugin enables a plugin
func (r *Resolver) EnablePlugin(p graphql.ResolveParams) (interface{}, error) {
	if r.pluginManager == nil {
		return nil, errors.New("plugin system not enabled")
	}

	key, ok := p.Args["key"].(string)
	if !ok {
		return nil, errors.New("invalid plugin key")
	}

	if err := r.pluginManager.StartPlugin(p.Context, key, nil); err != nil {
		return nil, err
	}

	return r.Plugin(p)
}

// DisablePlugin disables a plugin
func (r *Resolver) DisablePlugin(p graphql.ResolveParams) (interface{}, error) {
	if r.pluginManager == nil {
		return nil, errors.New("plugin system not enabled")
	}

	key, ok := p.Args["key"].(string)
	if !ok {
		return nil, errors.New("invalid plugin key")
	}

	if err := r.pluginManager.StopPlugin(p.Context, key); err != nil {
		return nil, err
	}

	return r.Plugin(p)
}

// Helper functions

func getUserIDFromContext(ctx context.Context) uint {
	if userID, ok := ctx.Value(ContextKeyUserID).(uint); ok {
		return userID
	}
	return 0
}

func getTokenFromContext(ctx context.Context) string {
	if token, ok := ctx.Value(ContextKeyToken).(string); ok {
		return token
	}
	return ""
}

// ToUserResponse converts a user response for GraphQL
func ToUserResponse(user *response.UserResponse) map[string]interface{} {
	if user == nil {
		return nil
	}
	return map[string]interface{}{
		"id":         user.ID,
		"username":   user.Username,
		"email":      user.Email,
		"firstName":  user.FirstName,
		"lastName":   user.LastName,
		"role":       user.Role,
		"isActive":   user.IsActive,
		"isVerified": user.IsVerified,
		"createdAt":  user.CreatedAt,
		"updatedAt":  user.UpdatedAt,
	}
}
