package runner

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"time"

	"github.com/bornholm/oplet/internal/store"
	"github.com/pkg/errors"
)

// TaskRequestResponse represents the response from the task request endpoint
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

// TaskStatusRequest represents a task status update request
type TaskStatusRequest struct {
	Status      store.TaskExecutionStatus `json:"status"`
	ContainerID string                    `json:"container_id,omitempty"`
	ExitCode    *int                      `json:"exit_code,omitempty"`
	Error       string                    `json:"error,omitempty"`
	StartedAt   *time.Time                `json:"started_at,omitempty"`
	FinishedAt  *time.Time                `json:"finished_at,omitempty"`
	Timestamp   int64                     `json:"timestamp"`
}

// LogEntry represents a log entry
type LogEntry struct {
	Timestamp int64  `json:"timestamp"`
	Source    string `json:"source"`
	Message   string `json:"message"`
	Clock     uint   `json:"clock"`
}

// TaskTraceRequest represents a log submission request
type TaskTraceRequest struct {
	Logs []LogEntry `json:"logs"`
}

// HeartbeatResponse represents the response from heartbeat endpoint
type HeartbeatResponse struct {
	ID          uint      `json:"id"`
	Label       string    `json:"label"`
	ContactedAt time.Time `json:"contacted_at"`
}

// Client provides methods to interact with the runner API
type Client struct {
	serverURL *url.URL
	authToken string
	http      *http.Client
}

// NewClient creates a new runner API client
func NewClient(serverURL, authToken string, httpClient *http.Client) (*Client, error) {
	parsedURL, err := url.Parse(serverURL)
	if err != nil {
		return nil, errors.Wrapf(err, "invalid server URL: %s", serverURL)
	}

	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	return &Client{
		serverURL: parsedURL,
		authToken: authToken,
		http:      httpClient,
	}, nil
}

// SendHeartbeat sends a heartbeat to the server
func (c *Client) SendHeartbeat(ctx context.Context) (*HeartbeatResponse, error) {
	heartbeatURL := c.serverURL.JoinPath("/runner/heartbeat")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, heartbeatURL.String(), nil)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	req.Header.Set("Authorization", "Bearer "+c.authToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("heartbeat failed with status %d", resp.StatusCode)
	}

	var heartbeatResp HeartbeatResponse
	if err := json.NewDecoder(resp.Body).Decode(&heartbeatResp); err != nil {
		return nil, errors.Wrap(err, "failed to decode heartbeat response")
	}

	return &heartbeatResp, nil
}

// RequestTask requests the next available task from the server
func (c *Client) RequestTask(ctx context.Context) (*TaskRequestResponse, error) {
	requestTaskURL := c.serverURL.JoinPath("/runner/request-task")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestTaskURL.String(), nil)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	req.Header.Set("Authorization", "Bearer "+c.authToken)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent {
		return nil, nil // No tasks available
	}

	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("task request failed with status %d", resp.StatusCode)
	}

	var taskResp TaskRequestResponse
	if err := json.NewDecoder(resp.Body).Decode(&taskResp); err != nil {
		return nil, errors.Wrap(err, "failed to decode task response")
	}

	return &taskResp, nil
}

// UpdateTaskStatus updates the status of a task execution
func (c *Client) UpdateTaskStatus(ctx context.Context, taskID uint, statusReq TaskStatusRequest) error {
	statusURL := c.serverURL.JoinPath("/runner/tasks/" + strconv.FormatUint(uint64(taskID), 10) + "/status")

	reqBody, err := json.Marshal(statusReq)
	if err != nil {
		return errors.Wrap(err, "failed to marshal status request")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, statusURL.String(), bytes.NewReader(reqBody))
	if err != nil {
		return errors.WithStack(err)
	}

	req.Header.Set("Authorization", "Bearer "+c.authToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return errors.WithStack(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.Errorf("status update failed with status %d", resp.StatusCode)
	}

	return nil
}

// SubmitLogs submits execution logs to the server
func (c *Client) SubmitLogs(ctx context.Context, taskID uint, logs []LogEntry) error {
	traceURL := c.serverURL.JoinPath("/runner/tasks/" + strconv.FormatUint(uint64(taskID), 10) + "/trace")

	traceReq := TaskTraceRequest{Logs: logs}
	reqBody, err := json.Marshal(traceReq)
	if err != nil {
		return errors.Wrap(err, "failed to marshal trace request")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, traceURL.String(), bytes.NewReader(reqBody))
	if err != nil {
		return errors.WithStack(err)
	}

	req.Header.Set("Authorization", "Bearer "+c.authToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return errors.WithStack(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.Errorf("log submission failed with status %d", resp.StatusCode)
	}

	return nil
}

// ListInputFiles lists available input files for a task
func (c *Client) ListInputFiles(ctx context.Context, taskID uint) ([]map[string]interface{}, error) {
	inputsURL := c.serverURL.JoinPath("/runner/tasks/" + strconv.FormatUint(uint64(taskID), 10) + "/inputs")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, inputsURL.String(), nil)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	req.Header.Set("Authorization", "Bearer "+c.authToken)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("list input files failed with status %d", resp.StatusCode)
	}

	var response struct {
		ExecutionID uint                     `json:"execution_id"`
		Files       []map[string]interface{} `json:"files"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, errors.Wrap(err, "failed to decode input files response")
	}

	return response.Files, nil
}

// DownloadInputFile downloads a specific input file for a task
func (c *Client) DownloadInputFile(ctx context.Context, taskID uint, filename string) (io.ReadCloser, error) {
	inputsURL := c.serverURL.JoinPath("/runner/tasks/" + strconv.FormatUint(uint64(taskID), 10) + "/inputs")

	// Add filename as query parameter
	query := inputsURL.Query()
	query.Set("file", filename)
	inputsURL.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, inputsURL.String(), nil)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	req.Header.Set("Authorization", "Bearer "+c.authToken)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, errors.Errorf("download input file failed with status %d", resp.StatusCode)
	}

	return resp.Body, nil
}

// UploadOutputFiles uploads output files for a task
func (c *Client) UploadOutputFiles(ctx context.Context, taskID uint, files map[string]io.Reader) error {
	outputsURL := c.serverURL.JoinPath("/runner/tasks/" + strconv.FormatUint(uint64(taskID), 10) + "/outputs")

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	for fieldName, fileReader := range files {
		part, err := writer.CreateFormFile(fieldName, filepath.Base(fieldName))
		if err != nil {
			return errors.Wrapf(err, "failed to create form file for %s", fieldName)
		}

		if _, err := io.Copy(part, fileReader); err != nil {
			return errors.Wrapf(err, "failed to copy file content for %s", fieldName)
		}
	}

	if err := writer.Close(); err != nil {
		return errors.Wrap(err, "failed to close multipart writer")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, outputsURL.String(), &buf)
	if err != nil {
		return errors.WithStack(err)
	}

	req.Header.Set("Authorization", "Bearer "+c.authToken)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.http.Do(req)
	if err != nil {
		return errors.WithStack(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.Errorf("output file upload failed with status %d", resp.StatusCode)
	}

	return nil
}
