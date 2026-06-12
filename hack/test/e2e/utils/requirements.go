package utils

import (
	"context"
	"testing"

	"github.com/akuity/kargo/hack/test/e2e/envfuncs"
	"github.com/akuity/kargo/pkg/client/generated"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func RequireEnvValue(path []string) features.Func {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		ctx = RequireContextValue("env")(ctx, t, cfg)
		_, err := envfuncs.GetEnv(ctx, path)
		if err != nil {
			t.Fatalf("cannot get value for path %v from context %v", path, ctx)
		}
		return ctx
	}
}

func RequireContextValue(key string) features.Func {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		_, ok := ctx.Value(key).(generated.KargoAPI)
		if !ok {
			t.Fatalf("%v is required in context", key)
		}
		return ctx
	}
}

func RequireKargoCli(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	return RequireContextValue("kargo_cli")(ctx, t, cfg)
}
