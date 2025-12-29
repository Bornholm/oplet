# Runner API Documentation

This document describes the complete REST API for the Oplet runner system, which enables distributed task execution across multiple runner instances.

## Overview

The Runner API provides endpoints for:

- Runner heartbeat monitoring
- Task assignment and execution
- Status reporting and logging
- File upload/download for task inputs and outputs

## Authentication

All runner API endpoints require Bearer token authentication:

```
Authorization: Bearer <runner_token>
```

The runner token is obtained when a runner is created via the web UI.

## Base URL

All endpoints are prefixed with `/runner/`

## Endpoints

### 1. Heartbeat

**POST** `/runner/heartbeat`

Sends a heartbeat signal to indicate the runner is alive and active.

#### Request

- **Method**: POST
- **Content-Type**: application/json
- **Body**: Empty (optional metadata can be added in future)

#### Response

```json
{
  "id": 123,
  "label": "runner-001",
  "contacted_at": "2023-12-29T14:30:00Z"
}
```

#### Status Codes

- `200 OK`: Heartbeat received successfully
- `401 Unauthorized`: Invalid or missing runner token
- `500 Internal Server Error`: Server error

---

### 2. Request Task

**GET** `/runner/request-task`

Requests the next available task for execution. This endpoint uses long polling (30 second timeout).

#### Request

- **Method**: GET
- **Headers**: Authorization required

#### Response

**Success (Task Available)**:

```json
{
  "execution_id": 456,
  "task_id": 789,
  "image_ref": "docker.io/myorg/mytask:latest",
  "environment": {
    "INPUT_FILE": "data.txt",
    "OUTPUT_FORMAT": "json"
  },
  "input_parameters": "{\"file\":\"data.txt\",\"format\":\"json\"}",
  "runner_token": "exec_token_abc123",
  "inputs_dir": "/oplet/inputs",
  "outputs_dir": "/oplet/outputs",
  "created_at": "2023-12-29T14:25:00Z"
}
```

**No Tasks Available**:

- **Status**: `204 No Content`
- **Body**: Empty

#### Status Codes

- `200 OK`: Task assigned successfully
- `204 No Content`: No tasks available
- `401 Unauthorized`: Invalid runner token
- `500 Internal Server Error`: Server error

---

### 3. Update Task Status

**POST** `/runner/tasks/{taskID}/status`

Updates the execution status of a task.

#### Request

- **Method**: POST
- **Content-Type**: application/json
- **Path Parameters**:
  - `taskID`: Task ID (integer)

**Body**:

```json
{
  "status": "running",
  "container_id": "container_abc123",
  "exit_code": 0,
  "error": "",
  "started_at": "2023-12-29T14:30:00Z",
  "finished_at": "2023-12-29T14:35:00Z"
}
```

#### Request Fields

- `status` (required): Task execution status
- `container_id` (optional): Docker container ID
- `exit_code` (optional): Container exit code
- `error` (optional): Error message if failed
- `started_at` (optional): Task start timestamp
- `finished_at` (optional): Task completion timestamp

#### Valid Status Values

- `pending`
- `pulling_image`
- `image_pulled`
- `creating_container`
- `container_created`
- `uploading_files`
- `files_uploaded`
- `starting_container`
- `container_started`
- `running`
- `finished`
- `downloading_files`
- `files_downloaded`
- `succeeded`
- `failed`
- `killed`

#### Response

```json
{
  "execution_id": 456,
  "status": "running",
  "updated_at": "2023-12-29T14:30:00Z"
}
```

#### Status Codes

- `200 OK`: Status updated successfully
- `400 Bad Request`: Invalid request data
- `401 Unauthorized`: Invalid runner token
- `404 Not Found`: Task not found
- `500 Internal Server Error`: Server error

---

### 4. Submit Task Logs

**POST** `/runner/tasks/{taskID}/trace`

Submits execution logs for a task.

#### Request

- **Method**: POST
- **Content-Type**: application/json
- **Path Parameters**:
  - `taskID`: Task ID (integer)

**Body**:

```json
{
  "logs": [
    {
      "timestamp": 1703862600000000,
      "source": "container",
      "message": "Starting task execution..."
    },
    {
      "timestamp": 1703862601000000,
      "source": "system",
      "message": "Container started successfully"
    }
  ]
}
```

#### Log Entry Fields

- `timestamp` (required): Unix timestamp in microseconds
- `source` (required): Log source (`container` or `system`)
- `message` (required): Log message text

#### Response

```json
{
  "execution_id": 456,
  "logs_added": 2
}
```

#### Status Codes

- `200 OK`: Logs submitted successfully
- `400 Bad Request`: Invalid log data
- `401 Unauthorized`: Invalid runner token
- `404 Not Found`: Task not found
- `500 Internal Server Error`: Server error

---

### 5. Upload Input Files

**GET** `/runner/tasks/{taskID}/inputs`

Downloads input files for task execution.

#### Request

- **Method**: POST
- **Content-Type**: multipart/form-data
- **Path Parameters**:
  - `taskID`: Task ID (integer)

**Body**: Multipart form with file fields

#### Response

```json
{
  "execution_id": 456,
  "files_stored": 3,
  "message": "Stored 3 input files"
}
```

#### Status Codes

- `200 OK`: Files uploaded successfully
- `400 Bad Request`: Invalid multipart data
- `401 Unauthorized`: Invalid runner token
- `404 Not Found`: Task not found
- `500 Internal Server Error`: Server error

---

### 6. Upload Output Files

**POST** `/runner/tasks/{taskID}/outputs`

Uploads output files after task completion.

#### Request

- **Method**: POST
- **Content-Type**: multipart/form-data
- **Path Parameters**:
  - `taskID`: Task ID (integer)

**Body**: Multipart form with file fields

#### Response

```json
{
  "execution_id": 456,
  "files_stored": 2,
  "message": "Stored 2 output files"
}
```

#### Status Codes

- `200 OK`: Files uploaded successfully
- `400 Bad Request`: Invalid multipart data
- `401 Unauthorized`: Invalid runner token
- `404 Not Found`: Task not found
- `500 Internal Server Error`: Server error

---

## Error Responses

All endpoints return consistent error responses:

```json
{
  "error": "Error message",
  "code": "error_code",
  "details": {
    "field": "Additional error details"
  }
}
```

### Common Error Codes

- `validation_error`: Request validation failed
- `not_found`: Resource not found
- `unauthorized`: Authentication failed

## Task Execution Flow

1. **Runner Startup**: Runner sends initial heartbeat
2. **Task Request**: Runner polls for available tasks
3. **Task Assignment**: Server assigns task to runner
4. **Status Updates**: Runner reports execution progress
5. **Log Streaming**: Runner submits execution logs
6. **File Upload**: Runner uploads input/output files
7. **Completion**: Runner reports final status

## Example Usage

### Go Client Example

```go
package main

import (
    "context"
    "log"
    "github.com/bornholm/oplet/internal/runner"
)

func main() {
    client, err := runner.NewClient("http://localhost:8080", "runner_token", nil)
    if err != nil {
        log.Fatal(err)
    }

    ctx := context.Background()

    // Send heartbeat
    resp, err := client.SendHeartbeat(ctx)
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("Heartbeat: %+v", resp)

    // Request task
    task, err := client.RequestTask(ctx)
    if err != nil {
        log.Fatal(err)
    }
    if task != nil {
        log.Printf("Received task: %+v", task)
    }
}
```

### cURL Examples

**Send Heartbeat**:

```bash
curl -X POST http://localhost:8080/runner/heartbeat \
  -H "Authorization: Bearer your_runner_token" \
  -H "Content-Type: application/json"
```

**Request Task**:

```bash
curl -X GET http://localhost:8080/runner/request-task \
  -H "Authorization: Bearer your_runner_token"
```

**Update Task Status**:

```bash
curl -X POST http://localhost:8080/runner/tasks/123/status \
  -H "Authorization: Bearer your_runner_token" \
  -H "Content-Type: application/json" \
  -d '{"status": "running", "container_id": "abc123"}'
```

## Security Considerations

1. **Token Security**: Runner tokens should be kept secure and rotated regularly
2. **HTTPS**: Use HTTPS in production environments
3. **Rate Limiting**: Consider implementing rate limiting for API endpoints
4. **File Validation**: Validate uploaded files for security

## Monitoring and Logging

The runner API includes comprehensive logging for:

- Authentication attempts
- Task assignments
- Status updates
- File uploads
- Error conditions

Monitor these logs for operational insights and security events.
