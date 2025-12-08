package setup

import (
	"context"

	"github.com/bornholm/oplet/internal/config"
	"github.com/bornholm/oplet/internal/task"
	"github.com/bornholm/oplet/internal/task/oci"
)

var getTaskProviderFromConfig = createFromConfigOnce(func(ctx context.Context, conf *config.Config) (task.Provider, error) {
	provider := oci.NewProvider()
	return provider, nil
})
