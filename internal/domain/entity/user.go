package entity

import (
	"time"

	"gorm.io/gorm"
)

// UserRole represents user roles in the system
type UserRole string

const (
	RoleUser  UserRole = "USER"
	RoleAdmin UserRole = "ADMIN"
)

// User represents a user entity in the system
type User struct {
	ID         uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	Username   string         `gorm:"uniqueIndex;size:50;not null" json:"username"`
	Email      string         `gorm:"uniqueIndex;size:100;not null" json:"email"`
	Password   string         `gorm:"not null" json:"-"`
	FirstName  string         `gorm:"column:first_name;size:50" json:"first_name,omitempty"`
	LastName   string         `gorm:"column:last_name;size:50" json:"last_name,omitempty"`
	Role       UserRole       `gorm:"size:20;not null;default:USER" json:"role"`
	IsActive   bool           `gorm:"column:is_active;default:true" json:"is_active"`
	IsVerified bool           `gorm:"column:is_verified;default:false" json:"is_verified"`
	CreatedAt  time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt  time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName specifies the table name for User
func (User) TableName() string {
	return "users"
}

// RefreshToken represents a refresh token for JWT authentication
type RefreshToken struct {
	ID        uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID    uint           `gorm:"index;not null" json:"user_id"`
	Token     string         `gorm:"uniqueIndex;size:500;not null" json:"token"`
	ExpiresAt time.Time      `gorm:"not null" json:"expires_at"`
	Revoked   bool           `gorm:"default:false" json:"revoked"`
	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	User User `gorm:"foreignKey:UserID" json:"-"`
}

// TableName specifies the table name for RefreshToken
func (RefreshToken) TableName() string {
	return "refresh_tokens"
}

// IsExpired checks if the refresh token is expired
func (rt *RefreshToken) IsExpired() bool {
	return time.Now().After(rt.ExpiresAt)
}

// IsValid checks if the refresh token is valid
func (rt *RefreshToken) IsValid() bool {
	return !rt.Revoked && !rt.IsExpired()
}
