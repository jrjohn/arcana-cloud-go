package handler

import (
	"context"
	"testing"

	"go.uber.org/zap"

	"github.com/jrjohn/arcana-cloud-go/internal/jobs"
	"github.com/jrjohn/arcana-cloud-go/internal/jobs/worker"
)

// mockQueue is a no-op Queue for unit testing without Redis
type mockQueue struct{}

func (m *mockQueue) Enqueue(ctx context.Context, job *jobs.JobPayload) error { return nil }
func (m *mockQueue) Dequeue(ctx context.Context, priorities ...jobs.Priority) (*jobs.JobPayload, error) {
	return nil, nil
}
func (m *mockQueue) GetJob(ctx context.Context, jobID string) (*jobs.JobPayload, error) {
	return nil, nil
}
func (m *mockQueue) UpdateJob(ctx context.Context, job *jobs.JobPayload) error  { return nil }
func (m *mockQueue) Complete(ctx context.Context, jobID string) error            { return nil }
func (m *mockQueue) Fail(ctx context.Context, jobID string, jobErr error) error  { return nil }
func (m *mockQueue) ProcessScheduled(ctx context.Context) (int, error)           { return 0, nil }
func (m *mockQueue) GetDLQJobs(ctx context.Context, limit int64) ([]*jobs.JobPayload, error) {
	return nil, nil
}
func (m *mockQueue) RetryDLQJob(ctx context.Context, jobID string) error        { return nil }
func (m *mockQueue) DeleteJob(ctx context.Context, jobID string) error           { return nil }
func (m *mockQueue) RequeueJob(ctx context.Context, jobID string, queueKey string) error { return nil }
func (m *mockQueue) GetStats(ctx context.Context) (map[string]int64, error) {
	return map[string]int64{}, nil
}

func newTestRegistry(t *testing.T) *Registry {
	t.Helper()
	logger, _ := zap.NewDevelopment()
	q := &mockQueue{}
	poolConfig := worker.DefaultWorkerPoolConfig()
	pool := worker.NewWorkerPool(q, logger, poolConfig)
	return NewRegistry(pool, logger)
}

func TestNewRegistry_Unit(t *testing.T) {
	r := newTestRegistry(t)
	if r == nil {
		t.Fatal("NewRegistry returned nil")
	}
	if r.types == nil {
		t.Error("types map is nil")
	}
	if r.pool == nil {
		t.Error("pool is nil")
	}
}

func TestRegister_Unit_StoresType(t *testing.T) {
	r := newTestRegistry(t)

	type MyPayload struct {
		Name string `json:"name"`
	}

	Register(r, "my-job", func(ctx context.Context, p MyPayload) error {
		return nil
	})

	handlers := r.ListHandlers()
	if len(handlers) != 1 {
		t.Fatalf("len(handlers) = %d, want 1", len(handlers))
	}
	if _, ok := handlers["my-job"]; !ok {
		t.Error("my-job not found in handlers")
	}
}

func TestRegister_Unit_MultipleHandlers(t *testing.T) {
	r := newTestRegistry(t)

	type P1 struct{ A string }
	type P2 struct{ B int }

	Register(r, "job-a", func(ctx context.Context, p P1) error { return nil })
	Register(r, "job-b", func(ctx context.Context, p P2) error { return nil })

	handlers := r.ListHandlers()
	if len(handlers) != 2 {
		t.Fatalf("len(handlers) = %d, want 2", len(handlers))
	}
}

func TestListHandlers_Unit_ReturnsCopy(t *testing.T) {
	r := newTestRegistry(t)

	type P struct{ X int }
	Register(r, "copy-test", func(ctx context.Context, p P) error { return nil })

	h1 := r.ListHandlers()
	h1["injected"] = "evil"

	h2 := r.ListHandlers()
	if _, ok := h2["injected"]; ok {
		t.Error("ListHandlers should return a copy, not the underlying map")
	}
	if len(h2) != 1 {
		t.Errorf("len(h2) = %d, want 1", len(h2))
	}
}

func TestListHandlers_Unit_Empty(t *testing.T) {
	r := newTestRegistry(t)
	handlers := r.ListHandlers()
	if len(handlers) != 0 {
		t.Errorf("len(handlers) = %d, want 0", len(handlers))
	}
}

func TestRegister_Unit_OverwriteHandler(t *testing.T) {
	r := newTestRegistry(t)

	type P struct{ V int }
	Register(r, "overwrite", func(ctx context.Context, p P) error { return nil })
	Register(r, "overwrite", func(ctx context.Context, p P) error { return nil })

	handlers := r.ListHandlers()
	if len(handlers) != 1 {
		t.Errorf("len(handlers) = %d, want 1 (overwrite)", len(handlers))
	}
}
