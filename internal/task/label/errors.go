package label

import (
	"github.com/pkg/errors"
)

var (
	// ErrInvalidLabels is returned when the image labels are malformed or missing required fields
	ErrInvalidLabels = errors.New("invalid or missing task labels")
)
