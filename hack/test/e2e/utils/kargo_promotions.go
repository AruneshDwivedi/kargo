package utils

import (
	"context"
	"errors"
	"testing"
	"time"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/client/generated"
	"github.com/akuity/kargo/pkg/client/generated/core"
	"github.com/akuity/kargo/pkg/client/generated/models"
	"github.com/akuity/kargo/pkg/client/watch"
)

func PromoteAndWaitForPhase(
	ctx context.Context,
	t *testing.T,
	kargoClient generated.KargoAPI,
	watchClient watch.Client,
	project, stage, freightName string,
	phase kargoapi.PromotionPhase,
	timeout time.Duration,
) (*kargoapi.Promotion, error) {
	promotion, err := PromoteAndWaitForCompletion(ctx, t, kargoClient, watchClient, project, stage, freightName, timeout)
	if err != nil {
		return nil, err
	}
	if promotion.Status.Phase != phase {
		t.Fatalf(
			"Promotion '%v' did not finish with phase '%v', actual phase: '%v'",
			promotion.Name, phase, promotion.Status.Phase)
	}
	return promotion, err
}

// func PromoteDownstream(
// 	t *testing.T,
// 	kargoClient generated.KargoAPI,
// 	project, stage, freightName string,
// 	timeout time.Duration,
// ) {

// 	promoteRes, err := kargoClient.Core.PromoteDownstream(
// 		core.NewPromoteDownstreamParams().
// 			WithProject(project).
// 			WithStage(stage).
// 			WithBody(&models.PromoteDownstreamRequest{
// 				Freight: freightName,
// 			}),
// 		nil,
// 	)

// 	if err != nil {
// 		t.Fatalf("Error promoting %v", err)
// 	}

// 	promoName := promoteRes.Payload.Metadata.Name

// 	promotion, err := WaitForPromotion(t, kargoClient, project, promoName, 5*time.Minute)

// 	if err != nil {
// 		t.Fatalf("Error getting promotion %v", err)
// 	}
// 	return promotion, nil
// }

func PromoteAndWaitForCompletion(
	ctx context.Context,
	t *testing.T,
	kargoClient generated.KargoAPI,
	watchClient watch.Client,
	project, stage, freightName string,
	timeout time.Duration,
) (*kargoapi.Promotion, error) {

	if _, err := kargoClient.Core.GetStage(
		core.NewGetStageParams().WithProject(project).WithStage(stage),
		nil,
	); err != nil {
		t.Fatalf("error getting stage: %v", err)
	}

	promoteRes, err := kargoClient.Core.PromoteToStage(
		core.NewPromoteToStageParams().
			WithProject(project).
			WithStage(stage).
			WithBody(&models.PromoteToStageRequest{
				Freight: freightName,
			}),
		nil,
	)
	if err != nil {
		t.Fatalf("Error promoting %v", err)
	}

	promoName := promoteRes.Payload.Metadata.Name
	promotion, err := WaitForPromotion(ctx, t, watchClient, project, promoName, timeout)

	if err != nil {
		t.Fatalf("Error getting promotion %v", err)
	}
	return promotion, nil

}

func WaitForPromotion(
	ctx context.Context,
	t *testing.T,
	watchClient watch.Client,
	project, name string,
	timeout time.Duration,
) (*kargoapi.Promotion, error) {
	timedCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	watchChan, errorChan := watchClient.WatchPromotion(timedCtx, project, name)
	for {
		select {
		case event := <-watchChan:
			if event.Object != nil {
				phase := event.Object.Status.Phase
				if phase == "" || phase == kargoapi.PromotionPhaseRunning || phase == kargoapi.PromotionPhasePending {
					continue
				}
				return event.Object, nil
			}
		case err := <-errorChan:
			return nil, err
		case <-timedCtx.Done():
			return nil, errors.New("context canceled")
		}
	}
}

func WaitForLatestFreight(ctx context.Context, watchClient watch.Client, project, origin string, timeout time.Duration) (string, error) {
	timedCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	watchChan, errorChan := watchClient.WatchWarehouse(timedCtx, project, origin)
	for {
		select {
		case event := <-watchChan:
			if event.Object != nil {
				return event.Object.Status.LastFreightID, nil
			}
		case err := <-errorChan:
			return "", err
		case <-timedCtx.Done():
			return "", errors.New("context canceled")
		}
	}
}

func WaitForFreight(ctx context.Context, watchClient watch.Client, project, freightID string, timeout time.Duration, filter func(*kargoapi.Freight) bool) (*kargoapi.Freight, error) {
	timedCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	watchChan, errorChan := watchClient.WatchFreight(timedCtx, project, freightID)
	for {
		select {
		case event := <-watchChan:
			if filter(event.Object) {
				return event.Object, nil
			}
		case err := <-errorChan:
			return nil, err
		case <-timedCtx.Done():
			return nil, errors.New("context canceled")
		}
	}
}


func WaitForFreightToBeVerified(ctx context.Context, t *testing.T, watchClient watch.Client, project, freightID, stage string, timeout time.Duration) *kargoapi.Freight {
	freight, err := WaitForFreight(ctx, watchClient, project, freightID, 10*time.Minute, func(freight *kargoapi.Freight) bool {
		if freight != nil {
			_, ok := freight.Status.VerifiedIn[stage]
			return ok
		}
		return false
	})
	if err != nil {
		t.Fatalf("Error waiting for freight to be verified %v", err)
	}
	return freight 
}

// func getAnyFreight(kargoClient generated.KargoAPI, project, origin string) (*kargoapi.Freight, error) {

// 	params := core.NewQueryFreightsRestParams().WithProject(project).WithOrigins([]string{origin})

// 	freightRes, err := kargoClient.Core.QueryFreightsRest(params, nil)
// 	if err != nil {
// 		return nil, fmt.Errorf("Error querying freight %v", err)
// 	}

// 	// FIXME: change that once we make freight response typed
// 	var freightJSON []byte
// 	if freightJSON, err = json.Marshal(freightRes.Payload); err != nil {
// 		return nil, fmt.Errorf("marshal freight: %w", err)
// 	}
// 	// The response is {"groups": {"": {"items": [...]}}}
// 	type freightList struct {
// 		Items []*kargoapi.Freight `json:"items"`
// 	}
// 	var result struct {
// 		Groups map[string]*freightList `json:"groups"`
// 	}
// 	if err = json.Unmarshal(freightJSON, &result); err != nil {
// 		return nil, fmt.Errorf("unmarshal freight: %v", err)
// 	}
// 	freights := result.Groups[""].Items
// 	if len(freights) < 1 {
// 		return nil, fmt.Errorf("no freights found")
// 	}
// 	return freights[0], nil
// }
