package form

import "github.com/a-h/templ"

// Field represents a form field defined at runtime
type Field struct {
	Name        string
	Label       string
	Type        string
	Required    bool
	Validation  []ValidationRule
	Placeholder string
	FormOptions []SelectOption // for select fields
	Attributes  map[string]any
}

func (f Field) IsFile() bool {
	return f.Type == "file"
}

// FieldContext contains all information needed to render a form field
type FieldContext struct {
	Name        string
	Value       string
	Label       string
	Type        string
	Error       string
	Required    bool
	Placeholder string
	Class       string
	Attributes  map[string]any
}

// FieldRenderer describes a component that can render a single field
type FieldRenderer interface {
	RenderField(ctx FieldContext) templ.Component
}
