package kargo_promotion_fail_test

import (
	"context"
	"testing"
	"time"

	"github.com/akuity/kargo/hack/test/e2e/utils"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/client/generated"
	"github.com/akuity/kargo/pkg/client/watch"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

// This file provides necessary setup for a test package to run environment setup for e2e test.
// Because golang doesn't allow import of test code, this code needs to be added to each test package.
func TestMain(m *testing.M) {
	utils.InitEnv(m)
}

func TestKargoPromotionFail(t *testing.T) {
	feature := features.New("Example kargo promotion")
	feature.Setup(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		// TODO: get it from project fixture??
		return context.WithValue(ctx, "project", "kargo-promotion-fail")
	})

	// Setup and teardown fixtures from testdata folder
	feature.Setup(utils.RequireKargoCli)
	feature.Setup(utils.SetupKargoFixtures)
	feature.Teardown(utils.TeardownKargoFixtures)

	feature.Assess("promotion fails", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		project := ctx.Value("project").(string)
		stage := "kargo-promotion-fail-stage"
		origin := "images"

		kargoClient := ctx.Value("kargo_cli").(generated.KargoAPI)
		watchClient := ctx.Value("kargo_watch").(watch.Client)

		anyFreightID, err := utils.WaitForLatestFreight(ctx, watchClient, project, origin, 5*time.Minute)
		if err != nil {
			t.Fatal(err)
		}

		_, err = utils.PromoteAndWaitForPhase(t, kargoClient, watchClient, project, stage, anyFreightID, kargoapi.PromotionPhaseFailed, 5*time.Minute)
		if err != nil {
			t.Fatal(err)
		}

		return ctx
	})

	utils.TestEnv.Test(t, feature.Feature())

}
