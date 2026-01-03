package entity

import (
	"testing"
	"time"
)

func TestUserRole_Constants(t *testing.T) {
	tests := []struct {
		name     string
		role     UserRole
		expected string
	}{
		{"USER role", RoleUser, "USER"},
		{"ADMIN role", RoleAdmin, "ADMIN"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.role) != tt.expected {
				t.Errorf("UserRole = %v, want %v", tt.role, tt.expected)
			}
		})
	}
}

func TestUser_TableName(t *testing.T) {
	user := User{}
	if tableName := user.TableName(); tableName != "users" {
		t.Errorf("User.TableName() = %v, want users", tableName)
	}
}

func TestUser_Struct(t *testing.T) {
	now := time.Now()
	user := User{
		ID:         1,
		Username:   "testuser",
		Email:      "test@example.com",
		Password:   "hashedpassword",
		FirstName:  "Test",
		LastName:   "User",
		Role:       RoleUser,
		IsActive:   true,
		IsVerified: false,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if user.ID != 1 {
		t.Errorf("User.ID = %v, want 1", user.ID)
	}
	if user.Username != "testuser" {
		t.Errorf("User.Username = %v, want testuser", user.Username)
	}
	if user.Email != "test@example.com" {
		t.Errorf("User.Email = %v, want test@example.com", user.Email)
	}
	if user.Password != "hashedpassword" {
		t.Errorf("User.Password = %v, want hashedpassword", user.Password)
	}
	if user.FirstName != "Test" {
		t.Errorf("User.FirstName = %v, want Test", user.FirstName)
	}
	if user.LastName != "User" {
		t.Errorf("User.LastName = %v, want User", user.LastName)
	}
	if user.Role != RoleUser {
		t.Errorf("User.Role = %v, want %v", user.Role, RoleUser)
	}
	if !user.IsActive {
		t.Error("User.IsActive should be true")
	}
	if user.IsVerified {
		t.Error("User.IsVerified should be false")
	}
}

func TestRefreshToken_TableName(t *testing.T) {
	rt := RefreshToken{}
	if tableName := rt.TableName(); tableName != "refresh_tokens" {
		t.Errorf("RefreshToken.TableName() = %v, want refresh_tokens", tableName)
	}
}

func TestRefreshToken_IsExpired(t *testing.T) {
	tests := []struct {
		name      string
		expiresAt time.Time
		expected  bool
	}{
		{
			name:      "not expired",
			expiresAt: time.Now().Add(time.Hour),
			expected:  false,
		},
		{
			name:      "expired",
			expiresAt: time.Now().Add(-time.Hour),
			expected:  true,
		},
		{
			name:      "expires now",
			expiresAt: time.Now().Add(-time.Millisecond),
			expected:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rt := &RefreshToken{ExpiresAt: tt.expiresAt}
			if got := rt.IsExpired(); got != tt.expected {
				t.Errorf("RefreshToken.IsExpired() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestRefreshToken_IsValid(t *testing.T) {
	tests := []struct {
		name      string
		expiresAt time.Time
		revoked   bool
		expected  bool
	}{
		{
			name:      "valid token",
			expiresAt: time.Now().Add(time.Hour),
			revoked:   false,
			expected:  true,
		},
		{
			name:      "revoked token",
			expiresAt: time.Now().Add(time.Hour),
			revoked:   true,
			expected:  false,
		},
		{
			name:      "expired token",
			expiresAt: time.Now().Add(-time.Hour),
			revoked:   false,
			expected:  false,
		},
		{
			name:      "expired and revoked",
			expiresAt: time.Now().Add(-time.Hour),
			revoked:   true,
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rt := &RefreshToken{
				ExpiresAt: tt.expiresAt,
				Revoked:   tt.revoked,
			}
			if got := rt.IsValid(); got != tt.expected {
				t.Errorf("RefreshToken.IsValid() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestRefreshToken_Struct(t *testing.T) {
	now := time.Now()
	expiresAt := now.Add(24 * time.Hour)

	user := User{ID: 1, Username: "testuser"}
	rt := RefreshToken{
		ID:        1,
		UserID:    1,
		Token:     "test-refresh-token",
		ExpiresAt: expiresAt,
		Revoked:   false,
		CreatedAt: now,
		User:      user,
	}

	if rt.ID != 1 {
		t.Errorf("RefreshToken.ID = %v, want 1", rt.ID)
	}
	if rt.UserID != 1 {
		t.Errorf("RefreshToken.UserID = %v, want 1", rt.UserID)
	}
	if rt.Token != "test-refresh-token" {
		t.Errorf("RefreshToken.Token = %v, want test-refresh-token", rt.Token)
	}
	if !rt.ExpiresAt.Equal(expiresAt) {
		t.Errorf("RefreshToken.ExpiresAt = %v, want %v", rt.ExpiresAt, expiresAt)
	}
	if rt.Revoked {
		t.Error("RefreshToken.Revoked should be false")
	}
	if rt.User.ID != 1 {
		t.Errorf("RefreshToken.User.ID = %v, want 1", rt.User.ID)
	}
}

func TestUser_AdminRole(t *testing.T) {
	admin := User{
		ID:       1,
		Username: "admin",
		Role:     RoleAdmin,
	}

	if admin.Role != RoleAdmin {
		t.Errorf("Admin user role = %v, want %v", admin.Role, RoleAdmin)
	}
}

func TestRefreshToken_EdgeCases(t *testing.T) {
	// Test with zero time
	rt := &RefreshToken{ExpiresAt: time.Time{}}
	if !rt.IsExpired() {
		t.Error("Zero time should be expired")
	}
	if rt.IsValid() {
		t.Error("Zero time token should not be valid")
	}

	// Test just expired
	rt2 := &RefreshToken{ExpiresAt: time.Now().Add(-time.Nanosecond)}
	if !rt2.IsExpired() {
		t.Error("Just expired token should be expired")
	}

	// Test about to expire
	rt3 := &RefreshToken{ExpiresAt: time.Now().Add(time.Nanosecond * 100)}
	time.Sleep(time.Millisecond)
	// After sleep it should be expired
	if !rt3.IsExpired() {
		t.Error("Token should be expired after wait")
	}
}

// Benchmarks
func BenchmarkRefreshToken_IsExpired(b *testing.B) {
	rt := &RefreshToken{ExpiresAt: time.Now().Add(time.Hour)}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rt.IsExpired()
	}
}

func BenchmarkRefreshToken_IsValid(b *testing.B) {
	rt := &RefreshToken{
		ExpiresAt: time.Now().Add(time.Hour),
		Revoked:   false,
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rt.IsValid()
	}
}

func BenchmarkUser_TableName(b *testing.B) {
	user := User{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		user.TableName()
	}
}
