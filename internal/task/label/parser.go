package label

import (
	"strconv"
	"strings"

	"github.com/bornholm/oplet/internal/task"
	"github.com/pkg/errors"
)

// Parser handles parsing of Docker labels into task definitions
type Parser struct{}

// NewParser creates a new label parser
func NewParser() *Parser {
	return &Parser{}
}

// ParseLabels extracts and parses Oplet-specific labels from a label map
func (p *Parser) ParseLabels(labels map[string]string) (*ParsedLabels, error) {
	parsed := &ParsedLabels{
		Inputs: make(map[string]InputLabels),
		Config: make(map[string]InputLabels),
	}

	// Parse meta labels
	parsed.Meta = p.parseMetaLabels(labels)

	// Parse input and config labels
	if err := p.parseInputLabels(labels, parsed); err != nil {
		return nil, err
	}

	return parsed, nil
}

// parseMetaLabels extracts metadata labels
func (p *Parser) parseMetaLabels(labels map[string]string) MetaLabels {
	return MetaLabels{
		Name:        labels[LabelMetaName],
		Description: labels[LabelMetaDescription],
		Author:      labels[LabelMetaAuthor],
		URL:         labels[LabelMetaURL],
	}
}

// parseInputLabels extracts input and configuration labels
func (p *Parser) parseInputLabels(labels map[string]string, parsed *ParsedLabels) error {
	inputGroups := make(map[string]map[string]string)
	configGroups := make(map[string]map[string]string)

	// Group labels by input/config name
	for key, value := range labels {
		if strings.HasPrefix(key, LabelPrefixInputs+".") {
			if err := p.groupLabel(key, value, LabelPrefixInputs, inputGroups); err != nil {
				return errors.WithStack(err)
			}
		} else if strings.HasPrefix(key, LabelPrefixConfig+".") {
			if err := p.groupLabel(key, value, LabelPrefixConfig, configGroups); err != nil {
				return errors.WithStack(err)
			}
		}
	}

	// Convert grouped labels to InputLabels
	for name, props := range inputGroups {
		inputLabels, err := p.buildInputLabels(props)
		if err != nil {
			return errors.Wrapf(err, "invalid input '%s'", name)
		}
		parsed.Inputs[name] = inputLabels
	}

	for name, props := range configGroups {
		configLabels, err := p.buildInputLabels(props)
		if err != nil {
			return errors.Wrapf(err, "invalid config '%s'", name)
		}
		parsed.Config[name] = configLabels
	}

	return nil
}

// groupLabel groups a label by its name and property
func (p *Parser) groupLabel(key, value, prefix string, groups map[string]map[string]string) error {
	// Remove prefix and split by dots
	suffix := strings.TrimPrefix(key, prefix+".")
	parts := strings.SplitN(suffix, ".", 2)

	if len(parts) != 2 {
		return errors.Errorf("invalid label format: %s", key)
	}

	name := parts[0]
	property := parts[1]

	if groups[name] == nil {
		groups[name] = make(map[string]string)
	}
	groups[name][property] = value

	return nil
}

// buildInputLabels creates InputLabels from a property map
func (p *Parser) buildInputLabels(props map[string]string) (InputLabels, error) {
	inputLabels := InputLabels{
		Type:        props[PropertyType],
		ValueType:   props[PropertyValueType],
		Description: props[PropertyDescription],
		Required:    props[PropertyRequired],
	}

	// Validate required fields
	if inputLabels.Type == "" {
		return inputLabels, errors.New("missing required property 'type'")
	}
	if inputLabels.ValueType == "" {
		return inputLabels, errors.New("missing required property 'value_type'")
	}

	// Validate type values
	if !p.isValidInputType(inputLabels.Type) {
		return inputLabels, errors.Errorf("invalid type '%s', must be 'env' or 'file'", inputLabels.Type)
	}
	if !p.isValidValueType(inputLabels.ValueType) {
		return inputLabels, errors.Errorf("invalid value_type '%s', must be 'text', 'number', or 'file'", inputLabels.ValueType)
	}

	return inputLabels, nil
}

// isValidInputType checks if the input type is valid
func (p *Parser) isValidInputType(inputType string) bool {
	return inputType == string(task.InputTypeEnv) || inputType == string(task.InputTypeFile)
}

// isValidValueType checks if the value type is valid
func (p *Parser) isValidValueType(valueType string) bool {
	return valueType == string(task.ValueTypeText) ||
		valueType == string(task.ValueTypeNumber) ||
		valueType == string(task.ValueTypeFile)
}

// BuildTaskDefinition converts parsed labels to a task.Definition
func (p *Parser) BuildTaskDefinition(parsed *ParsedLabels, imageRef string) (*task.Definition, error) {
	// Validate that we have at least a name
	if parsed.Meta.Name == "" {
		return nil, errors.Wrap(ErrInvalidLabels, "missing required meta.name label")
	}

	definition := &task.Definition{
		Name:          parsed.Meta.Name,
		Description:   parsed.Meta.Description,
		Author:        parsed.Meta.Author,
		URL:           parsed.Meta.URL,
		ImageRef:      imageRef,
		Inputs:        make([]*task.Input, 0),
		Configuration: make([]*task.Input, 0),
	}

	// Convert inputs
	for name, inputLabels := range parsed.Inputs {
		taskInput, err := p.convertToTaskInput(name, inputLabels)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to convert input '%s'", name)
		}
		definition.Inputs = append(definition.Inputs, taskInput)
	}

	// Convert configuration
	for name, configLabels := range parsed.Config {
		taskInput, err := p.convertToTaskInput(name, configLabels)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to convert config '%s'", name)
		}
		definition.Configuration = append(definition.Configuration, taskInput)
	}

	return definition, nil
}

// convertToTaskInput converts InputLabels to task.TaskInput
func (p *Parser) convertToTaskInput(name string, inputLabels InputLabels) (*task.Input, error) {
	// Parse required field
	required := false
	if inputLabels.Required != "" {
		var err error
		required, err = strconv.ParseBool(inputLabels.Required)
		if err != nil {
			return nil, errors.Errorf("invalid required value '%s', must be 'true' or 'false'", inputLabels.Required)
		}
	}

	return &task.Input{
		Name:        name,
		InputType:   task.InputType(inputLabels.Type),
		Description: inputLabels.Description,
		ValueType:   task.ValueType(inputLabels.ValueType),
		Required:    required,
		Constraints: []task.Constraint{}, // No constraints for now
	}, nil
}
