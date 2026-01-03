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
	"github.com/jrjohn/arcana-cloud-go/internal/security"
)

// AuthServiceServer implements the gRPC AuthService for the Service layer
type AuthServiceServer struct {
	pb.UnimplementedAuthServiceServer
	authService domainservice.AuthService
	jwtProvider *security.JWTProvider
	logger      *zap.Logger
}

// NewAuthServiceServer creates a new AuthServiceServer
func NewAuthServiceServer(
	authService domainservice.AuthService,
	jwtProvider *security.JWTProvider,
	logger *zap.Logger,
) *AuthServiceServer {
	return &AuthServiceServer{
		authService: authService,
		jwtProvider: jwtProvider,
		logger:      logger,
	}
}

// Register creates a new user account
func (s *AuthServiceServer) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.AuthResponse, error) {
	registerReq := &request.RegisterRequest{
		Username:  req.Username,
		Email:     req.Email,
		Password:  req.Password,
		FirstName: req.GetFirstName(),
		LastName:  req.GetLastName(),
	}

	authResp, err := s.authService.Register(ctx, registerReq)
	if err != nil {
		s.logger.Error("failed to register user", zap.Error(err))
		return nil, s.mapError(err)
	}

	return s.toProtoAuthResponse(authResp), nil
}

// Login authenticates a user
func (s *AuthServiceServer) Login(ctx context.Context, req *pb.LoginRequest) (*pb.AuthResponse, error) {
	loginReq := &request.LoginRequest{
		UsernameOrEmail: req.UsernameOrEmail,
		Password:        req.Password,
	}

	authResp, err := s.authService.Login(ctx, loginReq)
	if err != nil {
		s.logger.Error("failed to login", zap.Error(err))
		return nil, s.mapError(err)
	}

	return s.toProtoAuthResponse(authResp), nil
}

// RefreshToken generates new tokens
func (s *AuthServiceServer) RefreshToken(ctx context.Context, req *pb.RefreshTokenRequest) (*pb.AuthResponse, error) {
	refreshReq := &request.RefreshTokenRequest{
		RefreshToken: req.RefreshToken,
	}

	authResp, err := s.authService.RefreshToken(ctx, refreshReq)
	if err != nil {
		s.logger.Error("failed to refresh token", zap.Error(err))
		return nil, s.mapError(err)
	}

	return s.toProtoAuthResponse(authResp), nil
}

// Logout invalidates the current token
func (s *AuthServiceServer) Logout(ctx context.Context, req *pb.LogoutRequest) (*pb.Empty, error) {
	if err := s.authService.Logout(ctx, req.Token); err != nil {
		s.logger.Error("failed to logout", zap.Error(err))
		return nil, s.mapError(err)
	}
	return &pb.Empty{}, nil
}

// LogoutAll invalidates all tokens for a user
func (s *AuthServiceServer) LogoutAll(ctx context.Context, req *pb.LogoutAllRequest) (*pb.Empty, error) {
	if err := s.authService.LogoutAll(ctx, uint(req.UserId)); err != nil {
		s.logger.Error("failed to logout all", zap.Error(err))
		return nil, s.mapError(err)
	}
	return &pb.Empty{}, nil
}

// ValidateToken validates an access token
func (s *AuthServiceServer) ValidateToken(ctx context.Context, req *pb.ValidateTokenRequest) (*pb.ValidateTokenResponse, error) {
	claims, err := s.jwtProvider.ValidateAccessToken(req.Token)
	if err != nil {
		return &pb.ValidateTokenResponse{Valid: false}, nil
	}

	return &pb.ValidateTokenResponse{
		Valid:    true,
		UserId:   uint64(claims.UserID),
		Username: claims.Username,
		Role:     string(claims.Role),
	}, nil
}

func (s *AuthServiceServer) toProtoAuthResponse(resp *response.AuthResponse) *pb.AuthResponse {
	if resp == nil {
		return nil
	}
	return &pb.AuthResponse{
		AccessToken:  resp.AccessToken,
		RefreshToken: resp.RefreshToken,
		TokenType:    resp.TokenType,
		ExpiresIn:    resp.ExpiresIn,
		User: &pb.UserResponse{
			Id:         uint64(resp.User.ID),
			Username:   resp.User.Username,
			Email:      resp.User.Email,
			FirstName:  resp.User.FirstName,
			LastName:   resp.User.LastName,
			Role:       resp.User.Role,
			IsActive:   resp.User.IsActive,
			IsVerified: resp.User.IsVerified,
			CreatedAt:  toProtoTimestamp(resp.User.CreatedAt),
			UpdatedAt:  toProtoTimestamp(resp.User.UpdatedAt),
		},
	}
}

func toProtoTimestamp(t time.Time) *pb.Timestamp {
	return &pb.Timestamp{
		Seconds: t.Unix(),
		Nanos:   int32(t.Nanosecond()),
	}
}

func (s *AuthServiceServer) mapError(err error) error {
	switch err {
	case domainservice.ErrUserNotFound:
		return status.Error(codes.NotFound, "user not found")
	case domainservice.ErrInvalidCredentials:
		return status.Error(codes.Unauthenticated, "invalid credentials")
	case domainservice.ErrUserAlreadyExists:
		return status.Error(codes.AlreadyExists, "user already exists")
	case domainservice.ErrInvalidToken:
		return status.Error(codes.Unauthenticated, "invalid or expired token")
	case domainservice.ErrUserInactive:
		return status.Error(codes.PermissionDenied, "user account is inactive")
	default:
		return status.Errorf(codes.Internal, "internal error: %v", err)
	}
}
