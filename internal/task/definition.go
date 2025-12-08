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

type Type string

const (
	TypeText    Type = "text"
	TypeNumber  Type = "number"
	TypeFile    Type = "file"
	TypeSecret  Type = "secret"
	TypeBoolean Type = "boolean"
)

type Input struct {
	Name         string
	Label        string
	Type         Type
	Description  string
	Required     bool
	DefaultValue string
	Tags         []string
	Constraints  []Constraint
}

var (
	ErrSkipConstraint = errors.New("skip constraint")
)

type Constraint interface {
	FileConstraint
	ValueConstraint
}

type ValueConstraint interface {
	AssertValue(ctx context.Context, input *Input, value string) error
}

type ValueConstraintFunc func(ctx context.Context, input *Input, value string) error

func (fn ValueConstraintFunc) AssertValue(ctx context.Context, input *Input, value string) error {
	return fn(ctx, input, value)
}

type valueConstraint struct {
	assert ValueConstraintFunc
}

// AssertFile implements Constraint.
func (c *valueConstraint) AssertFile(ctx context.Context, input *Input, r io.Reader) error {
	return errors.WithStack(ErrSkipConstraint)
}

// AssertValue implements Constraint.
func (c *valueConstraint) AssertValue(ctx context.Context, input *Input, value string) error {
	return errors.WithStack(c.assert(ctx, input, value))
}

var _ Constraint = &valueConstraint{}

func NewValueConstraint(fn ValueConstraintFunc) Constraint {
	return &valueConstraint{fn}
}

type FileConstraint interface {
	AssertFile(ctx context.Context, input *Input, r io.Reader) error
}

type FileConstraintFunc func(ctx context.Context, input *Input, r io.Reader) error

func (fn FileConstraintFunc) AssertFile(ctx context.Context, input *Input, r io.Reader) error {
	return fn(ctx, input, r)
}

type fileConstraint struct {
	assert FileConstraintFunc
}

// AssertFile implements Constraint.
func (c *fileConstraint) AssertFile(ctx context.Context, input *Input, r io.Reader) error {
	return errors.WithStack(c.assert(ctx, input, r))
}

// AssertValue implements Constraint.
func (c *fileConstraint) AssertValue(ctx context.Context, input *Input, value string) error {
	return errors.WithStack(ErrSkipConstraint)
}

var _ Constraint = &fileConstraint{}

func NewFileConstraint(fn FileConstraintFunc) Constraint {
	return &fileConstraint{fn}
}
