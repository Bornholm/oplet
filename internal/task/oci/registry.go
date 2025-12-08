package oci

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/pkg/errors"
)

// RegistryClient handles OCI registry operations
type RegistryClient struct {
	timeout time.Duration
	logger  *slog.Logger
}

// NewRegistryClient creates a new registry client with default settings
func NewRegistryClient() *RegistryClient {
	return &RegistryClient{
		timeout: 30 * time.Second,
		logger:  slog.Default().With("component", "oci-registry"),
	}
}

// NewRegistryClientWithLogger creates a new registry client with a custom logger
func NewRegistryClientWithLogger(logger *slog.Logger) *RegistryClient {
	return &RegistryClient{
		timeout: 30 * time.Second,
		logger:  logger.With("component", "oci-registry"),
	}
}

// FetchImageConfig fetches the image configuration from the registry
func (c *RegistryClient) FetchImageConfig(ctx context.Context, imageRef string) (*v1.ConfigFile, error) {
	c.logger.Debug("starting image config fetch", "image_ref", imageRef)

	// Parse the image reference
	ref, err := name.ParseReference(imageRef)
	if err != nil {
		c.logger.Error("failed to parse image reference", "image_ref", imageRef, "error", err)
		return nil, errors.Wrap(ErrInvalidImageRef, err.Error())
	}

	c.logger.Debug("parsed image reference",
		"image_ref", imageRef,
		"registry", ref.Context().Registry.Name(),
		"repository", ref.Context().RepositoryStr(),
		"tag", ref.Identifier())

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	c.logger.Debug("fetching image from registry",
		"image_ref", imageRef,
		"timeout", c.timeout)

	// Fetch the image
	img, err := remote.Image(ref, remote.WithContext(ctx))
	if err != nil {
		if isNotFoundError(err) {
			c.logger.Warn("image not found in registry", "image_ref", imageRef, "error", err)
			return nil, errors.Wrap(ErrImageNotFound, err.Error())
		}
		c.logger.Error("registry unavailable or connection failed", "image_ref", imageRef, "error", err)
		return nil, errors.Wrap(ErrRegistryUnavailable, err.Error())
	}

	c.logger.Debug("successfully fetched image, extracting config", "image_ref", imageRef)

	// Get the image configuration
	configFile, err := img.ConfigFile()
	if err != nil {
		c.logger.Error("failed to extract image config", "image_ref", imageRef, "error", err)
		return nil, errors.Wrap(ErrUnsupportedImageFormat, err.Error())
	}

	c.logger.Debug("successfully extracted image config",
		"image_ref", imageRef,
		"architecture", configFile.Architecture,
		"os", configFile.OS,
		"label_count", len(configFile.Config.Labels))

	return configFile, nil
}

// isNotFoundError checks if the error indicates the image was not found
func isNotFoundError(err error) bool {
	// Check for common "not found" error patterns
	errStr := err.Error()
	return contains(errStr, "not found") ||
		contains(errStr, "404") ||
		contains(errStr, "MANIFEST_UNKNOWN") ||
		contains(errStr, "NAME_UNKNOWN")
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			(len(s) > len(substr) &&
				(s[:len(substr)] == substr ||
					s[len(s)-len(substr):] == substr ||
					indexOfSubstring(s, substr) >= 0)))
}

// indexOfSubstring finds the index of a substring in a string
func indexOfSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
