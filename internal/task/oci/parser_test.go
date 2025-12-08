package oci

import (
	"testing"

	"github.com/bornholm/oplet/internal/task"
)

func TestLabelParser_ParseLabels(t *testing.T) {
	parser := NewLabelParser()

	tests := []struct {
		name        string
		labels      map[string]string
		expected    *ParsedLabels
		expectError bool
	}{
		{
			name: "valid complete labels",
			labels: map[string]string{
				"io.oplet.task.meta.name":                     "CSV Processor",
				"io.oplet.task.meta.description":              "Process CSV files",
				"io.oplet.task.meta.author":                   "John Doe",
				"io.oplet.task.meta.url":                      "https://example.com",
				"io.oplet.task.inputs.data_file.type":         "file",
				"io.oplet.task.inputs.data_file.value_type":   "file",
				"io.oplet.task.inputs.data_file.description":  "Input CSV file",
				"io.oplet.task.inputs.data_file.required":     "true",
				"io.oplet.task.config.max_memory.type":        "env",
				"io.oplet.task.config.max_memory.value_type":  "number",
				"io.oplet.task.config.max_memory.description": "Max memory in MB",
				"io.oplet.task.config.max_memory.required":    "false",
			},
			expected: &ParsedLabels{
				Meta: MetaLabels{
					Name:        "CSV Processor",
					Description: "Process CSV files",
					Author:      "John Doe",
					URL:         "https://example.com",
				},
				Inputs: map[string]InputLabels{
					"data_file": {
						Type:        "file",
						ValueType:   "file",
						Description: "Input CSV file",
						Required:    "true",
					},
				},
				Config: map[string]InputLabels{
					"max_memory": {
						Type:        "env",
						ValueType:   "number",
						Description: "Max memory in MB",
						Required:    "false",
					},
				},
			},
			expectError: false,
		},
		{
			name: "minimal valid labels",
			labels: map[string]string{
				"io.oplet.task.meta.name":                "Simple Task",
				"io.oplet.task.inputs.input1.type":       "env",
				"io.oplet.task.inputs.input1.value_type": "text",
			},
			expected: &ParsedLabels{
				Meta: MetaLabels{
					Name: "Simple Task",
				},
				Inputs: map[string]InputLabels{
					"input1": {
						Type:      "env",
						ValueType: "text",
					},
				},
				Config: map[string]InputLabels{},
			},
			expectError: false,
		},
		{
			name: "missing required type",
			labels: map[string]string{
				"io.oplet.task.meta.name":                "Invalid Task",
				"io.oplet.task.inputs.input1.value_type": "text",
			},
			expectError: true,
		},
		{
			name: "invalid input type",
			labels: map[string]string{
				"io.oplet.task.meta.name":                "Invalid Task",
				"io.oplet.task.inputs.input1.type":       "invalid",
				"io.oplet.task.inputs.input1.value_type": "text",
			},
			expectError: true,
		},
		{
			name: "invalid value type",
			labels: map[string]string{
				"io.oplet.task.meta.name":                "Invalid Task",
				"io.oplet.task.inputs.input1.type":       "env",
				"io.oplet.task.inputs.input1.value_type": "invalid",
			},
			expectError: true,
		},
		{
			name: "malformed label format",
			labels: map[string]string{
				"io.oplet.task.meta.name":     "Invalid Task",
				"io.oplet.task.inputs.input1": "malformed",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.ParseLabels(tt.labels)

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

			// Compare meta
			if result.Meta.Name != tt.expected.Meta.Name {
				t.Errorf("meta.name: expected %q, got %q", tt.expected.Meta.Name, result.Meta.Name)
			}
			if result.Meta.Description != tt.expected.Meta.Description {
				t.Errorf("meta.description: expected %q, got %q", tt.expected.Meta.Description, result.Meta.Description)
			}
			if result.Meta.Author != tt.expected.Meta.Author {
				t.Errorf("meta.author: expected %q, got %q", tt.expected.Meta.Author, result.Meta.Author)
			}
			if result.Meta.URL != tt.expected.Meta.URL {
				t.Errorf("meta.url: expected %q, got %q", tt.expected.Meta.URL, result.Meta.URL)
			}

			// Compare inputs
			if len(result.Inputs) != len(tt.expected.Inputs) {
				t.Errorf("inputs count: expected %d, got %d", len(tt.expected.Inputs), len(result.Inputs))
			}
			for name, expected := range tt.expected.Inputs {
				actual, exists := result.Inputs[name]
				if !exists {
					t.Errorf("missing input: %s", name)
					continue
				}
				compareInputLabels(t, name, expected, actual)
			}

			// Compare config
			if len(result.Config) != len(tt.expected.Config) {
				t.Errorf("config count: expected %d, got %d", len(tt.expected.Config), len(result.Config))
			}
			for name, expected := range tt.expected.Config {
				actual, exists := result.Config[name]
				if !exists {
					t.Errorf("missing config: %s", name)
					continue
				}
				compareInputLabels(t, name, expected, actual)
			}
		})
	}
}

func compareInputLabels(t *testing.T, name string, expected, actual InputLabels) {
	if actual.Type != expected.Type {
		t.Errorf("%s.type: expected %q, got %q", name, expected.Type, actual.Type)
	}
	if actual.ValueType != expected.ValueType {
		t.Errorf("%s.value_type: expected %q, got %q", name, expected.ValueType, actual.ValueType)
	}
	if actual.Description != expected.Description {
		t.Errorf("%s.description: expected %q, got %q", name, expected.Description, actual.Description)
	}
	if actual.Required != expected.Required {
		t.Errorf("%s.required: expected %q, got %q", name, expected.Required, actual.Required)
	}
}

func TestLabelParser_BuildTaskDefinition(t *testing.T) {
	parser := NewLabelParser()

	tests := []struct {
		name        string
		parsed      *ParsedLabels
		imageRef    string
		expectError bool
		validate    func(t *testing.T, def *task.Definition)
	}{
		{
			name: "valid task definition",
			parsed: &ParsedLabels{
				Meta: MetaLabels{
					Name:        "Test Task",
					Description: "A test task",
					Author:      "Test Author",
					URL:         "https://test.com",
				},
				Inputs: map[string]InputLabels{
					"input_file": {
						Type:        "file",
						ValueType:   "file",
						Description: "Input file",
						Required:    "true",
					},
					"env_var": {
						Type:        "env",
						ValueType:   "text",
						Description: "Environment variable",
						Required:    "false",
					},
				},
				Config: map[string]InputLabels{
					"timeout": {
						Type:        "env",
						ValueType:   "number",
						Description: "Timeout in seconds",
						Required:    "false",
					},
				},
			},
			imageRef:    "registry.example.com/test:latest",
			expectError: false,
			validate: func(t *testing.T, def *task.Definition) {
				if def.Name != "Test Task" {
					t.Errorf("name: expected 'Test Task', got %q", def.Name)
				}
				if def.Description != "A test task" {
					t.Errorf("description: expected 'A test task', got %q", def.Description)
				}
				if def.Author != "Test Author" {
					t.Errorf("author: expected 'Test Author', got %q", def.Author)
				}
				if def.URL != "https://test.com" {
					t.Errorf("url: expected 'https://test.com', got %q", def.URL)
				}
				if def.ImageRef != "registry.example.com/test:latest" {
					t.Errorf("imageRef: expected 'registry.example.com/test:latest', got %q", def.ImageRef)
				}
				if len(def.Inputs) != 2 {
					t.Errorf("inputs count: expected 2, got %d", len(def.Inputs))
				}
				if len(def.Configuration) != 1 {
					t.Errorf("configuration count: expected 1, got %d", len(def.Configuration))
				}

				// Check specific input
				var inputFile *task.Input
				for _, input := range def.Inputs {
					if input.Name == "input_file" {
						inputFile = input
						break
					}
				}
				if inputFile == nil {
					t.Error("missing input_file")
				} else {
					if inputFile.InputType != task.InputTypeFile {
						t.Errorf("input_file.InputType: expected %v, got %v", task.InputTypeFile, inputFile.InputType)
					}
					if inputFile.ValueType != task.ValueTypeFile {
						t.Errorf("input_file.ValueType: expected %v, got %v", task.ValueTypeFile, inputFile.ValueType)
					}
					if !inputFile.Required {
						t.Error("input_file.Required: expected true, got false")
					}
				}
			},
		},
		{
			name: "missing name",
			parsed: &ParsedLabels{
				Meta: MetaLabels{
					Description: "A test task without name",
				},
				Inputs: map[string]InputLabels{},
				Config: map[string]InputLabels{},
			},
			imageRef:    "registry.example.com/test:latest",
			expectError: true,
		},
		{
			name: "invalid required value",
			parsed: &ParsedLabels{
				Meta: MetaLabels{
					Name: "Test Task",
				},
				Inputs: map[string]InputLabels{
					"input1": {
						Type:      "env",
						ValueType: "text",
						Required:  "invalid",
					},
				},
				Config: map[string]InputLabels{},
			},
			imageRef:    "registry.example.com/test:latest",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.BuildTaskDefinition(tt.parsed, tt.imageRef)

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

			if tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}
