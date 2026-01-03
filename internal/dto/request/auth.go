package request

// RegisterRequest represents a user registration request
type RegisterRequest struct {
	Username  string `json:"username" binding:"required,min=3,max=50"`
	Email     string `json:"email" binding:"required,email,max=100"`
	Password  string `json:"password" binding:"required,min=8,max=72"`
	FirstName string `json:"first_name,omitempty" binding:"max=50"`
	LastName  string `json:"last_name,omitempty" binding:"max=50"`
}

// LoginRequest represents a user login request
type LoginRequest struct {
	UsernameOrEmail string `json:"username_or_email" binding:"required"`
	Password        string `json:"password" binding:"required"`
}

// RefreshTokenRequest represents a token refresh request
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// ChangePasswordRequest represents a password change request
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=8,max=72"`
}

// UpdateProfileRequest represents a profile update request
type UpdateProfileRequest struct {
	FirstName string `json:"first_name,omitempty" binding:"max=50"`
	LastName  string `json:"last_name,omitempty" binding:"max=50"`
	Email     string `json:"email,omitempty" binding:"omitempty,email,max=100"`
}
