package task

import (
	"fmt"
	"slices"
	"strings"

	"github.com/bornholm/oplet/internal/http/handler/webui/common/form"
	"github.com/bornholm/oplet/internal/task"
)

// NewTaskForm creates a new form from task definition
func NewTaskForm(taskDef *task.Definition) *form.Form {

	// Convert task inputs to dynamic fields
	fields := make([]form.Field, 0)
	for _, input := range taskDef.Inputs {
		field := form.Field{
			Name:        input.Name,
			Label:       input.Description,
			Type:        mapValueTypeToHTMLType(input.ValueType),
			Required:    input.Required,
			Description: input.Description,
			Attributes:  make(map[string]any),
			Validation:  make([]form.ValidationRule, 0),
		}

		// Build validation rules
		field.Validation = buildValidationRules(input)

		// Set placeholder based on type and description
		field.Placeholder = generatePlaceholder(input)

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

	return form
}

// Helper functions

func mapValueTypeToHTMLType(valueType task.ValueType) string {
	switch valueType {
	case task.ValueTypeNumber:
		return "number"
	case task.ValueTypeFile:
		return "file"
	case task.ValueTypeText:
		return "text"
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
		// This would need to be implemented based on your constraint types
		// For now, we'll add some basic rules based on value type
		_ = constraint // TODO: implement constraint-based validation
		switch input.ValueType {
		case task.ValueTypeNumber:
			// Could add number range rules based on constraints
		case task.ValueTypeText:
			// Could add length rules based on constraints
		}
	}

	return rules
}

func generatePlaceholder(input *task.Input) string {
	switch input.ValueType {
	case task.ValueTypeNumber:
		return "Enter a number"
	case task.ValueTypeFile:
		return "Choose a file"
	case task.ValueTypeText:
		if input.Description != "" {
			return fmt.Sprintf("Enter %s", strings.ToLower(input.Description))
		}
		return "Enter text"
	default:
		return ""
	}
}
