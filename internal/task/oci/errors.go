package oci

import (
	"github.com/pkg/errors"
)

var (
	// ErrInvalidImageRef is returned when the image reference is malformed
	ErrInvalidImageRef = errors.New("invalid image reference")

	// ErrImageNotFound is returned when the image cannot be found in the registry
	ErrImageNotFound = errors.New("image not found")

	// ErrRegistryUnavailable is returned when the registry is not accessible
	ErrRegistryUnavailable = errors.New("registry unavailable")

	// ErrUnsupportedImageFormat is returned when the image format is not supported
	ErrUnsupportedImageFormat = errors.New("unsupported image format")
)
