package kargo_promotion_fail_test

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/akuity/kargo/hack/test/e2e/utils"
	"testing"
	"time"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/client/generated"
	"github.com/akuity/kargo/pkg/client/generated/core"
	"github.com/akuity/kargo/pkg/client/generated/models"
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

		anyFreight, err  := waitForFreight(kargoClient, project, origin, 5*time.Minute)
		if err != nil {
			t.Fatal(err)
		}

		_, err = kargoClient.Core.GetStage(
			core.NewGetStageParams().WithProject(project).WithStage(stage),
			nil,
		)
		if err != nil {
			t.Fatalf("get stage: %v", err)
		}

		promoteRes, err := kargoClient.Core.PromoteToStage(
			core.NewPromoteToStageParams().
				WithProject(project).
				WithStage(stage).
				WithBody(&models.PromoteToStageRequest{
					FreightAlias: anyFreight.Alias,
				}),
			nil,
		)
		if err != nil {
			t.Fatalf("Error promoting %v", err)
		}

		promoName := promoteRes.Payload.Metadata.Name

		promotion, err := waitForPromotion(t, kargoClient, project, promoName, 5*time.Minute)

		if err != nil {
			t.Fatalf("Error getting promotion %v", err)
		}
		if promotion.Payload.Status.Phase != string(kargoapi.PromotionPhaseFailed) {
			t.Fatalf("Promotion is not failed: '%v'", promotion.Payload.Status.Phase)
		}
		return ctx
	})

	utils.TestEnv.Test(t, feature.Feature())

}

func waitForPromotion(t *testing.T, kargoClient generated.KargoAPI, project, name string, timeout time.Duration) (*core.GetPromotionOK, error) {
	start := time.Now()
	t.Log("Waiting for promotion to finish...")
	for {
		if time.Since(start) > timeout {

			return nil, fmt.Errorf("timeout waiting for promotion")
		}
		promotion, err := kargoClient.Core.GetPromotion(
			core.NewGetPromotionParams().WithProject(project).WithPromotion(name),
			nil)
		if err != nil {
			return nil, err
		}
		phase := promotion.Payload.Status.Phase
		if phase == "" || phase == string(kargoapi.PromotionPhaseRunning) || phase == string(kargoapi.PromotionPhasePending) {
			time.Sleep(100)
			continue
		}
		return promotion, err
	}
}

func waitForFreight(kargoClient generated.KargoAPI, project, origin string, timeout time.Duration) (*kargoapi.Freight, error) {
	start := time.Now()
	for {
		if time.Since(start) > timeout {
			return nil, fmt.Errorf("timeout waiting for freight")
		}
		freight, err := getAnyFreight(kargoClient, project, origin)
		if err != nil {
			if err.Error() == "no freights found" {
				time.Sleep(100)
				continue
			}
			return nil, err
		}
		return freight, err
	}
}

func getAnyFreight(kargoClient generated.KargoAPI, project, origin string) (*kargoapi.Freight, error) {

	params := core.NewQueryFreightsRestParams().WithProject(project).WithOrigins([]string{origin})

	freightRes, err := kargoClient.Core.QueryFreightsRest(params, nil)
	if err != nil {
		return nil, fmt.Errorf("Error querying freight %v", err)
	}

	// FIXME: change that once we make freight response typed
	var freightJSON []byte
	if freightJSON, err = json.Marshal(freightRes.Payload); err != nil {
		return nil, fmt.Errorf("marshal freight: %w", err)
	}
	// The response is {"groups": {"": {"items": [...]}}}
	type freightList struct {
		Items []*kargoapi.Freight `json:"items"`
	}
	var result struct {
		Groups map[string]*freightList `json:"groups"`
	}
	if err = json.Unmarshal(freightJSON, &result); err != nil {
		return nil, fmt.Errorf("unmarshal freight: %v", err)
	}
	freights := result.Groups[""].Items
	if len(freights) < 1 {
		return nil, fmt.Errorf("no freights found")
	}
	return freights[0], nil
}
