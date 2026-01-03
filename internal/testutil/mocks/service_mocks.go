package mocks

import (
	"context"
	"errors"
	"io"
	"time"

	"github.com/jrjohn/arcana-cloud-go/internal/domain/entity"
	"github.com/jrjohn/arcana-cloud-go/internal/dto/request"
	"github.com/jrjohn/arcana-cloud-go/internal/dto/response"
	"github.com/jrjohn/arcana-cloud-go/internal/jobs"
)

// MockAuthService is a mock implementation of AuthService
type MockAuthService struct {
	RegisterFunc     func(ctx context.Context, req *request.RegisterRequest) (*response.AuthResponse, error)
	LoginFunc        func(ctx context.Context, req *request.LoginRequest) (*response.AuthResponse, error)
	RefreshTokenFunc func(ctx context.Context, req *request.RefreshTokenRequest) (*response.AuthResponse, error)
	LogoutFunc       func(ctx context.Context, token string) error
	LogoutAllFunc    func(ctx context.Context, userID uint) error
}

func NewMockAuthService() *MockAuthService {
	return &MockAuthService{}
}

func (m *MockAuthService) Register(ctx context.Context, req *request.RegisterRequest) (*response.AuthResponse, error) {
	if m.RegisterFunc != nil {
		return m.RegisterFunc(ctx, req)
	}
	return &response.AuthResponse{
		AccessToken:  "mock-access-token",
		RefreshToken: "mock-refresh-token",
		TokenType:    "Bearer",
		ExpiresIn:    3600,
		User: response.UserResponse{
			ID:       1,
			Username: req.Username,
			Email:    req.Email,
		},
	}, nil
}

func (m *MockAuthService) Login(ctx context.Context, req *request.LoginRequest) (*response.AuthResponse, error) {
	if m.LoginFunc != nil {
		return m.LoginFunc(ctx, req)
	}
	return &response.AuthResponse{
		AccessToken:  "mock-access-token",
		RefreshToken: "mock-refresh-token",
		TokenType:    "Bearer",
		ExpiresIn:    3600,
		User: response.UserResponse{
			ID:       1,
			Username: "testuser",
			Email:    "test@example.com",
		},
	}, nil
}

func (m *MockAuthService) RefreshToken(ctx context.Context, req *request.RefreshTokenRequest) (*response.AuthResponse, error) {
	if m.RefreshTokenFunc != nil {
		return m.RefreshTokenFunc(ctx, req)
	}
	return &response.AuthResponse{
		AccessToken:  "mock-new-access-token",
		RefreshToken: "mock-new-refresh-token",
		TokenType:    "Bearer",
		ExpiresIn:    3600,
	}, nil
}

func (m *MockAuthService) Logout(ctx context.Context, token string) error {
	if m.LogoutFunc != nil {
		return m.LogoutFunc(ctx, token)
	}
	return nil
}

func (m *MockAuthService) LogoutAll(ctx context.Context, userID uint) error {
	if m.LogoutAllFunc != nil {
		return m.LogoutAllFunc(ctx, userID)
	}
	return nil
}

// MockUserService is a mock implementation of UserService
type MockUserService struct {
	GetByIDFunc         func(ctx context.Context, id uint) (*response.UserResponse, error)
	GetByUsernameFunc   func(ctx context.Context, username string) (*response.UserResponse, error)
	GetByEmailFunc      func(ctx context.Context, email string) (*response.UserResponse, error)
	ListFunc            func(ctx context.Context, page, size int) (*response.PagedResponse[response.UserResponse], error)
	UpdateFunc          func(ctx context.Context, id uint, req *request.UpdateProfileRequest) (*response.UserResponse, error)
	ChangePasswordFunc  func(ctx context.Context, id uint, req *request.ChangePasswordRequest) error
	DeleteFunc          func(ctx context.Context, id uint) error
	ExistsByUsernameFunc func(ctx context.Context, username string) (bool, error)
	ExistsByEmailFunc   func(ctx context.Context, email string) (bool, error)
}

func NewMockUserService() *MockUserService {
	return &MockUserService{}
}

func (m *MockUserService) GetByID(ctx context.Context, id uint) (*response.UserResponse, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return &response.UserResponse{
		ID:       id,
		Username: "testuser",
		Email:    "test@example.com",
	}, nil
}

func (m *MockUserService) GetByUsername(ctx context.Context, username string) (*response.UserResponse, error) {
	if m.GetByUsernameFunc != nil {
		return m.GetByUsernameFunc(ctx, username)
	}
	return &response.UserResponse{
		ID:       1,
		Username: username,
		Email:    "test@example.com",
	}, nil
}

func (m *MockUserService) GetByEmail(ctx context.Context, email string) (*response.UserResponse, error) {
	if m.GetByEmailFunc != nil {
		return m.GetByEmailFunc(ctx, email)
	}
	return &response.UserResponse{
		ID:       1,
		Username: "testuser",
		Email:    email,
	}, nil
}

func (m *MockUserService) List(ctx context.Context, page, size int) (*response.PagedResponse[response.UserResponse], error) {
	if m.ListFunc != nil {
		return m.ListFunc(ctx, page, size)
	}
	resp := response.NewPagedResponse([]response.UserResponse{
		{ID: 1, Username: "user1"},
		{ID: 2, Username: "user2"},
	}, page, size, 2)
	return &resp, nil
}

func (m *MockUserService) Update(ctx context.Context, id uint, req *request.UpdateProfileRequest) (*response.UserResponse, error) {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, id, req)
	}
	return &response.UserResponse{
		ID:        id,
		Username:  "testuser",
		Email:     req.Email,
		FirstName: req.FirstName,
		LastName:  req.LastName,
	}, nil
}

func (m *MockUserService) ChangePassword(ctx context.Context, id uint, req *request.ChangePasswordRequest) error {
	if m.ChangePasswordFunc != nil {
		return m.ChangePasswordFunc(ctx, id, req)
	}
	return nil
}

func (m *MockUserService) Delete(ctx context.Context, id uint) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, id)
	}
	return nil
}

func (m *MockUserService) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	if m.ExistsByUsernameFunc != nil {
		return m.ExistsByUsernameFunc(ctx, username)
	}
	return false, nil
}

func (m *MockUserService) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	if m.ExistsByEmailFunc != nil {
		return m.ExistsByEmailFunc(ctx, email)
	}
	return false, nil
}

// MockPluginService is a mock implementation of PluginService
type MockPluginService struct {
	InstallFunc         func(ctx context.Context, req *request.InstallPluginRequest, file io.Reader) (*response.PluginResponse, error)
	InstallFromPathFunc func(ctx context.Context, req *request.InstallPluginRequest, filePath string) (*response.PluginResponse, error)
	GetByKeyFunc        func(ctx context.Context, key string) (*response.PluginDetailResponse, error)
	ListFunc            func(ctx context.Context, page, size int) (*response.PagedResponse[response.PluginResponse], error)
	EnableFunc          func(ctx context.Context, key string) (*response.PluginResponse, error)
	DisableFunc         func(ctx context.Context, key string) (*response.PluginResponse, error)
	UninstallFunc       func(ctx context.Context, key string) error
	GetHealthFunc       func(ctx context.Context) (*response.PluginHealthResponse, error)
}

func NewMockPluginService() *MockPluginService {
	return &MockPluginService{}
}

func (m *MockPluginService) Install(ctx context.Context, req *request.InstallPluginRequest, file io.Reader) (*response.PluginResponse, error) {
	if m.InstallFunc != nil {
		return m.InstallFunc(ctx, req, file)
	}
	return &response.PluginResponse{
		ID:      1,
		Key:     "test-plugin",
		Name:    req.Name,
		Version: req.Version,
		State:   string(entity.PluginStateInstalled),
	}, nil
}

func (m *MockPluginService) InstallFromPath(ctx context.Context, req *request.InstallPluginRequest, filePath string) (*response.PluginResponse, error) {
	if m.InstallFromPathFunc != nil {
		return m.InstallFromPathFunc(ctx, req, filePath)
	}
	return &response.PluginResponse{
		ID:      1,
		Key:     "test-plugin",
		Name:    req.Name,
		Version: req.Version,
		State:   string(entity.PluginStateInstalled),
	}, nil
}

func (m *MockPluginService) GetByKey(ctx context.Context, key string) (*response.PluginDetailResponse, error) {
	if m.GetByKeyFunc != nil {
		return m.GetByKeyFunc(ctx, key)
	}
	return &response.PluginDetailResponse{
		PluginResponse: response.PluginResponse{
			ID:    1,
			Key:   key,
			Name:  "Test Plugin",
			State: string(entity.PluginStateInstalled),
		},
	}, nil
}

func (m *MockPluginService) List(ctx context.Context, page, size int) (*response.PagedResponse[response.PluginResponse], error) {
	if m.ListFunc != nil {
		return m.ListFunc(ctx, page, size)
	}
	resp := response.NewPagedResponse([]response.PluginResponse{
		{ID: 1, Key: "plugin-1", Name: "Plugin 1"},
		{ID: 2, Key: "plugin-2", Name: "Plugin 2"},
	}, page, size, 2)
	return &resp, nil
}

func (m *MockPluginService) Enable(ctx context.Context, key string) (*response.PluginResponse, error) {
	if m.EnableFunc != nil {
		return m.EnableFunc(ctx, key)
	}
	return &response.PluginResponse{
		ID:    1,
		Key:   key,
		State: string(entity.PluginStateEnabled),
	}, nil
}

func (m *MockPluginService) Disable(ctx context.Context, key string) (*response.PluginResponse, error) {
	if m.DisableFunc != nil {
		return m.DisableFunc(ctx, key)
	}
	return &response.PluginResponse{
		ID:    1,
		Key:   key,
		State: string(entity.PluginStateDisabled),
	}, nil
}

func (m *MockPluginService) Uninstall(ctx context.Context, key string) error {
	if m.UninstallFunc != nil {
		return m.UninstallFunc(ctx, key)
	}
	return nil
}

func (m *MockPluginService) GetHealth(ctx context.Context) (*response.PluginHealthResponse, error) {
	if m.GetHealthFunc != nil {
		return m.GetHealthFunc(ctx)
	}
	return &response.PluginHealthResponse{
		Status:          "healthy",
		TotalPlugins:    5,
		EnabledPlugins:  3,
		DisabledPlugins: 2,
		ErrorPlugins:    0,
	}, nil
}

// MockJobService is a mock implementation of jobs.Service
type MockJobService struct {
	EnqueueFunc       func(ctx context.Context, jobType string, payload any, opts ...jobs.JobOption) (string, error)
	EnqueueAtFunc     func(ctx context.Context, jobType string, payload any, scheduledAt time.Time, opts ...jobs.JobOption) (string, error)
	EnqueueInFunc     func(ctx context.Context, jobType string, payload any, delay time.Duration, opts ...jobs.JobOption) (string, error)
	GetJobFunc        func(ctx context.Context, jobID string) (*jobs.JobPayload, error)
	CancelJobFunc     func(ctx context.Context, jobID string) error
	RetryJobFunc      func(ctx context.Context, jobID string) error
	GetQueueStatsFunc func(ctx context.Context) (*jobs.QueueStats, error)
	GetDLQJobsFunc    func(ctx context.Context, limit int) ([]*jobs.JobPayload, error)
	RetryDLQJobFunc   func(ctx context.Context, jobID string) error
	PurgeDLQFunc      func(ctx context.Context) error
}

func NewMockJobService() *MockJobService {
	return &MockJobService{}
}

func (m *MockJobService) Enqueue(ctx context.Context, jobType string, payload any, opts ...jobs.JobOption) (string, error) {
	if m.EnqueueFunc != nil {
		return m.EnqueueFunc(ctx, jobType, payload, opts...)
	}
	return "job-12345", nil
}

func (m *MockJobService) EnqueueAt(ctx context.Context, jobType string, payload any, scheduledAt time.Time, opts ...jobs.JobOption) (string, error) {
	if m.EnqueueAtFunc != nil {
		return m.EnqueueAtFunc(ctx, jobType, payload, scheduledAt, opts...)
	}
	return "job-12345", nil
}

func (m *MockJobService) EnqueueIn(ctx context.Context, jobType string, payload any, delay time.Duration, opts ...jobs.JobOption) (string, error) {
	if m.EnqueueInFunc != nil {
		return m.EnqueueInFunc(ctx, jobType, payload, delay, opts...)
	}
	return "job-12345", nil
}

func (m *MockJobService) GetJob(ctx context.Context, jobID string) (*jobs.JobPayload, error) {
	if m.GetJobFunc != nil {
		return m.GetJobFunc(ctx, jobID)
	}
	now := time.Now()
	return &jobs.JobPayload{
		ID:        jobID,
		Type:      "test-job",
		Status:    jobs.JobStatusPending,
		Priority:  jobs.PriorityNormal,
		CreatedAt: now,
	}, nil
}

func (m *MockJobService) CancelJob(ctx context.Context, jobID string) error {
	if m.CancelJobFunc != nil {
		return m.CancelJobFunc(ctx, jobID)
	}
	return nil
}

func (m *MockJobService) RetryJob(ctx context.Context, jobID string) error {
	if m.RetryJobFunc != nil {
		return m.RetryJobFunc(ctx, jobID)
	}
	return nil
}

func (m *MockJobService) GetQueueStats(ctx context.Context) (*jobs.QueueStats, error) {
	if m.GetQueueStatsFunc != nil {
		return m.GetQueueStatsFunc(ctx)
	}
	return &jobs.QueueStats{
		Pending:    10,
		Scheduled:  5,
		Completed:  100,
		Failed:     2,
		Dead:       1,
		QueueSizes: map[string]int64{"default": 10},
		WorkerStats: jobs.WorkerStats{
			Running:       true,
			ActiveWorkers: 4,
			Concurrency:   8,
		},
	}, nil
}

func (m *MockJobService) GetDLQJobs(ctx context.Context, limit int) ([]*jobs.JobPayload, error) {
	if m.GetDLQJobsFunc != nil {
		return m.GetDLQJobsFunc(ctx, limit)
	}
	return []*jobs.JobPayload{}, nil
}

func (m *MockJobService) RetryDLQJob(ctx context.Context, jobID string) error {
	if m.RetryDLQJobFunc != nil {
		return m.RetryDLQJobFunc(ctx, jobID)
	}
	return nil
}

func (m *MockJobService) PurgeDLQ(ctx context.Context) error {
	if m.PurgeDLQFunc != nil {
		return m.PurgeDLQFunc(ctx)
	}
	return nil
}

// Common errors for testing
var (
	ErrMockNotFound      = errors.New("not found")
	ErrMockUnauthorized  = errors.New("unauthorized")
	ErrMockInternalError = errors.New("internal error")
)
