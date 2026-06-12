package argocd_update_test

import (
	"context"
	"testing"
	"time"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/hack/test/e2e/envfuncs"
	"github.com/akuity/kargo/hack/test/e2e/utils"
	"github.com/akuity/kargo/pkg/client/generated"
	"github.com/akuity/kargo/pkg/client/watch"

	// "github.com/akuity/kargo/pkg/client/generated/core"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

// This file provides necessary setup for a test package to run environment setup for e2e test.
// Because golang doesn't allow import of test code, this code needs to be added to each test package.
func TestMain(m *testing.M) {
	utils.InitEnv(m)
}

func TestArgocdUpdate(t *testing.T) {
	feature := features.New("argocd-update")

	feature.Setup(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		// TODO: get it from project fixture??
		return context.WithValue(ctx, "project", "kargo-argocd-update")
	})

	// FIXME: do not do LoadKargoClient, replace with this
	feature.Setup(utils.SetupKargoClients)

	// Setup and teardown fixtures from testdata folder
	feature.Setup(utils.RequireKargoCli)
	feature.Setup(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		kargoDemoRepoVal, err := envfuncs.GetEnv(ctx, []string{"context", "kargo_demo_gitops_repo"})
		if err != nil {
			t.Fatalf("cannot get kargo_demo_gitops_repo %v", err)
		}
		kargoDemoRepo := kargoDemoRepoVal.(string)

		return utils.NewSetupKargoFixtures( 
		utils.UpdatePromotionTasksVar("promo-process", "gitRepo", kargoDemoRepo),
		utils.UpdateWarehouseGitRepoURL("kargo-demo", kargoDemoRepo),
		)(ctx, t, cfg)
	})
	// feature.Teardown(utils.TeardownKargoFixtures)

	feature.Assess("require freight", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		project := ctx.Value("project").(string)
		origin := "kargo-demo"
		watchClient := ctx.Value("kargo_watch").(watch.Client)
		anyFreightId, err  := utils.WaitForLatestFreight(ctx, watchClient, project, origin, 10*time.Minute)
		if err != nil {
			t.Fatal(err)
		}
		return context.WithValue(ctx, "freight_id", anyFreightId)
	})

	feature.Assess("promote test", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		freightID := ctx.Value("freight_id").(string)
		project := ctx.Value("project").(string)
		kargoClient := ctx.Value("kargo_cli").(generated.KargoAPI)
		watchClient := ctx.Value("kargo_watch").(watch.Client)
		stage := "test"

		_, err := utils.PromoteAndWaitForPhase(ctx, t, kargoClient, watchClient, project, stage, freightID, kargoapi.PromotionPhaseSucceeded, 10*time.Minute)
		if err != nil {
			t.Fatal(err)
		}

		_ = utils.WaitForFreightToBeVerified(ctx, t, watchClient, project, freightID, stage, 10*time.Minute)

		return ctx
	})

	feature.Assess("promote uat", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		freightID := ctx.Value("freight_id").(string)
		project := ctx.Value("project").(string)
		kargoClient := ctx.Value("kargo_cli").(generated.KargoAPI)
		watchClient := ctx.Value("kargo_watch").(watch.Client)
		stage := "uat"

		_, err := utils.PromoteAndWaitForPhase(ctx, t, kargoClient, watchClient, project, stage, freightID, kargoapi.PromotionPhaseSucceeded, 10*time.Minute)
		if err != nil {
			t.Fatal(err)
		}

		_ = utils.WaitForFreightToBeVerified(ctx, t, watchClient, project, freightID, stage, 10*time.Minute)

		return ctx
	})

	feature.Assess("promote prod", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		freightID := ctx.Value("freight_id").(string)
		project := ctx.Value("project").(string)
		kargoClient := ctx.Value("kargo_cli").(generated.KargoAPI)
		watchClient := ctx.Value("kargo_watch").(watch.Client)

		stage := "prod"

		_, err := utils.PromoteAndWaitForPhase(ctx, t, kargoClient, watchClient, project, stage, freightID, kargoapi.PromotionPhaseSucceeded, 10*time.Minute)
		if err != nil {
			t.Fatal(err)
		}

		_ = utils.WaitForFreightToBeVerified(ctx, t, watchClient, project, freightID, stage, 10*time.Minute)

		return ctx
	})

	utils.TestEnv.Test(t, feature.Feature())
}
