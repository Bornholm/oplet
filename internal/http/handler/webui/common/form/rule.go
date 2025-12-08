package form

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

// ValidationRule represents a validation rule that can be applied at runtime
type ValidationRule interface {
	Validate(form *Form, field Field) error
	ErrorMessage() string
}

// RequiredRule validates that a field is not empty
type RequiredRule struct{}

var _ ValidationRule = &RequiredRule{}

func (r RequiredRule) Validate(f *Form, field Field) error {
	found := true
	if field.IsFile() {
		if _, exists := f.Files[field.Name]; !exists {
			found = false
		}
	} else {
		value, exists := f.Values[field.Name]
		if !exists || strings.TrimSpace(value) == "" {
			found = false
		}
	}

	if !found {
		return errors.New("this field is required")
	}

	return nil
}

func (r RequiredRule) ErrorMessage() string {
	return "This field is required"
}

// MinLengthRule validates minimum string length
type MinLengthRule struct {
	MinLength int
}

var _ ValidationRule = &MinLengthRule{}

func (r MinLengthRule) Validate(f *Form, field Field) error {
	value := f.Values[field.Name]
	if len(value) < r.MinLength {
		return errors.Errorf("minimum length is %d characters", r.MinLength)
	}
	return nil
}

func (r MinLengthRule) ErrorMessage() string {
	return fmt.Sprintf("Minimum length is %d characters", r.MinLength)
}

// MaxLengthRule validates maximum string length
type MaxLengthRule struct {
	MaxLength int
}

func (r MaxLengthRule) Validate(f *Form, field *Field) error {
	value := f.Values[field.Name]
	if len(value) > r.MaxLength {
		return errors.Errorf("maximum length is %d characters", r.MaxLength)
	}
	return nil
}

func (r MaxLengthRule) ErrorMessage() string {
	return fmt.Sprintf("Maximum length is %d characters", r.MaxLength)
}

// NumberRangeRule validates number ranges
type NumberRangeRule struct {
	Min *int
	Max *int
}

var _ ValidationRule = &NumberRangeRule{}

func (r NumberRangeRule) Validate(f *Form, field Field) error {
	value := f.Values[field.Name]
	if value == "" {
		return nil // Let required rule handle empty values
	}

	num, err := strconv.Atoi(value)
	if err != nil {
		return errors.New("must be a valid number")
	}

	if r.Min != nil && num < *r.Min {
		return errors.Errorf("must be at least %d", *r.Min)
	}

	if r.Max != nil && num > *r.Max {
		return errors.Errorf("must be at most %d", *r.Max)
	}

	return nil
}

func (r NumberRangeRule) ErrorMessage() string {
	if r.Min != nil && r.Max != nil {
		return fmt.Sprintf("Must be between %d and %d", *r.Min, *r.Max)
	} else if r.Min != nil {
		return fmt.Sprintf("Must be at least %d", *r.Min)
	} else if r.Max != nil {
		return fmt.Sprintf("Must be at most %d", *r.Max)
	}
	return "Invalid number"
}
