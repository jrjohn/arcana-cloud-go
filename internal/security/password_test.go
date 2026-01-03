package security

import (
	"strings"
	"testing"
)

func TestNewPasswordHasher(t *testing.T) {
	hasher := NewPasswordHasher()
	if hasher == nil {
		t.Fatal("NewPasswordHasher() returned nil")
	}
	if hasher.cost != DefaultCost {
		t.Errorf("cost = %v, want %v", hasher.cost, DefaultCost)
	}
}

func TestPasswordHasher_Hash(t *testing.T) {
	hasher := NewPasswordHasher()

	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{
			name:     "simple password",
			password: "password123",
			wantErr:  false,
		},
		{
			name:     "complex password",
			password: "C0mpl3x!P@ssw0rd#2024",
			wantErr:  false,
		},
		{
			name:     "empty password",
			password: "",
			wantErr:  false, // bcrypt handles empty strings
		},
		{
			name:     "unicode password",
			password: "密码パスワード🔐",
			wantErr:  false,
		},
		{
			name:     "very long password",
			password: strings.Repeat("a", 72), // bcrypt max is 72 bytes
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := hasher.Hash(tt.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("Hash() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && hash == "" {
				t.Error("Hash() returned empty string")
			}
			if !tt.wantErr && hash == tt.password {
				t.Error("Hash() returned unhashed password")
			}
		})
	}
}

func TestPasswordHasher_Hash_UniqueSalts(t *testing.T) {
	hasher := NewPasswordHasher()
	password := "samepassword"

	// Hash the same password multiple times
	hashes := make([]string, 3)
	for i := 0; i < 3; i++ {
		hash, err := hasher.Hash(password)
		if err != nil {
			t.Fatalf("Hash() error = %v", err)
		}
		hashes[i] = hash
	}

	// All hashes should be different (due to unique salts)
	for i := 0; i < len(hashes); i++ {
		for j := i + 1; j < len(hashes); j++ {
			if hashes[i] == hashes[j] {
				t.Errorf("Hash %d and %d are identical, should be unique", i, j)
			}
		}
	}
}

func TestPasswordHasher_Verify(t *testing.T) {
	hasher := NewPasswordHasher()

	tests := []struct {
		name     string
		password string
		verify   string
		expected bool
	}{
		{
			name:     "correct password",
			password: "password123",
			verify:   "password123",
			expected: true,
		},
		{
			name:     "wrong password",
			password: "password123",
			verify:   "wrongpassword",
			expected: false,
		},
		{
			name:     "case sensitive",
			password: "Password",
			verify:   "password",
			expected: false,
		},
		{
			name:     "empty password match",
			password: "",
			verify:   "",
			expected: true,
		},
		{
			name:     "empty password no match",
			password: "password",
			verify:   "",
			expected: false,
		},
		{
			name:     "unicode password",
			password: "密码123",
			verify:   "密码123",
			expected: true,
		},
		{
			name:     "unicode password mismatch",
			password: "密码123",
			verify:   "密码124",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := hasher.Hash(tt.password)
			if err != nil {
				t.Fatalf("Hash() error = %v", err)
			}

			result := hasher.Verify(tt.verify, hash)
			if result != tt.expected {
				t.Errorf("Verify() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestPasswordHasher_Verify_InvalidHash(t *testing.T) {
	hasher := NewPasswordHasher()

	tests := []struct {
		name     string
		password string
		hash     string
	}{
		{
			name:     "invalid hash format",
			password: "password",
			hash:     "invalid-hash",
		},
		{
			name:     "empty hash",
			password: "password",
			hash:     "",
		},
		{
			name:     "truncated hash",
			password: "password",
			hash:     "$2a$12$truncated",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasher.Verify(tt.password, tt.hash)
			if result {
				t.Error("Verify() should return false for invalid hash")
			}
		})
	}
}

func TestDefaultCost(t *testing.T) {
	if DefaultCost != 12 {
		t.Errorf("DefaultCost = %v, want 12", DefaultCost)
	}
}

func TestPasswordHasher_HashFormat(t *testing.T) {
	hasher := NewPasswordHasher()
	password := "testpassword"

	hash, err := hasher.Hash(password)
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}

	// bcrypt hash format: $2a$<cost>$<22 char salt><31 char hash>
	if !strings.HasPrefix(hash, "$2a$") && !strings.HasPrefix(hash, "$2b$") {
		t.Errorf("Hash format should start with $2a$ or $2b$, got %s", hash[:4])
	}

	// Total length should be 60 characters for bcrypt
	if len(hash) != 60 {
		t.Errorf("Hash length = %d, want 60", len(hash))
	}
}

func TestPasswordHasher_CrossVerification(t *testing.T) {
	hasher1 := NewPasswordHasher()
	hasher2 := NewPasswordHasher()
	password := "testpassword"

	// Hash with first hasher
	hash, err := hasher1.Hash(password)
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}

	// Verify with second hasher (should work since bcrypt is deterministic for verification)
	if !hasher2.Verify(password, hash) {
		t.Error("Cross-hasher verification should work")
	}
}

func TestPasswordHasher_LongPassword(t *testing.T) {
	hasher := NewPasswordHasher()

	// Go's bcrypt library enforces a strict 72-byte limit
	// Passwords at or under 72 bytes work fine
	password72 := strings.Repeat("a", 72)

	hash72, err := hasher.Hash(password72)
	if err != nil {
		t.Fatalf("Hash 72 chars error = %v", err)
	}

	// Verify that 72-char password matches its hash
	if !hasher.Verify(password72, hash72) {
		t.Error("72-char password should verify")
	}

	// Passwords over 72 bytes should return an error
	password73 := strings.Repeat("a", 73)
	_, err = hasher.Hash(password73)
	if err == nil {
		t.Error("Hash should return error for password > 72 bytes")
	}
}

// Benchmarks
func BenchmarkPasswordHasher_Hash(b *testing.B) {
	hasher := NewPasswordHasher()
	password := "benchmarkpassword"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hasher.Hash(password)
	}
}

func BenchmarkPasswordHasher_Verify_Success(b *testing.B) {
	hasher := NewPasswordHasher()
	password := "benchmarkpassword"
	hash, _ := hasher.Hash(password)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hasher.Verify(password, hash)
	}
}

func BenchmarkPasswordHasher_Verify_Failure(b *testing.B) {
	hasher := NewPasswordHasher()
	password := "benchmarkpassword"
	hash, _ := hasher.Hash(password)
	wrongPassword := "wrongpassword"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hasher.Verify(wrongPassword, hash)
	}
}
