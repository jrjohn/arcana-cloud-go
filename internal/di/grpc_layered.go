package di

import (
	"context"
	"os"
	"strconv"

	"go.uber.org/fx"
	"go.uber.org/zap"

	"github.com/jrjohn/arcana-cloud-go/api/proto/pb"
	"github.com/jrjohn/arcana-cloud-go/internal/config"
	grpcctrl "github.com/jrjohn/arcana-cloud-go/internal/controller/grpc"
	grpcrepo "github.com/jrjohn/arcana-cloud-go/internal/controller/grpc/repository"
	grpcsvc "github.com/jrjohn/arcana-cloud-go/internal/controller/grpc/service"
	"github.com/jrjohn/arcana-cloud-go/internal/domain/repository"
	repograpc "github.com/jrjohn/arcana-cloud-go/internal/domain/repository/grpc"
	"github.com/jrjohn/arcana-cloud-go/internal/domain/service"
	grpcclient "github.com/jrjohn/arcana-cloud-go/internal/grpc/client"
	"github.com/jrjohn/arcana-cloud-go/internal/security"
)

// GRPCLayeredModule provides layered gRPC architecture dependencies
var GRPCLayeredModule = fx.Module("grpc_layered",
	fx.Provide(provideGRPCClientConfig),
	fx.Provide(provideRepositoryGRPCClient),
	fx.Provide(provideServiceGRPCClient),
	fx.Provide(provideGRPCUserRepository),
	fx.Invoke(registerRepositoryLayerGRPCServices),
	fx.Invoke(registerServiceLayerGRPCServices),
	fx.Invoke(startRepositoryLayerGRPCServer),
)

// GRPCClientConfigs holds configs for connecting to other layers
type GRPCClientConfigs struct {
	RepositoryHost string
	RepositoryPort int
	ServiceHost    string
	ServicePort    int
}

func provideGRPCClientConfig() *GRPCClientConfigs {
	return &GRPCClientConfigs{
		RepositoryHost: getEnvOrDefault("REPOSITORY_GRPC_HOST", "localhost"),
		RepositoryPort: getEnvOrDefaultInt("REPOSITORY_GRPC_PORT", 9090),
		ServiceHost:    getEnvOrDefault("SERVICE_GRPC_HOST", "localhost"),
		ServicePort:    getEnvOrDefaultInt("SERVICE_GRPC_PORT", 9091),
	}
}

// RepositoryGRPCClient wraps the gRPC client for repository layer
type RepositoryGRPCClient struct {
	Client *grpcclient.GRPCClient
}

// ServiceGRPCClient wraps the gRPC client for service layer
type ServiceGRPCClient struct {
	Client *grpcclient.GRPCClient
}

func provideRepositoryGRPCClient(
	configs *GRPCClientConfigs,
	deployConfig *config.DeploymentConfig,
	logger *zap.Logger,
) (*RepositoryGRPCClient, error) {
	// Only create client if we're NOT the repository layer (i.e., we need to call it)
	if deployConfig.IsRepositoryLayer() {
		return &RepositoryGRPCClient{}, nil
	}

	clientConfig := grpcclient.DefaultClientConfig(configs.RepositoryHost, configs.RepositoryPort)
	client, err := grpcclient.NewGRPCClient(clientConfig, logger.Named("repo-grpc-client"))
	if err != nil {
		return nil, err
	}

	return &RepositoryGRPCClient{Client: client}, nil
}

func provideServiceGRPCClient(
	configs *GRPCClientConfigs,
	deployConfig *config.DeploymentConfig,
	logger *zap.Logger,
) (*ServiceGRPCClient, error) {
	// Only create client if we're the controller layer (i.e., we need to call service layer)
	if !deployConfig.IsControllerLayer() || deployConfig.IsServiceLayer() {
		return &ServiceGRPCClient{}, nil
	}

	clientConfig := grpcclient.DefaultClientConfig(configs.ServiceHost, configs.ServicePort)
	client, err := grpcclient.NewGRPCClient(clientConfig, logger.Named("svc-grpc-client"))
	if err != nil {
		return nil, err
	}

	return &ServiceGRPCClient{Client: client}, nil
}

// GRPCUserRepository is a tagged type for gRPC-backed user repository
type GRPCUserRepository struct {
	fx.Out
	Repository repository.UserRepository `name:"grpc_user_repository"`
}

func provideGRPCUserRepository(
	repoClient *RepositoryGRPCClient,
	deployConfig *config.DeploymentConfig,
) GRPCUserRepository {
	// If we are the repository layer, we don't need the gRPC repository
	if deployConfig.IsRepositoryLayer() {
		return GRPCUserRepository{}
	}

	if repoClient.Client == nil {
		return GRPCUserRepository{}
	}

	return GRPCUserRepository{
		Repository: repograpc.NewUserRepositoryGRPC(repoClient.Client.UserServiceClient()),
	}
}

// Repository Layer gRPC Services
type RepositoryLayerGRPCServices struct {
	fx.In

	UserRepo repository.UserRepository
	Logger   *zap.Logger
}

func registerRepositoryLayerGRPCServices(
	server *grpcctrl.Server,
	params RepositoryLayerGRPCServices,
	deployConfig *config.DeploymentConfig,
) {
	if !deployConfig.IsRepositoryLayer() || !deployConfig.IsGRPC() {
		return
	}

	// Register UserService for repository layer
	userService := grpcrepo.NewUserServiceServer(params.UserRepo, params.Logger.Named("grpc-user"))
	pb.RegisterUserServiceServer(server.GetServer(), userService)
}

// Service Layer gRPC Services
type ServiceLayerGRPCServices struct {
	fx.In

	AuthService service.AuthService
	UserService service.UserService
	JWTProvider *security.JWTProvider
	Logger      *zap.Logger
}

func registerServiceLayerGRPCServices(
	server *grpcctrl.Server,
	params ServiceLayerGRPCServices,
	deployConfig *config.DeploymentConfig,
) {
	if !deployConfig.IsServiceLayer() || !deployConfig.IsGRPC() {
		return
	}

	// Register AuthService for service layer
	authService := grpcsvc.NewAuthServiceServer(
		params.AuthService,
		params.JWTProvider,
		params.Logger.Named("grpc-auth"),
	)
	pb.RegisterAuthServiceServer(server.GetServer(), authService)

	// Register UserService for service layer
	userService := grpcsvc.NewUserServiceServer(
		params.UserService,
		params.Logger.Named("grpc-user"),
	)
	pb.RegisterUserServiceServer(server.GetServer(), userService)
}

func startRepositoryLayerGRPCServer(
	lc fx.Lifecycle,
	server *grpcctrl.Server,
	cfg *config.DeploymentConfig,
	logger *zap.Logger,
) {
	if !cfg.IsRepositoryLayer() || !cfg.IsGRPC() {
		return
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				logger.Info("Starting Repository Layer gRPC server")
				if err := server.Start(); err != nil {
					logger.Error("Repository gRPC server error", zap.Error(err))
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			server.Stop()
			return nil
		},
	})
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvOrDefaultInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
