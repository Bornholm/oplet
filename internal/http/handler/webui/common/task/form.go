package task

import (
	"context"
	"slices"
	"strings"

	"github.com/bornholm/oplet/internal/http/handler/webui/common/form"
	"github.com/bornholm/oplet/internal/store"
	"github.com/bornholm/oplet/internal/task"
	"github.com/pkg/errors"
)

// NewInputForm creates a new form from task definition inputs
func NewInputForm(taskDef *task.Definition) *form.Form {
	return newForm(taskDef.Inputs, map[string]string{})
}

// NewConfigurationForm creates a new form from task definition configurations
func NewConfigurationForm(taskDef *task.Definition, task *store.Task) *form.Form {
	values := make(map[string]string)
	for _, tc := range task.Configurations {
		values[tc.Name] = tc.Value
	}

	return newForm(taskDef.Configuration, values)
}

func newForm(inputs []*task.Input, defaultValues map[string]string) *form.Form {
	// Convert task inputs to dynamic fields
	fields := make([]form.Field, 0)
	for _, input := range inputs {
		field := form.Field{
			Name:        input.Name,
			Label:       input.Label,
			Type:        mapInputTypeToHTMLType(input.Type),
			Required:    input.Required,
			Placeholder: input.Description,
			Attributes:  map[string]any{},
			Validation:  make([]form.ValidationRule, 0),
		}

		if field.Type == "password" {
			field.Attributes["autocomplete"] = "one-time-code"
		}

		// Build validation rules
		field.Validation = buildValidationRules(input)

		fields = append(fields, field)
	}

	slices.SortFunc(fields, func(f1 form.Field, f2 form.Field) int {
		if f1.Required && !f2.Required {
			return -1
		}
		if !f1.Required && f2.Required {
			return 1
		}

		return strings.Compare(f1.Name, f2.Name)
	})

	form := form.New(fields)

	form.Values = defaultValues

	return form
}

// NewImageRefForm creates a new form
func NewImageRefForm() *form.Form {
	imageRef := form.Field{
		Name:        "image_ref",
		Label:       "Image Reference",
		Type:        "text",
		Required:    true,
		Placeholder: "The Docker image reference",
		Attributes:  make(map[string]any),
		Validation:  make([]form.ValidationRule, 0),
	}

	form := form.New([]form.Field{imageRef})

	return form
}

// Helper functions

func mapInputTypeToHTMLType(inputType task.Type) string {
	switch inputType {
	case task.TypeNumber:
		return "number"
	case task.TypeFile:
		return "file"
	case task.TypeText:
		return "text"
	case task.TypeSecret:
		return "password"
	case task.TypeBoolean:
		return "checkbox"
	default:
		return "text"
	}
}

func buildValidationRules(input *task.Input) []form.ValidationRule {
	var rules []form.ValidationRule

	if input.Required {
		rules = append(rules, form.RequiredRule{})
	}

	// Add constraint-based validation rules
	for _, constraint := range input.Constraints {
		rules = append(rules, &constraintBasedValidationRule{input, constraint})
	}

	return rules
}

type constraintBasedValidationRule struct {
	input      *task.Input
	constraint task.Constraint
}

// Validate implements form.ValidationRule.
func (r *constraintBasedValidationRule) Validate(ctx context.Context, form *form.Form, field form.Field) error {
	switch field.Type {
	case "file":
		fileHeaders, exists := form.Files[field.Name]
		if !exists {
			return nil
		}

		for _, fh := range fileHeaders {
			file, err := fh.Open()
			if err != nil {
				return errors.WithStack(err)
			}
			defer file.Close()

			if err := r.constraint.AssertFile(ctx, r.input, file); err != nil && !errors.Is(err, task.ErrSkipConstraint) {
				return errors.WithStack(err)
			}
		}

	default:
		value := form.Values[field.Name]

		if err := r.constraint.AssertValue(ctx, r.input, value); err != nil && !errors.Is(err, task.ErrSkipConstraint) {
			return errors.WithStack(err)
		}
	}

	return nil
}

var _ form.ValidationRule = &constraintBasedValidationRule{}
