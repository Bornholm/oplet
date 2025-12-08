package oci

import (
	"context"
	"testing"
	"time"

	"github.com/bornholm/oplet/internal/task"
)

func TestProvider_FetchTaskDefinition_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	provider := NewProvider()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tests := []struct {
		name        string
		imageRef    string
		expectError bool
		validate    func(t *testing.T, imageRef string)
	}{
		{
			name:        "public alpine image (should fail - no oplet labels)",
			imageRef:    "alpine:latest",
			expectError: true,
		},
		{
			name:        "invalid image reference",
			imageRef:    "invalid-image-ref",
			expectError: true,
		},
		{
			name:        "non-existent image",
			imageRef:    "registry.example.com/non-existent:latest",
			expectError: true,
		},
		{
			name:        "empty image reference",
			imageRef:    "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := provider.FetchTaskDefinition(ctx, tt.imageRef)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				t.Logf("Expected error: %v", err)
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if result == nil {
				t.Error("result should not be nil")
				return
			}

			if tt.validate != nil {
				tt.validate(t, tt.imageRef)
			}
		})
	}
}

func TestProvider_FetchTaskDefinition_MockImage(t *testing.T) {
	// This test would require a mock registry or a test image with proper labels
	// For now, we'll test the error handling paths
	provider := NewProvider()
	ctx := context.Background()

	// Test with empty image reference
	_, err := provider.FetchTaskDefinition(ctx, "")
	if err == nil {
		t.Error("expected error for empty image reference")
	}

	// Test with malformed image reference
	_, err = provider.FetchTaskDefinition(ctx, "not-a-valid-ref")
	if err == nil {
		t.Error("expected error for malformed image reference")
	}
}

func TestRegistryClient_FetchImageConfig(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	client := NewRegistryClient()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tests := []struct {
		name        string
		imageRef    string
		expectError bool
	}{
		{
			name:        "valid public image",
			imageRef:    "alpine:latest",
			expectError: false,
		},
		{
			name:        "invalid image reference",
			imageRef:    "invalid-ref",
			expectError: true,
		},
		{
			name:        "non-existent image",
			imageRef:    "registry.example.com/non-existent:latest",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := client.FetchImageConfig(ctx, tt.imageRef)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if result == nil {
				t.Error("result should not be nil")
				return
			}

			// Basic validation of the config
			if result.Config.Labels == nil {
				t.Log("Image has no labels (this is normal for alpine)")
			}

			t.Logf("Successfully fetched config for %s", tt.imageRef)
		})
	}
}

// TestProvider_EndToEnd tests the complete flow with a hypothetical image
func TestProvider_EndToEnd_Simulation(t *testing.T) {
	// This simulates what would happen with a properly labeled image
	parser := NewLabelParser()

	// Simulate labels from a real Oplet task image
	mockLabels := map[string]string{
		"io.oplet.task.meta.name":        "CSV Data Processor",
		"io.oplet.task.meta.description": "Process and transform CSV files",
		"io.oplet.task.meta.author":      "Oplet Team",
		"io.oplet.task.meta.url":         "https://github.com/example/csv-processor",

		"io.oplet.task.inputs.input_file.type":        "file",
		"io.oplet.task.inputs.input_file.value_type":  "file",
		"io.oplet.task.inputs.input_file.description": "CSV file to process",
		"io.oplet.task.inputs.input_file.required":    "true",

		"io.oplet.task.inputs.output_format.type":        "env",
		"io.oplet.task.inputs.output_format.value_type":  "text",
		"io.oplet.task.inputs.output_format.description": "Output format (json|csv|xml)",
		"io.oplet.task.inputs.output_format.required":    "false",

		"io.oplet.task.config.max_rows.type":        "env",
		"io.oplet.task.config.max_rows.value_type":  "number",
		"io.oplet.task.config.max_rows.description": "Maximum number of rows to process",
		"io.oplet.task.config.max_rows.required":    "false",
	}

	// Parse the labels
	parsed, err := parser.ParseLabels(mockLabels)
	if err != nil {
		t.Fatalf("failed to parse labels: %v", err)
	}

	// Build task definition
	imageRef := "registry.example.com/csv-processor:v1.0.0"
	definition, err := parser.BuildTaskDefinition(parsed, imageRef)
	if err != nil {
		t.Fatalf("failed to build task definition: %v", err)
	}

	// Validate the result
	if definition.Name != "CSV Data Processor" {
		t.Errorf("name: expected 'CSV Data Processor', got %q", definition.Name)
	}

	if definition.ImageRef != imageRef {
		t.Errorf("imageRef: expected %q, got %q", imageRef, definition.ImageRef)
	}

	if len(definition.Inputs) != 2 {
		t.Errorf("inputs: expected 2, got %d", len(definition.Inputs))
	}

	if len(definition.Configuration) != 1 {
		t.Errorf("configuration: expected 1, got %d", len(definition.Configuration))
	}

	// Find and validate specific inputs
	var inputFile, outputFormat *task.Input
	for _, input := range definition.Inputs {
		switch input.Name {
		case "input_file":
			inputFile = input
		case "output_format":
			outputFormat = input
		}
	}

	if inputFile == nil {
		t.Error("missing input_file")
	} else {
		if !inputFile.Required {
			t.Error("input_file should be required")
		}
		if inputFile.InputType != task.InputTypeFile {
			t.Errorf("input_file type: expected %v, got %v", task.InputTypeFile, inputFile.InputType)
		}
	}

	if outputFormat == nil {
		t.Error("missing output_format")
	} else {
		if outputFormat.Required {
			t.Error("output_format should not be required")
		}
		if outputFormat.InputType != task.InputTypeEnv {
			t.Errorf("output_format type: expected %v, got %v", task.InputTypeEnv, outputFormat.InputType)
		}
	}

	t.Logf("Successfully processed task definition: %s", definition.Name)
}
