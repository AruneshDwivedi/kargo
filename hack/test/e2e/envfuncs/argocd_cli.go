package envfuncs

import (
	"context"
	"fmt"

	"sigs.k8s.io/e2e-framework/pkg/envconf"
)

func LoadArgocdConfig(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
	// TODO: other ways to discover/setup argocd config
	if argocdEnvConfig, err := GetEnv(ctx, []string{"argocd_cli", "config_file"}); err == nil {
		return context.WithValue(ctx, "argocd_config_file", argocdEnvConfig), nil
	}
	fmt.Println("Cannot load argocd config from env")
	// Argocd config is optional. Do not fail here
	return ctx, nil
}
