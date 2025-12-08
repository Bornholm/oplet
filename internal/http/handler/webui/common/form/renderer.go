package form

import (
	"github.com/a-h/templ"
)

// SelectOption represents an option in a select dropdown
type SelectOption struct {
	Value string
	Label string
}

// DefaultFieldRenderer provides basic HTML field rendering
type DefaultFieldRenderer struct{}

// RenderField renders a field using basic HTML
func (r *DefaultFieldRenderer) RenderField(ctx FieldContext) templ.Component {
	switch ctx.Type {
	case "textarea":
		return DefaultTextarea(ctx)
	case "checkbox":
		return DefaultCheckbox(ctx)
	case "file":
		return DefaultFileInput(ctx)
	case "select":
		// For select fields, we need FormOptions - this would need to be extended
		return DefaultInput(ctx)
	default:
		return DefaultInput(ctx)
	}
}

// TextareaRenderer renders textarea fields
type TextareaRenderer struct{}

func (r *TextareaRenderer) RenderField(ctx FieldContext) templ.Component {
	return DefaultTextarea(ctx)
}

// CheckboxRenderer renders checkbox fields
type CheckboxRenderer struct{}

func (r *CheckboxRenderer) RenderField(ctx FieldContext) templ.Component {
	return DefaultCheckbox(ctx)
}

// FileRenderer renders file input fields
type FileRenderer struct{}

func (r *FileRenderer) RenderField(ctx FieldContext) templ.Component {
	return DefaultFileInput(ctx)
}

// SelectRenderer renders select dropdown fields
type SelectRenderer struct {
	FormOptions []SelectOption
}

func (r *SelectRenderer) RenderField(ctx FieldContext) templ.Component {
	return DefaultSelect(ctx, r.FormOptions)
}

// NewSelectRenderer creates a new select renderer with FormOptions
func NewSelectRenderer(FormOptions []SelectOption) *SelectRenderer {
	return &SelectRenderer{FormOptions: FormOptions}
}
