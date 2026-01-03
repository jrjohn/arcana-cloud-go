package request

import "encoding/json"

// EnqueueJobRequest represents a job enqueue request
type EnqueueJobRequest struct {
	Type        string          `json:"type" binding:"required"`
	Payload     json.RawMessage `json:"payload" binding:"required"`
	Priority    string          `json:"priority,omitempty"` // low, normal, high, critical
	ScheduledAt string          `json:"scheduled_at,omitempty"` // RFC3339 format
	DelaySeconds int            `json:"delay_seconds,omitempty"`
	UniqueKey   string          `json:"unique_key,omitempty"`
	Tags        []string        `json:"tags,omitempty"`
}

// RetryJobRequest represents a job retry request
type RetryJobRequest struct {
	ResetAttempts bool `json:"reset_attempts,omitempty"`
}
