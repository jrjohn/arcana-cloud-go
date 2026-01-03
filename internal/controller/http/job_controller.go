package http

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/jrjohn/arcana-cloud-go/internal/dto/request"
	"github.com/jrjohn/arcana-cloud-go/internal/dto/response"
	"github.com/jrjohn/arcana-cloud-go/internal/jobs"
	"github.com/jrjohn/arcana-cloud-go/internal/jobs/scheduler"
	"github.com/jrjohn/arcana-cloud-go/internal/middleware"
)

// JobController handles job management endpoints
type JobController struct {
	jobService     jobs.Service
	scheduler      *scheduler.Scheduler
	authMiddleware *middleware.AuthMiddleware
}

// NewJobController creates a new JobController instance
func NewJobController(
	jobService jobs.Service,
	scheduler *scheduler.Scheduler,
	authMiddleware *middleware.AuthMiddleware,
) *JobController {
	return &JobController{
		jobService:     jobService,
		scheduler:      scheduler,
		authMiddleware: authMiddleware,
	}
}

// RegisterRoutes registers the job routes
func (c *JobController) RegisterRoutes(router *gin.RouterGroup) {
	jobRoutes := router.Group("/jobs")
	{
		// Public health/stats endpoints
		jobRoutes.GET("/queues", c.GetQueueStats)
		jobRoutes.GET("/dashboard", c.GetDashboard)

		// Protected endpoints
		protected := jobRoutes.Group("")
		protected.Use(c.authMiddleware.Authenticate())
		{
			// Job management
			protected.POST("", c.EnqueueJob)
			protected.GET("/:id", c.GetJob)
			protected.DELETE("/:id", c.CancelJob)
			protected.POST("/:id/retry", c.RetryJob)

			// DLQ management
			protected.GET("/dlq", c.GetDLQJobs)
			protected.POST("/dlq/:id/retry", c.RetryDLQJob)
			protected.DELETE("/dlq", c.authMiddleware.RequireAdmin(), c.PurgeDLQ)

			// Scheduled jobs
			protected.GET("/scheduled", c.GetScheduledJobs)
		}
	}
}

// EnqueueJob adds a new job to the queue
// @Summary Enqueue a new job
// @Tags Jobs
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body request.EnqueueJobRequest true "Job request"
// @Success 201 {object} response.ApiResponse[response.JobEnqueueResponse]
// @Router /api/v1/jobs [post]
func (c *JobController) EnqueueJob(ctx *gin.Context) {
	var req request.EnqueueJobRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, response.NewErrorWithDetails[any]("validation failed", err.Error()))
		return
	}

	// Build options
	var opts []jobs.JobOption

	// Parse priority
	switch strings.ToLower(req.Priority) {
	case "low":
		opts = append(opts, jobs.WithPriority(jobs.PriorityLow))
	case "high":
		opts = append(opts, jobs.WithPriority(jobs.PriorityHigh))
	case "critical":
		opts = append(opts, jobs.WithPriority(jobs.PriorityCritical))
	default:
		opts = append(opts, jobs.WithPriority(jobs.PriorityNormal))
	}

	// Handle scheduling
	if req.ScheduledAt != "" {
		scheduledAt, err := time.Parse(time.RFC3339, req.ScheduledAt)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, response.NewError[any]("invalid scheduled_at format, use RFC3339"))
			return
		}
		opts = append(opts, jobs.WithScheduledAt(scheduledAt))
	} else if req.DelaySeconds > 0 {
		opts = append(opts, jobs.WithDelay(time.Duration(req.DelaySeconds)*time.Second))
	}

	if req.UniqueKey != "" {
		opts = append(opts, jobs.WithUniqueKey(req.UniqueKey))
	}

	if len(req.Tags) > 0 {
		opts = append(opts, jobs.WithTags(req.Tags...))
	}

	// Unmarshal payload to verify it's valid JSON
	var payload any
	if err := json.Unmarshal(req.Payload, &payload); err != nil {
		ctx.JSON(http.StatusBadRequest, response.NewError[any]("invalid payload JSON"))
		return
	}

	jobID, err := c.jobService.Enqueue(ctx.Request.Context(), req.Type, payload, opts...)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, response.NewError[any]("failed to enqueue job"))
		return
	}

	ctx.JSON(http.StatusCreated, response.NewSuccess(response.JobEnqueueResponse{
		JobID:   jobID,
		Message: "Job enqueued successfully",
	}, "Job enqueued"))
}

// GetJob retrieves a job by ID
// @Summary Get job by ID
// @Tags Jobs
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Job ID"
// @Success 200 {object} response.ApiResponse[response.JobResponse]
// @Router /api/v1/jobs/{id} [get]
func (c *JobController) GetJob(ctx *gin.Context) {
	jobID := ctx.Param("id")
	if jobID == "" {
		ctx.JSON(http.StatusBadRequest, response.NewError[any]("job ID required"))
		return
	}

	job, err := c.jobService.GetJob(ctx.Request.Context(), jobID)
	if err != nil {
		ctx.JSON(http.StatusNotFound, response.NewError[any]("job not found"))
		return
	}

	ctx.JSON(http.StatusOK, response.NewSuccessWithData(c.toJobResponse(job)))
}

// CancelJob cancels a pending job
// @Summary Cancel a job
// @Tags Jobs
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Job ID"
// @Success 200 {object} response.ApiResponse[any]
// @Router /api/v1/jobs/{id} [delete]
func (c *JobController) CancelJob(ctx *gin.Context) {
	jobID := ctx.Param("id")
	if jobID == "" {
		ctx.JSON(http.StatusBadRequest, response.NewError[any]("job ID required"))
		return
	}

	if err := c.jobService.CancelJob(ctx.Request.Context(), jobID); err != nil {
		ctx.JSON(http.StatusInternalServerError, response.NewError[any]("failed to cancel job"))
		return
	}

	ctx.JSON(http.StatusOK, response.NewSuccess[any](nil, "Job cancelled"))
}

// RetryJob retries a failed job
// @Summary Retry a job
// @Tags Jobs
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Job ID"
// @Success 200 {object} response.ApiResponse[any]
// @Router /api/v1/jobs/{id}/retry [post]
func (c *JobController) RetryJob(ctx *gin.Context) {
	jobID := ctx.Param("id")
	if jobID == "" {
		ctx.JSON(http.StatusBadRequest, response.NewError[any]("job ID required"))
		return
	}

	if err := c.jobService.RetryJob(ctx.Request.Context(), jobID); err != nil {
		ctx.JSON(http.StatusInternalServerError, response.NewError[any]("failed to retry job"))
		return
	}

	ctx.JSON(http.StatusOK, response.NewSuccess[any](nil, "Job retry initiated"))
}

// GetQueueStats returns queue statistics
// @Summary Get queue statistics
// @Tags Jobs
// @Accept json
// @Produce json
// @Success 200 {object} response.ApiResponse[response.QueueStatsResponse]
// @Router /api/v1/jobs/queues [get]
func (c *JobController) GetQueueStats(ctx *gin.Context) {
	stats, err := c.jobService.GetQueueStats(ctx.Request.Context())
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, response.NewError[any]("failed to get queue stats"))
		return
	}

	resp := response.QueueStatsResponse{
		Pending:    stats.Pending,
		Scheduled:  stats.Scheduled,
		Completed:  stats.Completed,
		Failed:     stats.Failed,
		Dead:       stats.Dead,
		QueueSizes: stats.QueueSizes,
		WorkerStats: response.WorkerStatsResponse{
			Running:       stats.WorkerStats.Running,
			ActiveWorkers: stats.WorkerStats.ActiveWorkers,
			Concurrency:   stats.WorkerStats.Concurrency,
			ProcessedJobs: stats.WorkerStats.ProcessedJobs,
			FailedJobs:    stats.WorkerStats.FailedJobs,
		},
		SchedulerStats: response.SchedulerStatsResponse{
			IsLeader:          stats.SchedulerStats.IsLeader,
			RegisteredJobs:    stats.SchedulerStats.RegisteredJobs,
			ScheduledJobNames: stats.SchedulerStats.ScheduledJobNames,
		},
	}

	ctx.JSON(http.StatusOK, response.NewSuccessWithData(resp))
}

// GetDashboard returns a comprehensive dashboard view
// @Summary Get jobs dashboard
// @Tags Jobs
// @Accept json
// @Produce json
// @Success 200 {object} response.ApiResponse[response.QueueStatsResponse]
// @Router /api/v1/jobs/dashboard [get]
func (c *JobController) GetDashboard(ctx *gin.Context) {
	c.GetQueueStats(ctx)
}

// GetDLQJobs returns jobs in the dead letter queue
// @Summary Get DLQ jobs
// @Tags Jobs
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param limit query int false "Limit" default(100)
// @Success 200 {object} response.ApiResponse[[]response.JobResponse]
// @Router /api/v1/jobs/dlq [get]
func (c *JobController) GetDLQJobs(ctx *gin.Context) {
	limit, _ := strconv.Atoi(ctx.DefaultQuery("limit", "100"))
	if limit <= 0 || limit > 1000 {
		limit = 100
	}

	dlqJobs, err := c.jobService.GetDLQJobs(ctx.Request.Context(), limit)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, response.NewError[any]("failed to get DLQ jobs"))
		return
	}

	resp := make([]response.JobResponse, len(dlqJobs))
	for i, job := range dlqJobs {
		resp[i] = *c.toJobResponse(job)
	}

	ctx.JSON(http.StatusOK, response.NewSuccessWithData(resp))
}

// RetryDLQJob retries a job from the DLQ
// @Summary Retry DLQ job
// @Tags Jobs
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Job ID"
// @Success 200 {object} response.ApiResponse[any]
// @Router /api/v1/jobs/dlq/{id}/retry [post]
func (c *JobController) RetryDLQJob(ctx *gin.Context) {
	jobID := ctx.Param("id")
	if jobID == "" {
		ctx.JSON(http.StatusBadRequest, response.NewError[any]("job ID required"))
		return
	}

	if err := c.jobService.RetryDLQJob(ctx.Request.Context(), jobID); err != nil {
		ctx.JSON(http.StatusInternalServerError, response.NewError[any]("failed to retry DLQ job"))
		return
	}

	ctx.JSON(http.StatusOK, response.NewSuccess[any](nil, "DLQ job retry initiated"))
}

// PurgeDLQ removes all jobs from the DLQ
// @Summary Purge DLQ
// @Tags Jobs
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.ApiResponse[any]
// @Router /api/v1/jobs/dlq [delete]
func (c *JobController) PurgeDLQ(ctx *gin.Context) {
	if err := c.jobService.PurgeDLQ(ctx.Request.Context()); err != nil {
		ctx.JSON(http.StatusInternalServerError, response.NewError[any]("failed to purge DLQ"))
		return
	}

	ctx.JSON(http.StatusOK, response.NewSuccess[any](nil, "DLQ purged"))
}

// GetScheduledJobs returns all scheduled cron jobs
// @Summary Get scheduled jobs
// @Tags Jobs
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.ApiResponse[[]response.ScheduledJobResponse]
// @Router /api/v1/jobs/scheduled [get]
func (c *JobController) GetScheduledJobs(ctx *gin.Context) {
	if c.scheduler == nil {
		ctx.JSON(http.StatusOK, response.NewSuccessWithData([]response.ScheduledJobResponse{}))
		return
	}

	scheduledJobs := c.scheduler.ListJobs()
	resp := make([]response.ScheduledJobResponse, len(scheduledJobs))

	for i, job := range scheduledJobs {
		nextRun, _ := c.scheduler.GetNextRun(job.Name)
		resp[i] = response.ScheduledJobResponse{
			Name:     job.Name,
			Schedule: job.Schedule,
			JobType:  job.JobType,
			NextRun:  nextRun,
			Priority: job.Priority,
		}
	}

	ctx.JSON(http.StatusOK, response.NewSuccessWithData(resp))
}

func (c *JobController) toJobResponse(job *jobs.JobPayload) *response.JobResponse {
	return &response.JobResponse{
		ID:            job.ID,
		Type:          job.Type,
		Priority:      job.Priority.String(),
		Status:        string(job.Status),
		Attempts:      job.Attempts,
		MaxRetries:    job.MaxRetries,
		ScheduledAt:   job.ScheduledAt,
		CreatedAt:     job.CreatedAt,
		StartedAt:     job.StartedAt,
		CompletedAt:   job.CompletedAt,
		LastError:     job.LastError,
		CorrelationID: job.CorrelationID,
		Tags:          job.Tags,
	}
}
