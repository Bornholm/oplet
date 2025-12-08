# OCI Task Provider

The OCI Task Provider enables Oplet to fetch task definitions from OCI-compatible container registries by parsing Docker labels embedded in container images.

## Overview

This package implements the `task.Provider` interface to:

- Connect to OCI-compatible registries (Docker Hub, GitHub Container Registry, etc.)
- Fetch image configurations and extract Docker labels
- Parse hierarchical label structures into task definitions
- Support both public and private registries (authentication support planned)

## Label Format Specification

The OCI provider uses a hierarchical label format to define task metadata, inputs, and configuration:

```
io.oplet.task.{category}.{name}.{property}
```

### Categories

- **`meta`** - Task metadata (name, description, author, etc.)
- **`inputs`** - User inputs (files, environment variables)
- **`config`** - Configuration parameters

### Meta Labels

| Label                            | Required | Description                 |
| -------------------------------- | -------- | --------------------------- |
| `io.oplet.task.meta.name`        | Yes      | Task display name           |
| `io.oplet.task.meta.description` | No       | Task description            |
| `io.oplet.task.meta.author`      | No       | Task author                 |
| `io.oplet.task.meta.url`         | No       | Documentation or source URL |

### Input/Config Properties

| Property      | Required | Values                   | Description                                    |
| ------------- | -------- | ------------------------ | ---------------------------------------------- |
| `type`        | Yes      | `env`, `file`            | How the input is provided                      |
| `value_type`  | Yes      | `text`, `number`, `file` | Type of the input value                        |
| `description` | No       | Any string               | Human-readable description                     |
| `required`    | No       | `true`, `false`          | Whether the input is required (default: false) |

## Example Dockerfile

```dockerfile
FROM alpine:latest

# Install your application
COPY my-app /usr/local/bin/my-app
RUN chmod +x /usr/local/bin/my-app

# Task metadata
LABEL io.oplet.task.meta.name="CSV Data Processor"
LABEL io.oplet.task.meta.description="Process and transform CSV files with custom rules"
LABEL io.oplet.task.meta.author="Data Team"
LABEL io.oplet.task.meta.url="https://github.com/company/csv-processor"

# File inputs
LABEL io.oplet.task.inputs.input_file.type="file"
LABEL io.oplet.task.inputs.input_file.value_type="file"
LABEL io.oplet.task.inputs.input_file.description="CSV file to process"
LABEL io.oplet.task.inputs.input_file.required="true"

# Environment variable inputs
LABEL io.oplet.task.inputs.output_format.type="env"
LABEL io.oplet.task.inputs.output_format.value_type="text"
LABEL io.oplet.task.inputs.output_format.description="Output format (json|csv|xml)"
LABEL io.oplet.task.inputs.output_format.required="false"

LABEL io.oplet.task.inputs.delimiter.type="env"
LABEL io.oplet.task.inputs.delimiter.value_type="text"
LABEL io.oplet.task.inputs.delimiter.description="CSV delimiter character"
LABEL io.oplet.task.inputs.delimiter.required="false"

# Configuration parameters
LABEL io.oplet.task.config.max_rows.type="env"
LABEL io.oplet.task.config.max_rows.value_type="number"
LABEL io.oplet.task.config.max_rows.description="Maximum number of rows to process"
LABEL io.oplet.task.config.max_rows.required="false"

LABEL io.oplet.task.config.timeout.type="env"
LABEL io.oplet.task.config.timeout.value_type="number"
LABEL io.oplet.task.config.timeout.description="Processing timeout in seconds"
LABEL io.oplet.task.config.timeout.required="false"

# Set the entrypoint
ENTRYPOINT ["/usr/local/bin/my-app"]
```

## Usage

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/bornholm/oplet/internal/task/oci"
)

func main() {
    // Create a new OCI provider
    provider := oci.NewProvider()

    // Fetch task definition from registry
    ctx := context.Background()
    definition, err := provider.FetchTaskDefinition(ctx, "registry.example.com/csv-processor:v1.0.0")
    if err != nil {
        log.Fatalf("Failed to fetch task definition: %v", err)
    }

    fmt.Printf("Task: %s\n", definition.Name)
    fmt.Printf("Description: %s\n", definition.Description)
    fmt.Printf("Inputs: %d\n", len(definition.Inputs))
    fmt.Printf("Configuration: %d\n", len(definition.Configuration))
}
```

## Supported Registries

The provider currently supports public registries without authentication:

- Docker Hub (`docker.io` or no prefix)
- GitHub Container Registry (`ghcr.io`)
- Google Container Registry (`gcr.io`)
- Any OCI-compatible registry with public access

## Error Handling

The provider returns specific error types for different failure scenarios:

- `ErrInvalidImageRef` - Malformed image reference
- `ErrImageNotFound` - Image not found in registry
- `ErrRegistryUnavailable` - Registry connection issues
- `ErrInvalidLabels` - Missing or malformed task labels
- `ErrUnsupportedImageFormat` - Unsupported image format

## Future Enhancements

- **Authentication Support** - Docker config, basic auth, token auth
- **Private Registry Support** - Custom CA certificates, insecure registries
- **Caching** - Cache image configurations to reduce registry calls
- **Label Validation** - Advanced validation rules for inputs and constraints
- **Multi-platform Images** - Support for platform-specific manifests

## Testing

Run the unit tests:

```bash
go test ./internal/task/oci -v
```

Run integration tests (requires network access):

```bash
go test ./internal/task/oci -v -run Integration
```

Skip integration tests:

```bash
go test ./internal/task/oci -v -short
```
