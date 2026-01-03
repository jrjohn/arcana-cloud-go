package grpc

import (
	"context"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/jrjohn/arcana-cloud-go/api/proto/pb"
	"github.com/jrjohn/arcana-cloud-go/internal/domain/entity"
	"github.com/jrjohn/arcana-cloud-go/internal/domain/repository"
)

// UserRepositoryGRPC implements UserRepository using gRPC calls
type UserRepositoryGRPC struct {
	client pb.UserServiceClient
}

// NewUserRepositoryGRPC creates a new gRPC-backed UserRepository
func NewUserRepositoryGRPC(client pb.UserServiceClient) repository.UserRepository {
	return &UserRepositoryGRPC{client: client}
}

// Create creates a new user
func (r *UserRepositoryGRPC) Create(ctx context.Context, user *entity.User) error {
	req := &pb.CreateUserRequest{
		Username: user.Username,
		Email:    user.Email,
		Password: user.Password,
	}
	if user.FirstName != "" {
		req.FirstName = &user.FirstName
	}
	if user.LastName != "" {
		req.LastName = &user.LastName
	}

	resp, err := r.client.CreateUser(ctx, req)
	if err != nil {
		return err
	}

	// Update user with returned ID
	user.ID = uint(resp.Id)
	user.CreatedAt = fromProtoTimestamp(resp.CreatedAt)
	user.UpdatedAt = fromProtoTimestamp(resp.UpdatedAt)
	return nil
}

// GetByID retrieves a user by ID
func (r *UserRepositoryGRPC) GetByID(ctx context.Context, id uint) (*entity.User, error) {
	resp, err := r.client.GetUser(ctx, &pb.GetUserRequest{Id: uint64(id)})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, nil
		}
		return nil, err
	}
	return r.toEntity(resp), nil
}

// GetByUsername retrieves a user by username
func (r *UserRepositoryGRPC) GetByUsername(ctx context.Context, username string) (*entity.User, error) {
	resp, err := r.client.GetUserByUsername(ctx, &pb.GetUserByUsernameRequest{Username: username})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, nil
		}
		return nil, err
	}
	return r.toEntity(resp), nil
}

// GetByEmail retrieves a user by email
func (r *UserRepositoryGRPC) GetByEmail(ctx context.Context, email string) (*entity.User, error) {
	resp, err := r.client.GetUserByEmail(ctx, &pb.GetUserByEmailRequest{Email: email})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, nil
		}
		return nil, err
	}
	return r.toEntity(resp), nil
}

// GetByUsernameOrEmail retrieves a user by username or email
func (r *UserRepositoryGRPC) GetByUsernameOrEmail(ctx context.Context, usernameOrEmail string) (*entity.User, error) {
	// Try username first
	user, err := r.GetByUsername(ctx, usernameOrEmail)
	if err != nil {
		return nil, err
	}
	if user != nil {
		return user, nil
	}

	// Try email
	return r.GetByEmail(ctx, usernameOrEmail)
}

// Update updates an existing user
func (r *UserRepositoryGRPC) Update(ctx context.Context, user *entity.User) error {
	req := &pb.UpdateUserRequest{
		Id:         uint64(user.ID),
		Email:      &user.Email,
		FirstName:  &user.FirstName,
		LastName:   &user.LastName,
		Password:   &user.Password,
		IsActive:   &user.IsActive,
		IsVerified: &user.IsVerified,
	}

	resp, err := r.client.UpdateUser(ctx, req)
	if err != nil {
		return err
	}

	user.UpdatedAt = fromProtoTimestamp(resp.UpdatedAt)
	return nil
}

// Delete soft-deletes a user by ID
func (r *UserRepositoryGRPC) Delete(ctx context.Context, id uint) error {
	_, err := r.client.DeleteUser(ctx, &pb.DeleteUserRequest{Id: uint64(id)})
	return err
}

// List retrieves users with pagination
func (r *UserRepositoryGRPC) List(ctx context.Context, page, size int) ([]*entity.User, int64, error) {
	resp, err := r.client.ListUsers(ctx, &pb.ListUsersRequest{
		Page: int32(page),
		Size: int32(size),
	})
	if err != nil {
		return nil, 0, err
	}

	users := make([]*entity.User, len(resp.Users))
	for i, u := range resp.Users {
		users[i] = r.toEntity(u)
	}

	return users, resp.PageInfo.TotalItems, nil
}

// ExistsByUsername checks if a username exists
func (r *UserRepositoryGRPC) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	resp, err := r.client.ExistsByUsername(ctx, &pb.ExistsByUsernameRequest{Username: username})
	if err != nil {
		return false, err
	}
	return resp.Value, nil
}

// ExistsByEmail checks if an email exists
func (r *UserRepositoryGRPC) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	resp, err := r.client.ExistsByEmail(ctx, &pb.ExistsByEmailRequest{Email: email})
	if err != nil {
		return false, err
	}
	return resp.Value, nil
}

func (r *UserRepositoryGRPC) toEntity(resp *pb.UserResponse) *entity.User {
	if resp == nil {
		return nil
	}
	return &entity.User{
		ID:         uint(resp.Id),
		Username:   resp.Username,
		Email:      resp.Email,
		FirstName:  resp.FirstName,
		LastName:   resp.LastName,
		Role:       entity.UserRole(resp.Role),
		IsActive:   resp.IsActive,
		IsVerified: resp.IsVerified,
		CreatedAt:  fromProtoTimestamp(resp.CreatedAt),
		UpdatedAt:  fromProtoTimestamp(resp.UpdatedAt),
	}
}

func fromProtoTimestamp(t *pb.Timestamp) time.Time {
	if t == nil {
		return time.Time{}
	}
	return time.Unix(t.Seconds, int64(t.Nanos))
}
