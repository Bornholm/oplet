package oci

import (
	"context"
	"log/slog"

	"github.com/bornholm/oplet/internal/task"
	"github.com/pkg/errors"
)

// Provider implements task.Provider for OCI registries
type Provider struct {
	registryClient *RegistryClient
	labelParser    *LabelParser
	logger         *slog.Logger
}

// NewProvider creates a new OCI provider with default settings
func NewProvider() *Provider {
	return &Provider{
		registryClient: NewRegistryClient(),
		labelParser:    NewLabelParser(),
		logger:         slog.Default().With("component", "oci-provider"),
	}
}

// NewProviderWithLogger creates a new OCI provider with a custom logger
func NewProviderWithLogger(logger *slog.Logger) *Provider {
	return &Provider{
		registryClient: NewRegistryClient(),
		labelParser:    NewLabelParser(),
		logger:         logger.With("component", "oci-provider"),
	}
}

// FetchTaskDefinition implements task.Provider.
// It fetches an image from an OCI registry and extracts task definition from labels.
func (p *Provider) FetchTaskDefinition(ctx context.Context, imageRef string) (*task.Definition, error) {
	p.logger.Info("fetching task definition", "image_ref", imageRef)

	// Validate input
	if imageRef == "" {
		p.logger.Error("empty image reference provided")
		return nil, errors.Wrap(ErrInvalidImageRef, "image reference cannot be empty")
	}

	// Fetch image configuration from registry
	p.logger.Debug("fetching image config from registry", "image_ref", imageRef)
	configFile, err := p.registryClient.FetchImageConfig(ctx, imageRef)
	if err != nil {
		p.logger.Error("failed to fetch image config", "image_ref", imageRef, "error", err)
		return nil, errors.Wrapf(err, "failed to fetch image config for '%s'", imageRef)
	}

	// Extract labels from image config
	labels := configFile.Config.Labels
	if labels == nil {
		labels = make(map[string]string)
	}

	p.logger.Debug("extracted labels from image",
		"image_ref", imageRef,
		"label_count", len(labels),
		"oplet_labels", p.countOpletLabels(labels))

	// Parse labels into structured format
	parsedLabels, err := p.labelParser.ParseLabels(labels)
	if err != nil {
		p.logger.Error("failed to parse labels", "image_ref", imageRef, "error", err)
		return nil, errors.Wrapf(err, "failed to parse labels for image '%s'", imageRef)
	}

	p.logger.Debug("parsed labels successfully",
		"image_ref", imageRef,
		"task_name", parsedLabels.Meta.Name,
		"input_count", len(parsedLabels.Inputs),
		"config_count", len(parsedLabels.Config))

	// Build task definition from parsed labels
	definition, err := p.labelParser.BuildTaskDefinition(parsedLabels, imageRef)
	if err != nil {
		p.logger.Error("failed to build task definition", "image_ref", imageRef, "error", err)
		return nil, errors.Wrapf(err, "failed to build task definition for image '%s'", imageRef)
	}

	p.logger.Info("successfully created task definition",
		"image_ref", imageRef,
		"task_name", definition.Name,
		"input_count", len(definition.Inputs),
		"config_count", len(definition.Configuration))

	return definition, nil
}

// countOpletLabels counts how many labels are Oplet-specific
func (p *Provider) countOpletLabels(labels map[string]string) int {
	count := 0
	for key := range labels {
		if len(key) >= len(LabelPrefixTask) && key[:len(LabelPrefixTask)] == LabelPrefixTask {
			count++
		}
	}
	return count
}

// Ensure Provider implements task.Provider interface
var _ task.Provider = &Provider{}
