package label

const (
	// Label prefixes for Oplet task definitions
	LabelPrefixTask   = "io.oplet.task"
	LabelPrefixMeta   = "io.oplet.task.meta"
	LabelPrefixInputs = "io.oplet.task.inputs"
	LabelPrefixConfig = "io.oplet.task.config"

	// Meta label keys
	LabelMetaName        = "io.oplet.task.meta.name"
	LabelMetaDescription = "io.oplet.task.meta.description"
	LabelMetaAuthor      = "io.oplet.task.meta.author"
	LabelMetaURL         = "io.oplet.task.meta.url"

	// Input/Config property suffixes
	PropertyType        = "type"
	PropertyValueType   = "value_type"
	PropertyDescription = "description"
	PropertyRequired    = "required"
)

// ParsedLabels represents the structured labels extracted from an image
type ParsedLabels struct {
	Meta   MetaLabels             `json:"meta"`
	Inputs map[string]InputLabels `json:"inputs"`
	Config map[string]InputLabels `json:"config"`
}

// MetaLabels contains metadata about the task
type MetaLabels struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Author      string `json:"author"`
	URL         string `json:"url"`
}

// InputLabels contains the properties for a single input or configuration item
type InputLabels struct {
	Type        string `json:"type"`
	ValueType   string `json:"value_type"`
	Description string `json:"description"`
	Required    string `json:"required"`
}
