//go:build integration

package integration

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/jrjohn/arcana-cloud-go/api/proto/pb"
	grpcclient "github.com/jrjohn/arcana-cloud-go/internal/grpc/client"
	"github.com/jrjohn/arcana-cloud-go/internal/testutil"
)

// ========================================
// Layered gRPC Integration Tests
// ========================================

func TestIntegration_Layered_gRPC_RepositoryLayer_UserService(t *testing.T) {
	testutil.SkipIfShort(t)

	host := getEnv("TEST_REPOSITORY_GRPC_HOST", "localhost")
	port := getEnvInt("TEST_REPOSITORY_GRPC_PORT", 9090)

	// Skip if repository layer is not available
	if host == "" {
		t.Skip("TEST_REPOSITORY_GRPC_HOST not set, skipping layered test")
	}

	logger := zap.NewNop()
	config := grpcclient.DefaultClientConfig(host, port)
	client, err := grpcclient.NewGRPCClient(config, logger)
	require.NoError(t, err)
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	userClient := client.UserServiceClient()

	// Test 1: Create a user
	t.Run("CreateUser", func(t *testing.T) {
		uniqueUsername := fmt.Sprintf("testuser_%d", time.Now().UnixNano())
		uniqueEmail := fmt.Sprintf("test_%d@example.com", time.Now().UnixNano())

		createReq := &pb.CreateUserRequest{
			Username: uniqueUsername,
			Email:    uniqueEmail,
			Password: "hashedpassword123",
		}

		resp, err := userClient.CreateUser(ctx, createReq)
		require.NoError(t, err)
		assert.NotZero(t, resp.Id)
		assert.Equal(t, uniqueUsername, resp.Username)
		assert.Equal(t, uniqueEmail, resp.Email)
		assert.True(t, resp.IsActive)
	})

	// Test 2: Get user by username
	t.Run("GetUserByUsername", func(t *testing.T) {
		uniqueUsername := fmt.Sprintf("getbyname_%d", time.Now().UnixNano())
		uniqueEmail := fmt.Sprintf("getbyname_%d@example.com", time.Now().UnixNano())

		// First create
		createResp, err := userClient.CreateUser(ctx, &pb.CreateUserRequest{
			Username: uniqueUsername,
			Email:    uniqueEmail,
			Password: "pass",
		})
		require.NoError(t, err)

		// Then get by username
		getResp, err := userClient.GetUserByUsername(ctx, &pb.GetUserByUsernameRequest{
			Username: uniqueUsername,
		})
		require.NoError(t, err)
		assert.Equal(t, createResp.Id, getResp.Id)
		assert.Equal(t, uniqueUsername, getResp.Username)
	})

	// Test 3: Exists by username
	t.Run("ExistsByUsername", func(t *testing.T) {
		uniqueUsername := fmt.Sprintf("exists_%d", time.Now().UnixNano())
		uniqueEmail := fmt.Sprintf("exists_%d@example.com", time.Now().UnixNano())

		// Should not exist initially
		existsResp, err := userClient.ExistsByUsername(ctx, &pb.ExistsByUsernameRequest{
			Username: uniqueUsername,
		})
		require.NoError(t, err)
		assert.False(t, existsResp.Value)

		// Create user
		_, err = userClient.CreateUser(ctx, &pb.CreateUserRequest{
			Username: uniqueUsername,
			Email:    uniqueEmail,
			Password: "pass",
		})
		require.NoError(t, err)

		// Should exist now
		existsResp, err = userClient.ExistsByUsername(ctx, &pb.ExistsByUsernameRequest{
			Username: uniqueUsername,
		})
		require.NoError(t, err)
		assert.True(t, existsResp.Value)
	})

	// Test 4: List users with pagination
	t.Run("ListUsers", func(t *testing.T) {
		listResp, err := userClient.ListUsers(ctx, &pb.ListUsersRequest{
			Page: 1,
			Size: 10,
		})
		require.NoError(t, err)
		assert.NotNil(t, listResp.PageInfo)
		assert.GreaterOrEqual(t, len(listResp.Users), 0)
	})
}

func TestIntegration_Layered_gRPC_ServiceLayer_AuthService(t *testing.T) {
	testutil.SkipIfShort(t)

	host := getEnv("TEST_SERVICE_GRPC_HOST", "")
	port := getEnvInt("TEST_SERVICE_GRPC_PORT", 9091)

	if host == "" {
		t.Skip("TEST_SERVICE_GRPC_HOST not set, skipping layered test")
	}

	logger := zap.NewNop()
	config := grpcclient.DefaultClientConfig(host, port)
	client, err := grpcclient.NewGRPCClient(config, logger)
	require.NoError(t, err)
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	authClient := client.AuthServiceClient()

	t.Run("Register", func(t *testing.T) {
		uniqueUsername := fmt.Sprintf("reguser_%d", time.Now().UnixNano())
		uniqueEmail := fmt.Sprintf("reg_%d@example.com", time.Now().UnixNano())

		resp, err := authClient.Register(ctx, &pb.RegisterRequest{
			Username: uniqueUsername,
			Email:    uniqueEmail,
			Password: "SecurePass123!",
		})
		require.NoError(t, err)
		assert.NotEmpty(t, resp.AccessToken)
		assert.NotEmpty(t, resp.RefreshToken)
		assert.Equal(t, "Bearer", resp.TokenType)
		assert.NotNil(t, resp.User)
		assert.Equal(t, uniqueUsername, resp.User.Username)
	})

	t.Run("Login", func(t *testing.T) {
		uniqueUsername := fmt.Sprintf("loginuser_%d", time.Now().UnixNano())
		uniqueEmail := fmt.Sprintf("login_%d@example.com", time.Now().UnixNano())
		password := "SecurePass123!"

		// Register first
		_, err := authClient.Register(ctx, &pb.RegisterRequest{
			Username: uniqueUsername,
			Email:    uniqueEmail,
			Password: password,
		})
		require.NoError(t, err)

		// Then login
		loginResp, err := authClient.Login(ctx, &pb.LoginRequest{
			UsernameOrEmail: uniqueUsername,
			Password:        password,
		})
		require.NoError(t, err)
		assert.NotEmpty(t, loginResp.AccessToken)
		assert.NotEmpty(t, loginResp.RefreshToken)
	})

	t.Run("ValidateToken", func(t *testing.T) {
		uniqueUsername := fmt.Sprintf("validateuser_%d", time.Now().UnixNano())
		uniqueEmail := fmt.Sprintf("validate_%d@example.com", time.Now().UnixNano())

		// Register to get a token
		regResp, err := authClient.Register(ctx, &pb.RegisterRequest{
			Username: uniqueUsername,
			Email:    uniqueEmail,
			Password: "SecurePass123!",
		})
		require.NoError(t, err)

		// Validate the token
		validateResp, err := authClient.ValidateToken(ctx, &pb.ValidateTokenRequest{
			Token: regResp.AccessToken,
		})
		require.NoError(t, err)
		assert.True(t, validateResp.Valid)
		assert.Equal(t, uniqueUsername, validateResp.Username)
	})
}

func TestIntegration_Layered_gRPC_FullFlow(t *testing.T) {
	testutil.SkipIfShort(t)

	repoHost := getEnv("TEST_REPOSITORY_GRPC_HOST", "")
	svcHost := getEnv("TEST_SERVICE_GRPC_HOST", "")

	if repoHost == "" || svcHost == "" {
		t.Skip("Layered hosts not set, skipping full flow test")
	}

	logger := zap.NewNop()

	// Connect to both layers
	repoConfig := grpcclient.DefaultClientConfig(repoHost, getEnvInt("TEST_REPOSITORY_GRPC_PORT", 9090))
	repoClient, err := grpcclient.NewGRPCClient(repoConfig, logger)
	require.NoError(t, err)
	defer repoClient.Close()

	svcConfig := grpcclient.DefaultClientConfig(svcHost, getEnvInt("TEST_SERVICE_GRPC_PORT", 9091))
	svcClient, err := grpcclient.NewGRPCClient(svcConfig, logger)
	require.NoError(t, err)
	defer svcClient.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Run("EndToEndLayeredCommunication", func(t *testing.T) {
		uniqueUsername := fmt.Sprintf("e2e_%d", time.Now().UnixNano())
		uniqueEmail := fmt.Sprintf("e2e_%d@example.com", time.Now().UnixNano())

		// 1. Register via Service Layer (which calls Repository Layer)
		authResp, err := svcClient.AuthServiceClient().Register(ctx, &pb.RegisterRequest{
			Username: uniqueUsername,
			Email:    uniqueEmail,
			Password: "SecurePass123!",
		})
		require.NoError(t, err)
		assert.NotEmpty(t, authResp.AccessToken)

		// 2. Verify user exists via Repository Layer directly
		existsResp, err := repoClient.UserServiceClient().ExistsByUsername(ctx, &pb.ExistsByUsernameRequest{
			Username: uniqueUsername,
		})
		require.NoError(t, err)
		assert.True(t, existsResp.Value, "User should exist in repository layer")

		// 3. Get user details via Repository Layer
		userResp, err := repoClient.UserServiceClient().GetUserByUsername(ctx, &pb.GetUserByUsernameRequest{
			Username: uniqueUsername,
		})
		require.NoError(t, err)
		assert.Equal(t, uniqueUsername, userResp.Username)
		assert.Equal(t, uniqueEmail, userResp.Email)

		// 4. Validate token via Service Layer
		validateResp, err := svcClient.AuthServiceClient().ValidateToken(ctx, &pb.ValidateTokenRequest{
			Token: authResp.AccessToken,
		})
		require.NoError(t, err)
		assert.True(t, validateResp.Valid)
		assert.Equal(t, uniqueUsername, validateResp.Username)
	})
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		var i int
		fmt.Sscanf(value, "%d", &i)
		return i
	}
	return defaultValue
}
