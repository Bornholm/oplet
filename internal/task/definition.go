package task

import (
	"context"
	"io"

	"github.com/pkg/errors"
)

type Definition struct {
	Name          string
	Description   string
	Author        string
	URL           string
	ImageRef      string
	Inputs        []*Input
	Configuration []*Input
}

type InputType string

const (
	InputTypeEnv  InputType = "env"
	InputTypeFile InputType = "file"
)

type ValueType string

const (
	ValueTypeText   ValueType = "text"
	ValueTypeNumber ValueType = "number"
	ValueTypeFile   ValueType = "file"
)

type Input struct {
	Name        string
	InputType   InputType
	Description string
	ValueType   ValueType
	Required    bool
	Constraints []Constraint
}

var (
	ErrSkipConstraint = errors.New("skip constraint")
)

type Constraint interface {
	AssertValue(ctx context.Context, input *Input, value string) (bool, error)
	AssertFile(ctx context.Context, input *Input, r io.Reader) (bool, error)
}
