package service

import (
	"context"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/jrjohn/arcana-cloud-go/api/proto/pb"
	domainservice "github.com/jrjohn/arcana-cloud-go/internal/domain/service"
	"github.com/jrjohn/arcana-cloud-go/internal/dto/request"
	"github.com/jrjohn/arcana-cloud-go/internal/dto/response"
)

// UserServiceServer implements the gRPC UserService for the Service layer
type UserServiceServer struct {
	pb.UnimplementedUserServiceServer
	userService domainservice.UserService
	logger      *zap.Logger
}

// NewUserServiceServer creates a new UserServiceServer
func NewUserServiceServer(userService domainservice.UserService, logger *zap.Logger) *UserServiceServer {
	return &UserServiceServer{
		userService: userService,
		logger:      logger,
	}
}

// GetUser retrieves a user by ID
func (s *UserServiceServer) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.UserResponse, error) {
	user, err := s.userService.GetByID(ctx, uint(req.Id))
	if err != nil {
		s.logger.Error("failed to get user", zap.Error(err), zap.Uint64("id", req.Id))
		return nil, s.mapError(err)
	}
	return s.toProtoUser(user), nil
}

// CreateUser is not implemented in UserService (use AuthService.Register instead)
func (s *UserServiceServer) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.UserResponse, error) {
	return nil, status.Error(codes.Unimplemented, "use AuthService.Register to create users")
}

// UpdateUser updates an existing user
func (s *UserServiceServer) UpdateUser(ctx context.Context, req *pb.UpdateUserRequest) (*pb.UserResponse, error) {
	updateReq := &request.UpdateProfileRequest{}
	if req.Email != nil {
		updateReq.Email = *req.Email
	}
	if req.FirstName != nil {
		updateReq.FirstName = *req.FirstName
	}
	if req.LastName != nil {
		updateReq.LastName = *req.LastName
	}

	user, err := s.userService.Update(ctx, uint(req.Id), updateReq)
	if err != nil {
		s.logger.Error("failed to update user", zap.Error(err))
		return nil, s.mapError(err)
	}

	return s.toProtoUser(user), nil
}

// DeleteUser deletes a user
func (s *UserServiceServer) DeleteUser(ctx context.Context, req *pb.DeleteUserRequest) (*pb.Empty, error) {
	if err := s.userService.Delete(ctx, uint(req.Id)); err != nil {
		s.logger.Error("failed to delete user", zap.Error(err))
		return nil, s.mapError(err)
	}
	return &pb.Empty{}, nil
}

// ListUsers retrieves users with pagination
func (s *UserServiceServer) ListUsers(ctx context.Context, req *pb.ListUsersRequest) (*pb.ListUsersResponse, error) {
	page := int(req.Page)
	size := int(req.Size)

	result, err := s.userService.List(ctx, page, size)
	if err != nil {
		s.logger.Error("failed to list users", zap.Error(err))
		return nil, s.mapError(err)
	}

	protoUsers := make([]*pb.UserResponse, len(result.Items))
	for i, user := range result.Items {
		protoUsers[i] = s.toProtoUser(&user)
	}

	return &pb.ListUsersResponse{
		Users: protoUsers,
		PageInfo: &pb.PageInfo{
			Page:       int32(result.PageInfo.Page),
			Size:       int32(result.PageInfo.Size),
			TotalItems: result.PageInfo.TotalItems,
			TotalPages: int32(result.PageInfo.TotalPages),
		},
	}, nil
}

// GetUserByUsername retrieves a user by username
func (s *UserServiceServer) GetUserByUsername(ctx context.Context, req *pb.GetUserByUsernameRequest) (*pb.UserResponse, error) {
	user, err := s.userService.GetByUsername(ctx, req.Username)
	if err != nil {
		s.logger.Error("failed to get user by username", zap.Error(err))
		return nil, s.mapError(err)
	}
	return s.toProtoUser(user), nil
}

// GetUserByEmail retrieves a user by email
func (s *UserServiceServer) GetUserByEmail(ctx context.Context, req *pb.GetUserByEmailRequest) (*pb.UserResponse, error) {
	user, err := s.userService.GetByEmail(ctx, req.Email)
	if err != nil {
		s.logger.Error("failed to get user by email", zap.Error(err))
		return nil, s.mapError(err)
	}
	return s.toProtoUser(user), nil
}

// ExistsByUsername checks if a username exists
func (s *UserServiceServer) ExistsByUsername(ctx context.Context, req *pb.ExistsByUsernameRequest) (*pb.BoolValue, error) {
	exists, err := s.userService.ExistsByUsername(ctx, req.Username)
	if err != nil {
		s.logger.Error("failed to check username exists", zap.Error(err))
		return nil, s.mapError(err)
	}
	return &pb.BoolValue{Value: exists}, nil
}

// ExistsByEmail checks if an email exists
func (s *UserServiceServer) ExistsByEmail(ctx context.Context, req *pb.ExistsByEmailRequest) (*pb.BoolValue, error) {
	exists, err := s.userService.ExistsByEmail(ctx, req.Email)
	if err != nil {
		s.logger.Error("failed to check email exists", zap.Error(err))
		return nil, s.mapError(err)
	}
	return &pb.BoolValue{Value: exists}, nil
}

func (s *UserServiceServer) toProtoUser(user *response.UserResponse) *pb.UserResponse {
	if user == nil {
		return nil
	}
	return &pb.UserResponse{
		Id:         uint64(user.ID),
		Username:   user.Username,
		Email:      user.Email,
		FirstName:  user.FirstName,
		LastName:   user.LastName,
		Role:       user.Role,
		IsActive:   user.IsActive,
		IsVerified: user.IsVerified,
		CreatedAt: &pb.Timestamp{
			Seconds: user.CreatedAt.Unix(),
			Nanos:   int32(user.CreatedAt.Nanosecond()),
		},
		UpdatedAt: &pb.Timestamp{
			Seconds: user.UpdatedAt.Unix(),
			Nanos:   int32(user.UpdatedAt.Nanosecond()),
		},
	}
}

func fromProtoTimestamp(t *pb.Timestamp) time.Time {
	if t == nil {
		return time.Time{}
	}
	return time.Unix(t.Seconds, int64(t.Nanos))
}

func (s *UserServiceServer) mapError(err error) error {
	switch err {
	case domainservice.ErrUserNotFound:
		return status.Error(codes.NotFound, "user not found")
	case domainservice.ErrUserAlreadyExists:
		return status.Error(codes.AlreadyExists, "user already exists")
	default:
		return status.Errorf(codes.Internal, "internal error: %v", err)
	}
}
