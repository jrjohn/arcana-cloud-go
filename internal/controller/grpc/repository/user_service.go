package repository

import (
	"context"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/jrjohn/arcana-cloud-go/api/proto/pb"
	"github.com/jrjohn/arcana-cloud-go/internal/domain/entity"
	"github.com/jrjohn/arcana-cloud-go/internal/domain/repository"
)

// UserServiceServer implements the gRPC UserService for the Repository layer
type UserServiceServer struct {
	pb.UnimplementedUserServiceServer
	userRepo repository.UserRepository
	logger   *zap.Logger
}

// NewUserServiceServer creates a new UserServiceServer
func NewUserServiceServer(userRepo repository.UserRepository, logger *zap.Logger) *UserServiceServer {
	return &UserServiceServer{
		userRepo: userRepo,
		logger:   logger,
	}
}

// GetUser retrieves a user by ID
func (s *UserServiceServer) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.UserResponse, error) {
	user, err := s.userRepo.GetByID(ctx, uint(req.Id))
	if err != nil {
		s.logger.Error("failed to get user", zap.Error(err), zap.Uint64("id", req.Id))
		return nil, status.Errorf(codes.Internal, "failed to get user: %v", err)
	}
	if user == nil {
		return nil, status.Error(codes.NotFound, "user not found")
	}
	return s.toProtoUser(user), nil
}

// CreateUser creates a new user
func (s *UserServiceServer) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.UserResponse, error) {
	user := &entity.User{
		Username: req.Username,
		Email:    req.Email,
		Password: req.Password, // Already hashed by service layer
		Role:     entity.RoleUser,
		IsActive: true,
	}

	if req.FirstName != nil {
		user.FirstName = *req.FirstName
	}
	if req.LastName != nil {
		user.LastName = *req.LastName
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		s.logger.Error("failed to create user", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to create user: %v", err)
	}

	return s.toProtoUser(user), nil
}

// UpdateUser updates an existing user
func (s *UserServiceServer) UpdateUser(ctx context.Context, req *pb.UpdateUserRequest) (*pb.UserResponse, error) {
	user, err := s.userRepo.GetByID(ctx, uint(req.Id))
	if err != nil {
		s.logger.Error("failed to get user for update", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to get user: %v", err)
	}
	if user == nil {
		return nil, status.Error(codes.NotFound, "user not found")
	}

	if req.Email != nil {
		user.Email = *req.Email
	}
	if req.FirstName != nil {
		user.FirstName = *req.FirstName
	}
	if req.LastName != nil {
		user.LastName = *req.LastName
	}
	if req.Password != nil {
		user.Password = *req.Password
	}
	if req.IsActive != nil {
		user.IsActive = *req.IsActive
	}
	if req.IsVerified != nil {
		user.IsVerified = *req.IsVerified
	}

	if err := s.userRepo.Update(ctx, user); err != nil {
		s.logger.Error("failed to update user", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to update user: %v", err)
	}

	return s.toProtoUser(user), nil
}

// DeleteUser deletes a user
func (s *UserServiceServer) DeleteUser(ctx context.Context, req *pb.DeleteUserRequest) (*pb.Empty, error) {
	if err := s.userRepo.Delete(ctx, uint(req.Id)); err != nil {
		s.logger.Error("failed to delete user", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to delete user: %v", err)
	}
	return &pb.Empty{}, nil
}

// ListUsers retrieves users with pagination
func (s *UserServiceServer) ListUsers(ctx context.Context, req *pb.ListUsersRequest) (*pb.ListUsersResponse, error) {
	page := int(req.Page)
	size := int(req.Size)
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 10
	}

	users, total, err := s.userRepo.List(ctx, page, size)
	if err != nil {
		s.logger.Error("failed to list users", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to list users: %v", err)
	}

	protoUsers := make([]*pb.UserResponse, len(users))
	for i, user := range users {
		protoUsers[i] = s.toProtoUser(user)
	}

	return &pb.ListUsersResponse{
		Users: protoUsers,
		PageInfo: &pb.PageInfo{
			Page:       int32(page),
			Size:       int32(size),
			TotalItems: total,
			TotalPages: int32((total + int64(size) - 1) / int64(size)),
		},
	}, nil
}

// GetUserByUsername retrieves a user by username
func (s *UserServiceServer) GetUserByUsername(ctx context.Context, req *pb.GetUserByUsernameRequest) (*pb.UserResponse, error) {
	user, err := s.userRepo.GetByUsername(ctx, req.Username)
	if err != nil {
		s.logger.Error("failed to get user by username", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to get user: %v", err)
	}
	if user == nil {
		return nil, status.Error(codes.NotFound, "user not found")
	}
	return s.toProtoUser(user), nil
}

// GetUserByEmail retrieves a user by email
func (s *UserServiceServer) GetUserByEmail(ctx context.Context, req *pb.GetUserByEmailRequest) (*pb.UserResponse, error) {
	user, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		s.logger.Error("failed to get user by email", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to get user: %v", err)
	}
	if user == nil {
		return nil, status.Error(codes.NotFound, "user not found")
	}
	return s.toProtoUser(user), nil
}

// ExistsByUsername checks if a username exists
func (s *UserServiceServer) ExistsByUsername(ctx context.Context, req *pb.ExistsByUsernameRequest) (*pb.BoolValue, error) {
	exists, err := s.userRepo.ExistsByUsername(ctx, req.Username)
	if err != nil {
		s.logger.Error("failed to check username exists", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to check: %v", err)
	}
	return &pb.BoolValue{Value: exists}, nil
}

// ExistsByEmail checks if an email exists
func (s *UserServiceServer) ExistsByEmail(ctx context.Context, req *pb.ExistsByEmailRequest) (*pb.BoolValue, error) {
	exists, err := s.userRepo.ExistsByEmail(ctx, req.Email)
	if err != nil {
		s.logger.Error("failed to check email exists", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to check: %v", err)
	}
	return &pb.BoolValue{Value: exists}, nil
}

func (s *UserServiceServer) toProtoUser(user *entity.User) *pb.UserResponse {
	return &pb.UserResponse{
		Id:         uint64(user.ID),
		Username:   user.Username,
		Email:      user.Email,
		FirstName:  user.FirstName,
		LastName:   user.LastName,
		Role:       string(user.Role),
		IsActive:   user.IsActive,
		IsVerified: user.IsVerified,
		CreatedAt:  toProtoTimestamp(user.CreatedAt),
		UpdatedAt:  toProtoTimestamp(user.UpdatedAt),
	}
}

func toProtoTimestamp(t time.Time) *pb.Timestamp {
	return &pb.Timestamp{
		Seconds: t.Unix(),
		Nanos:   int32(t.Nanosecond()),
	}
}
