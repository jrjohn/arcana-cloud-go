package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/jrjohn/arcana-cloud-go/api/proto/pb"
	domainservice "github.com/jrjohn/arcana-cloud-go/internal/domain/service"
	"github.com/jrjohn/arcana-cloud-go/internal/dto/request"
	"github.com/jrjohn/arcana-cloud-go/internal/dto/response"
)

// mockUserService for testing
type mockUserService struct {
	getByIDFn          func(ctx context.Context, id uint) (*response.UserResponse, error)
	getByUsernameFn    func(ctx context.Context, username string) (*response.UserResponse, error)
	getByEmailFn       func(ctx context.Context, email string) (*response.UserResponse, error)
	listFn             func(ctx context.Context, page, size int) (*response.PagedResponse[response.UserResponse], error)
	updateFn           func(ctx context.Context, id uint, req *request.UpdateProfileRequest) (*response.UserResponse, error)
	changePasswordFn   func(ctx context.Context, id uint, req *request.ChangePasswordRequest) error
	deleteFn           func(ctx context.Context, id uint) error
	existsByUsernameFn func(ctx context.Context, username string) (bool, error)
	existsByEmailFn    func(ctx context.Context, email string) (bool, error)
}

func (m *mockUserService) GetByID(ctx context.Context, id uint) (*response.UserResponse, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return newUserResponse(), nil
}

func (m *mockUserService) GetByUsername(ctx context.Context, username string) (*response.UserResponse, error) {
	if m.getByUsernameFn != nil {
		return m.getByUsernameFn(ctx, username)
	}
	return newUserResponse(), nil
}

func (m *mockUserService) GetByEmail(ctx context.Context, email string) (*response.UserResponse, error) {
	if m.getByEmailFn != nil {
		return m.getByEmailFn(ctx, email)
	}
	return newUserResponse(), nil
}

func (m *mockUserService) List(ctx context.Context, page, size int) (*response.PagedResponse[response.UserResponse], error) {
	if m.listFn != nil {
		return m.listFn(ctx, page, size)
	}
	return &response.PagedResponse[response.UserResponse]{
		Items: []response.UserResponse{*newUserResponse()},
		PageInfo: response.PageInfo{
			Page:       page,
			Size:       size,
			TotalItems: 1,
			TotalPages: 1,
		},
	}, nil
}

func (m *mockUserService) Update(ctx context.Context, id uint, req *request.UpdateProfileRequest) (*response.UserResponse, error) {
	if m.updateFn != nil {
		return m.updateFn(ctx, id, req)
	}
	return newUserResponse(), nil
}

func (m *mockUserService) ChangePassword(ctx context.Context, id uint, req *request.ChangePasswordRequest) error {
	if m.changePasswordFn != nil {
		return m.changePasswordFn(ctx, id, req)
	}
	return nil
}

func (m *mockUserService) Delete(ctx context.Context, id uint) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return nil
}

func (m *mockUserService) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	if m.existsByUsernameFn != nil {
		return m.existsByUsernameFn(ctx, username)
	}
	return false, nil
}

func (m *mockUserService) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	if m.existsByEmailFn != nil {
		return m.existsByEmailFn(ctx, email)
	}
	return false, nil
}

func newUserResponse() *response.UserResponse {
	now := time.Now()
	return &response.UserResponse{
		ID:         1,
		Username:   "testuser",
		Email:      "test@example.com",
		FirstName:  "Test",
		LastName:   "User",
		Role:       "user",
		IsActive:   true,
		IsVerified: true,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

func newUserServiceServer() *UserServiceServer {
	logger, _ := zap.NewDevelopment()
	svc := &mockUserService{}
	return NewUserServiceServer(svc, logger)
}

func TestNewUserServiceServer(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	svc := &mockUserService{}
	server := NewUserServiceServer(svc, logger)
	if server == nil {
		t.Error("NewUserServiceServer() returned nil")
	}
}

func TestUserServiceServer_GetUser_Success(t *testing.T) {
	server := newUserServiceServer()
	ctx := context.Background()

	req := &pb.GetUserRequest{Id: 1}
	resp, err := server.GetUser(ctx, req)
	if err != nil {
		t.Fatalf("GetUser() error = %v", err)
	}
	if resp == nil {
		t.Fatal("GetUser() returned nil response")
	}
	if resp.Id != 1 {
		t.Errorf("Id = %v, want 1", resp.Id)
	}
}

func TestUserServiceServer_GetUser_NotFound(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	svc := &mockUserService{
		getByIDFn: func(ctx context.Context, id uint) (*response.UserResponse, error) {
			return nil, domainservice.ErrUserNotFound
		},
	}
	server := NewUserServiceServer(svc, logger)
	ctx := context.Background()

	req := &pb.GetUserRequest{Id: 999}
	_, err := server.GetUser(ctx, req)
	if err == nil {
		t.Error("GetUser() should return error for not found user")
	}
}

func TestUserServiceServer_CreateUser(t *testing.T) {
	server := newUserServiceServer()
	ctx := context.Background()

	req := &pb.CreateUserRequest{
		Username: "newuser",
		Email:    "new@example.com",
		Password: "pass",
	}

	resp, err := server.CreateUser(ctx, req)
	if err == nil {
		t.Error("CreateUser() should return unimplemented error")
	}
	_ = resp
}

func TestUserServiceServer_UpdateUser_Success(t *testing.T) {
	email := "updated@example.com"
	firstName := "Updated"
	lastName := "Name"

	logger, _ := zap.NewDevelopment()
	svc := &mockUserService{
		updateFn: func(ctx context.Context, id uint, req *request.UpdateProfileRequest) (*response.UserResponse, error) {
			return &response.UserResponse{
				ID:        id,
				Email:     req.Email,
				FirstName: req.FirstName,
				LastName:  req.LastName,
			}, nil
		},
	}
	server := NewUserServiceServer(svc, logger)
	ctx := context.Background()

	req := &pb.UpdateUserRequest{
		Id:        1,
		Email:     &email,
		FirstName: &firstName,
		LastName:  &lastName,
	}

	resp, err := server.UpdateUser(ctx, req)
	if err != nil {
		t.Fatalf("UpdateUser() error = %v", err)
	}
	if resp == nil {
		t.Fatal("UpdateUser() returned nil")
	}
}

func TestUserServiceServer_UpdateUser_Error(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	svc := &mockUserService{
		updateFn: func(ctx context.Context, id uint, req *request.UpdateProfileRequest) (*response.UserResponse, error) {
			return nil, domainservice.ErrUserNotFound
		},
	}
	server := NewUserServiceServer(svc, logger)
	ctx := context.Background()

	req := &pb.UpdateUserRequest{Id: 999}
	_, err := server.UpdateUser(ctx, req)
	if err == nil {
		t.Error("UpdateUser() should return error for not found user")
	}
}

func TestUserServiceServer_DeleteUser_Success(t *testing.T) {
	server := newUserServiceServer()
	ctx := context.Background()

	req := &pb.DeleteUserRequest{Id: 1}
	resp, err := server.DeleteUser(ctx, req)
	if err != nil {
		t.Fatalf("DeleteUser() error = %v", err)
	}
	if resp == nil {
		t.Error("DeleteUser() returned nil response")
	}
}

func TestUserServiceServer_DeleteUser_Error(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	svc := &mockUserService{
		deleteFn: func(ctx context.Context, id uint) error {
			return errors.New("delete error")
		},
	}
	server := NewUserServiceServer(svc, logger)
	ctx := context.Background()

	req := &pb.DeleteUserRequest{Id: 1}
	_, err := server.DeleteUser(ctx, req)
	if err == nil {
		t.Error("DeleteUser() should return error on failure")
	}
}

func TestUserServiceServer_ListUsers_Success(t *testing.T) {
	server := newUserServiceServer()
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

func TestUserServiceServer_ListUsers_Error(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	svc := &mockUserService{
		listFn: func(ctx context.Context, page, size int) (*response.PagedResponse[response.UserResponse], error) {
			return nil, errors.New("list error")
		},
	}
	server := NewUserServiceServer(svc, logger)
	ctx := context.Background()

	req := &pb.ListUsersRequest{Page: 1, Size: 10}
	_, err := server.ListUsers(ctx, req)
	if err == nil {
		t.Error("ListUsers() should return error on failure")
	}
}

func TestUserServiceServer_GetUserByUsername_Success(t *testing.T) {
	server := newUserServiceServer()
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

func TestUserServiceServer_GetUserByUsername_Error(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	svc := &mockUserService{
		getByUsernameFn: func(ctx context.Context, username string) (*response.UserResponse, error) {
			return nil, domainservice.ErrUserNotFound
		},
	}
	server := NewUserServiceServer(svc, logger)
	ctx := context.Background()

	req := &pb.GetUserByUsernameRequest{Username: "nonexistent"}
	_, err := server.GetUserByUsername(ctx, req)
	if err == nil {
		t.Error("GetUserByUsername() should return error for not found user")
	}
}

func TestUserServiceServer_GetUserByEmail_Success(t *testing.T) {
	server := newUserServiceServer()
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

func TestUserServiceServer_GetUserByEmail_Error(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	svc := &mockUserService{
		getByEmailFn: func(ctx context.Context, email string) (*response.UserResponse, error) {
			return nil, domainservice.ErrUserNotFound
		},
	}
	server := NewUserServiceServer(svc, logger)
	ctx := context.Background()

	req := &pb.GetUserByEmailRequest{Email: "notfound@example.com"}
	_, err := server.GetUserByEmail(ctx, req)
	if err == nil {
		t.Error("GetUserByEmail() should return error for not found user")
	}
}

func TestUserServiceServer_ExistsByUsername_Success(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	svc := &mockUserService{
		existsByUsernameFn: func(ctx context.Context, username string) (bool, error) {
			return true, nil
		},
	}
	server := NewUserServiceServer(svc, logger)
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

func TestUserServiceServer_ExistsByUsername_Error(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	svc := &mockUserService{
		existsByUsernameFn: func(ctx context.Context, username string) (bool, error) {
			return false, errors.New("check error")
		},
	}
	server := NewUserServiceServer(svc, logger)
	ctx := context.Background()

	req := &pb.ExistsByUsernameRequest{Username: "testuser"}
	_, err := server.ExistsByUsername(ctx, req)
	if err == nil {
		t.Error("ExistsByUsername() should return error on failure")
	}
}

func TestUserServiceServer_ExistsByEmail_Success(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	svc := &mockUserService{
		existsByEmailFn: func(ctx context.Context, email string) (bool, error) {
			return false, nil
		},
	}
	server := NewUserServiceServer(svc, logger)
	ctx := context.Background()

	req := &pb.ExistsByEmailRequest{Email: "test@example.com"}
	resp, err := server.ExistsByEmail(ctx, req)
	if err != nil {
		t.Fatalf("ExistsByEmail() error = %v", err)
	}
	if resp.Value {
		t.Error("ExistsByEmail() should return false")
	}
}

func TestUserServiceServer_ExistsByEmail_Error(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	svc := &mockUserService{
		existsByEmailFn: func(ctx context.Context, email string) (bool, error) {
			return false, errors.New("check error")
		},
	}
	server := NewUserServiceServer(svc, logger)
	ctx := context.Background()

	req := &pb.ExistsByEmailRequest{Email: "test@example.com"}
	_, err := server.ExistsByEmail(ctx, req)
	if err == nil {
		t.Error("ExistsByEmail() should return error on failure")
	}
}

func TestUserServiceServer_MapError_AllCases(t *testing.T) {
	server := newUserServiceServer()

	tests := []struct {
		name string
		err  error
	}{
		{"UserNotFound", domainservice.ErrUserNotFound},
		{"UserAlreadyExists", domainservice.ErrUserAlreadyExists},
		{"InternalError", errors.New("some error")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := server.mapError(tt.err)
			if result == nil {
				t.Error("mapError() returned nil")
			}
		})
	}
}

func TestUserServiceServer_ToProtoUser_Nil(t *testing.T) {
	server := newUserServiceServer()
	result := server.toProtoUser(nil)
	if result != nil {
		t.Error("toProtoUser(nil) should return nil")
	}
}
