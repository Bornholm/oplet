package runner

import (
	"fmt"
	"time"

	"github.com/bornholm/oplet/internal/store"
)

// Heartbeat Models
type HeartbeatResponse struct {
	ID          uint      `json:"id"`
	Name        string    `json:"name"`
	ContactedAt time.Time `json:"contacted_at"`
}

// Task Request Models
type TaskRequestResponse struct {
	ExecutionID     uint              `json:"execution_id"`
	TaskID          uint              `json:"task_id"`
	ImageRef        string            `json:"image_ref"`
	Environment     map[string]string `json:"environment"`
	InputParameters string            `json:"input_parameters"`
	RunnerToken     string            `json:"runner_token"`
	InputsDir       string            `json:"inputs_dir"`
	OutputsDir      string            `json:"outputs_dir"`
	CreatedAt       time.Time         `json:"created_at"`
}

// Task Status Models
type TaskStatusRequest struct {
	Status      store.TaskExecutionStatus `json:"status" validate:"required"`
	ContainerID string                    `json:"container_id,omitempty"`
	ExitCode    *int                      `json:"exit_code,omitempty"`
	Error       string                    `json:"error,omitempty"`
	StartedAt   *time.Time                `json:"started_at,omitempty"`
	FinishedAt  *time.Time                `json:"finished_at,omitempty"`
	Timestamp   int64                     `json:"timestamp" validate:"required"`
}

type TaskStatusResponse struct {
	ExecutionID uint                      `json:"execution_id"`
	Status      store.TaskExecutionStatus `json:"status"`
	UpdatedAt   time.Time                 `json:"updated_at"`
}

// Task Log Models
type LogEntry struct {
	Timestamp int64  `json:"timestamp" validate:"required"`
	Source    string `json:"source" validate:"required,oneof=container system"`
	Message   string `json:"message" validate:"required"`
	Clock     uint   `json:"clock" validate:"required"`
}

type TaskTraceRequest struct {
	Logs []LogEntry `json:"logs" validate:"required,dive"`
}

type TaskTraceResponse struct {
	ExecutionID uint `json:"execution_id"`
	LogsAdded   int  `json:"logs_added"`
}

// Task Input Models
type TaskInputsResponse struct {
	ExecutionID uint   `json:"execution_id"`
	FilesStored int    `json:"files_stored"`
	Message     string `json:"message"`
}

// Task Output Models
type TaskOutputsResponse struct {
	ExecutionID uint   `json:"execution_id"`
	FilesStored int    `json:"files_stored"`
	Message     string `json:"message"`
}

// Error Response Models
type ErrorResponse struct {
	Error   string            `json:"error"`
	Code    string            `json:"code,omitempty"`
	Details map[string]string `json:"details,omitempty"`
}

// Success Response Models
type SuccessResponse struct {
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Error helper functions
func ErrInvalidRequest(format string, args ...interface{}) error {
	return fmt.Errorf("invalid request: "+format, args...)
}

func ErrNotFound(resource string) error {
	return fmt.Errorf("%s not found", resource)
}

func ErrUnauthorized(message string) error {
	return fmt.Errorf("unauthorized: %s", message)
}

// Validation helper functions
func (r *TaskStatusRequest) Validate() error {
	if r.Status == "" {
		return ErrInvalidRequest("status is required")
	}
	return nil
}

func (r *TaskTraceRequest) Validate() error {
	if len(r.Logs) == 0 {
		return ErrInvalidRequest("logs are required")
	}
	for i, log := range r.Logs {
		if log.Source == "" {
			return ErrInvalidRequest("log source is required for entry %d", i)
		}
		if log.Source != "container" && log.Source != "system" {
			return ErrInvalidRequest("log source must be 'container' or 'system' for entry %d", i)
		}
		if log.Message == "" {
			return ErrInvalidRequest("log message is required for entry %d", i)
		}
		if log.Timestamp <= 0 {
			return ErrInvalidRequest("log timestamp is required for entry %d", i)
		}
	}
	return nil
}
