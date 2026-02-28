package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/jrjohn/arcana-cloud-go/api/proto/pb"
	"github.com/jrjohn/arcana-cloud-go/internal/domain/entity"
)

// mockUserRepository for testing
type mockUserRepository struct {
	createFn            func(ctx context.Context, user *entity.User) error
	getByIDFn           func(ctx context.Context, id uint) (*entity.User, error)
	getByUsernameFn     func(ctx context.Context, username string) (*entity.User, error)
	getByEmailFn        func(ctx context.Context, email string) (*entity.User, error)
	getByUsernameOrEmailFn func(ctx context.Context, identifier string) (*entity.User, error)
	updateFn            func(ctx context.Context, user *entity.User) error
	deleteFn            func(ctx context.Context, id uint) error
	listFn              func(ctx context.Context, page, size int) ([]*entity.User, int64, error)
	existsByUsernameFn  func(ctx context.Context, username string) (bool, error)
	existsByEmailFn     func(ctx context.Context, email string) (bool, error)
}

func (m *mockUserRepository) Create(ctx context.Context, user *entity.User) error {
	if m.createFn != nil {
		return m.createFn(ctx, user)
	}
	user.ID = 1
	return nil
}

func (m *mockUserRepository) GetByID(ctx context.Context, id uint) (*entity.User, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return newTestUser(), nil
}

func (m *mockUserRepository) GetByUsername(ctx context.Context, username string) (*entity.User, error) {
	if m.getByUsernameFn != nil {
		return m.getByUsernameFn(ctx, username)
	}
	return newTestUser(), nil
}

func (m *mockUserRepository) GetByEmail(ctx context.Context, email string) (*entity.User, error) {
	if m.getByEmailFn != nil {
		return m.getByEmailFn(ctx, email)
	}
	return newTestUser(), nil
}

func (m *mockUserRepository) GetByUsernameOrEmail(ctx context.Context, identifier string) (*entity.User, error) {
	if m.getByUsernameOrEmailFn != nil {
		return m.getByUsernameOrEmailFn(ctx, identifier)
	}
	return newTestUser(), nil
}

func (m *mockUserRepository) Update(ctx context.Context, user *entity.User) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, user)
	}
	return nil
}

func (m *mockUserRepository) Delete(ctx context.Context, id uint) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return nil
}

func (m *mockUserRepository) List(ctx context.Context, page, size int) ([]*entity.User, int64, error) {
	if m.listFn != nil {
		return m.listFn(ctx, page, size)
	}
	return []*entity.User{newTestUser()}, 1, nil
}

func (m *mockUserRepository) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	if m.existsByUsernameFn != nil {
		return m.existsByUsernameFn(ctx, username)
	}
	return false, nil
}

func (m *mockUserRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	if m.existsByEmailFn != nil {
		return m.existsByEmailFn(ctx, email)
	}
	return false, nil
}

func newTestUser() *entity.User {
	now := time.Now()
	return &entity.User{
		ID:         1,
		Username:   "testuser",
		Email:      "test@example.com",
		FirstName:  "Test",
		LastName:   "User",
		Role:       entity.RoleUser,
		IsActive:   true,
		IsVerified: true,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

func newTestUserRepoServer() *UserServiceServer {
	logger, _ := zap.NewDevelopment()
	repo := &mockUserRepository{}
	return NewUserServiceServer(repo, logger)
}

func TestNewUserRepoServiceServer(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	repo := &mockUserRepository{}
	server := NewUserServiceServer(repo, logger)
	if server == nil {
		t.Error("NewUserServiceServer() returned nil")
	}
}

func TestUserRepoServer_GetUser_Success(t *testing.T) {
	server := newTestUserRepoServer()
	ctx := context.Background()

	req := &pb.GetUserRequest{Id: 1}
	resp, err := server.GetUser(ctx, req)
	if err != nil {
		t.Fatalf("GetUser() error = %v", err)
	}
	if resp == nil {
		t.Fatal("GetUser() returned nil")
	}
	if resp.Username != "testuser" {
		t.Errorf("Username = %v, want testuser", resp.Username)
	}
}

func TestUserRepoServer_GetUser_NilUser(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	repo := &mockUserRepository{
		getByIDFn: func(ctx context.Context, id uint) (*entity.User, error) {
			return nil, nil
		},
	}
	server := NewUserServiceServer(repo, logger)
	ctx := context.Background()

	req := &pb.GetUserRequest{Id: 999}
	_, err := server.GetUser(ctx, req)
	if err == nil {
		t.Error("GetUser() should return not found for nil user")
	}
}

func TestUserRepoServer_GetUser_Error(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	repo := &mockUserRepository{
		getByIDFn: func(ctx context.Context, id uint) (*entity.User, error) {
			return nil, errors.New("db error")
		},
	}
	server := NewUserServiceServer(repo, logger)
	ctx := context.Background()

	req := &pb.GetUserRequest{Id: 1}
	_, err := server.GetUser(ctx, req)
	if err == nil {
		t.Error("GetUser() should return error on db failure")
	}
}

func TestUserRepoServer_CreateUser_Success(t *testing.T) {
	server := newTestUserRepoServer()
	ctx := context.Background()

	firstName := "New"
	lastName := "User"
	req := &pb.CreateUserRequest{
		Username:  "newuser",
		Email:     "new@example.com",
		Password:  "hashed-pass",
		FirstName: &firstName,
		LastName:  &lastName,
	}

	resp, err := server.CreateUser(ctx, req)
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}
	if resp == nil {
		t.Fatal("CreateUser() returned nil")
	}
}

func TestUserRepoServer_CreateUser_Error(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	repo := &mockUserRepository{
		createFn: func(ctx context.Context, user *entity.User) error {
			return errors.New("create error")
		},
	}
	server := NewUserServiceServer(repo, logger)
	ctx := context.Background()

	req := &pb.CreateUserRequest{
		Username: "newuser",
		Email:    "new@example.com",
		Password: "pass",
	}

	_, err := server.CreateUser(ctx, req)
	if err == nil {
		t.Error("CreateUser() should return error on db failure")
	}
}

func TestUserRepoServer_UpdateUser_Success(t *testing.T) {
	email := "updated@example.com"
	firstName := "Updated"
	lastName := "Name"
	password := "newpass"
	isActive := true
	isVerified := true

	server := newTestUserRepoServer()
	ctx := context.Background()

	req := &pb.UpdateUserRequest{
		Id:         1,
		Email:      &email,
		FirstName:  &firstName,
		LastName:   &lastName,
		Password:   &password,
		IsActive:   &isActive,
		IsVerified: &isVerified,
	}

	resp, err := server.UpdateUser(ctx, req)
	if err != nil {
		t.Fatalf("UpdateUser() error = %v", err)
	}
	if resp == nil {
		t.Fatal("UpdateUser() returned nil")
	}
}

func TestUserRepoServer_UpdateUser_UserNotFound(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	repo := &mockUserRepository{
		getByIDFn: func(ctx context.Context, id uint) (*entity.User, error) {
			return nil, nil
		},
	}
	server := NewUserServiceServer(repo, logger)
	ctx := context.Background()

	req := &pb.UpdateUserRequest{Id: 999}
	_, err := server.UpdateUser(ctx, req)
	if err == nil {
		t.Error("UpdateUser() should return not found")
	}
}

func TestUserRepoServer_UpdateUser_GetError(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	repo := &mockUserRepository{
		getByIDFn: func(ctx context.Context, id uint) (*entity.User, error) {
			return nil, errors.New("db error")
		},
	}
	server := NewUserServiceServer(repo, logger)
	ctx := context.Background()

	req := &pb.UpdateUserRequest{Id: 1}
	_, err := server.UpdateUser(ctx, req)
	if err == nil {
		t.Error("UpdateUser() should return error on get failure")
	}
}

func TestUserRepoServer_UpdateUser_UpdateError(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	repo := &mockUserRepository{
		updateFn: func(ctx context.Context, user *entity.User) error {
			return errors.New("update error")
		},
	}
	server := NewUserServiceServer(repo, logger)
	ctx := context.Background()

	req := &pb.UpdateUserRequest{Id: 1}
	_, err := server.UpdateUser(ctx, req)
	if err == nil {
		t.Error("UpdateUser() should return error on update failure")
	}
}

func TestUserRepoServer_DeleteUser_Success(t *testing.T) {
	server := newTestUserRepoServer()
	ctx := context.Background()

	req := &pb.DeleteUserRequest{Id: 1}
	resp, err := server.DeleteUser(ctx, req)
	if err != nil {
		t.Fatalf("DeleteUser() error = %v", err)
	}
	if resp == nil {
		t.Error("DeleteUser() returned nil")
	}
}

func TestUserRepoServer_DeleteUser_Error(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	repo := &mockUserRepository{
		deleteFn: func(ctx context.Context, id uint) error {
			return errors.New("delete error")
		},
	}
	server := NewUserServiceServer(repo, logger)
	ctx := context.Background()

	req := &pb.DeleteUserRequest{Id: 1}
	_, err := server.DeleteUser(ctx, req)
	if err == nil {
		t.Error("DeleteUser() should return error on failure")
	}
}

func TestUserRepoServer_ListUsers_Success(t *testing.T) {
	server := newTestUserRepoServer()
	ctx := context.Background()

	req := &pb.ListUsersRequest{Page: 1, Size: 10}
	resp, err := server.ListUsers(ctx, req)
	if err != nil {
		t.Fatalf("ListUsers() error = %v", err)
	}
	if resp == nil {
		t.Fatal("ListUsers() returned nil")
	}
	if len(resp.Users) == 0 {
		t.Error("ListUsers() returned empty users")
	}
}

func TestUserRepoServer_ListUsers_DefaultPagination(t *testing.T) {
	server := newTestUserRepoServer()
	ctx := context.Background()

	// Test with zero/invalid values - should use defaults
	req := &pb.ListUsersRequest{Page: 0, Size: 0}
	resp, err := server.ListUsers(ctx, req)
	if err != nil {
		t.Fatalf("ListUsers() error = %v", err)
	}
	if resp == nil {
		t.Fatal("ListUsers() returned nil")
	}
	// Should default to size=10
	if resp.PageInfo.Size != 10 {
		t.Errorf("Default size = %v, want 10", resp.PageInfo.Size)
	}
}

func TestUserRepoServer_ListUsers_Error(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	repo := &mockUserRepository{
		listFn: func(ctx context.Context, page, size int) ([]*entity.User, int64, error) {
			return nil, 0, errors.New("list error")
		},
	}
	server := NewUserServiceServer(repo, logger)
	ctx := context.Background()

	req := &pb.ListUsersRequest{Page: 1, Size: 10}
	_, err := server.ListUsers(ctx, req)
	if err == nil {
		t.Error("ListUsers() should return error on db failure")
	}
}

func TestUserRepoServer_GetUserByUsername_Success(t *testing.T) {
	server := newTestUserRepoServer()
	ctx := context.Background()

	req := &pb.GetUserByUsernameRequest{Username: "testuser"}
	resp, err := server.GetUserByUsername(ctx, req)
	if err != nil {
		t.Fatalf("GetUserByUsername() error = %v", err)
	}
	if resp == nil {
		t.Fatal("GetUserByUsername() returned nil")
	}
}

func TestUserRepoServer_GetUserByUsername_NotFound(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	repo := &mockUserRepository{
		getByUsernameFn: func(ctx context.Context, username string) (*entity.User, error) {
			return nil, nil
		},
	}
	server := NewUserServiceServer(repo, logger)
	ctx := context.Background()

	req := &pb.GetUserByUsernameRequest{Username: "unknown"}
	_, err := server.GetUserByUsername(ctx, req)
	if err == nil {
		t.Error("GetUserByUsername() should return not found for nil user")
	}
}

func TestUserRepoServer_GetUserByUsername_Error(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	repo := &mockUserRepository{
		getByUsernameFn: func(ctx context.Context, username string) (*entity.User, error) {
			return nil, errors.New("db error")
		},
	}
	server := NewUserServiceServer(repo, logger)
	ctx := context.Background()

	req := &pb.GetUserByUsernameRequest{Username: "testuser"}
	_, err := server.GetUserByUsername(ctx, req)
	if err == nil {
		t.Error("GetUserByUsername() should return error on db failure")
	}
}

func TestUserRepoServer_GetUserByEmail_Success(t *testing.T) {
	server := newTestUserRepoServer()
	ctx := context.Background()

	req := &pb.GetUserByEmailRequest{Email: "test@example.com"}
	resp, err := server.GetUserByEmail(ctx, req)
	if err != nil {
		t.Fatalf("GetUserByEmail() error = %v", err)
	}
	if resp == nil {
		t.Fatal("GetUserByEmail() returned nil")
	}
}

func TestUserRepoServer_GetUserByEmail_NotFound(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	repo := &mockUserRepository{
		getByEmailFn: func(ctx context.Context, email string) (*entity.User, error) {
			return nil, nil
		},
	}
	server := NewUserServiceServer(repo, logger)
	ctx := context.Background()

	req := &pb.GetUserByEmailRequest{Email: "unknown@example.com"}
	_, err := server.GetUserByEmail(ctx, req)
	if err == nil {
		t.Error("GetUserByEmail() should return not found for nil user")
	}
}

func TestUserRepoServer_GetUserByEmail_Error(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	repo := &mockUserRepository{
		getByEmailFn: func(ctx context.Context, email string) (*entity.User, error) {
			return nil, errors.New("db error")
		},
	}
	server := NewUserServiceServer(repo, logger)
	ctx := context.Background()

	req := &pb.GetUserByEmailRequest{Email: "test@example.com"}
	_, err := server.GetUserByEmail(ctx, req)
	if err == nil {
		t.Error("GetUserByEmail() should return error on db failure")
	}
}

func TestUserRepoServer_ExistsByUsername_Success(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	repo := &mockUserRepository{
		existsByUsernameFn: func(ctx context.Context, username string) (bool, error) {
			return true, nil
		},
	}
	server := NewUserServiceServer(repo, logger)
	ctx := context.Background()

	req := &pb.ExistsByUsernameRequest{Username: "testuser"}
	resp, err := server.ExistsByUsername(ctx, req)
	if err != nil {
		t.Fatalf("ExistsByUsername() error = %v", err)
	}
	if !resp.Value {
		t.Error("ExistsByUsername() should return true")
	}
}

func TestUserRepoServer_ExistsByUsername_Error(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	repo := &mockUserRepository{
		existsByUsernameFn: func(ctx context.Context, username string) (bool, error) {
			return false, errors.New("check error")
		},
	}
	server := NewUserServiceServer(repo, logger)
	ctx := context.Background()

	req := &pb.ExistsByUsernameRequest{Username: "testuser"}
	_, err := server.ExistsByUsername(ctx, req)
	if err == nil {
		t.Error("ExistsByUsername() should return error on failure")
	}
}

func TestUserRepoServer_ExistsByEmail_Success(t *testing.T) {
	server := newTestUserRepoServer()
	ctx := context.Background()

	req := &pb.ExistsByEmailRequest{Email: "test@example.com"}
	resp, err := server.ExistsByEmail(ctx, req)
	if err != nil {
		t.Fatalf("ExistsByEmail() error = %v", err)
	}
	if resp == nil {
		t.Fatal("ExistsByEmail() returned nil")
	}
}

func TestUserRepoServer_ExistsByEmail_Error(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	repo := &mockUserRepository{
		existsByEmailFn: func(ctx context.Context, email string) (bool, error) {
			return false, errors.New("check error")
		},
	}
	server := NewUserServiceServer(repo, logger)
	ctx := context.Background()

	req := &pb.ExistsByEmailRequest{Email: "test@example.com"}
	_, err := server.ExistsByEmail(ctx, req)
	if err == nil {
		t.Error("ExistsByEmail() should return error on failure")
	}
}

func TestUserRepoServer_ToProtoTimestamp(t *testing.T) {
	now := time.Now()
	ts := toProtoTimestamp(now)
	if ts == nil {
		t.Error("toProtoTimestamp() returned nil")
	}
	if ts.Seconds != now.Unix() {
		t.Errorf("Seconds = %v, want %v", ts.Seconds, now.Unix())
	}
}
