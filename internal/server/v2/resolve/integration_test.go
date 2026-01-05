package resolve

import (
	"context"
	"testing"

	pbv2 "github.com/datacommonsorg/mixer/internal/proto/v2"
	"github.com/datacommonsorg/mixer/internal/store"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestIntegration_Resolve(t *testing.T) {
	// 1. Setup Mocks
	mockVertex := &MockVertexResolver{
		ResolveFunc: func(ctx context.Context, in *pbv2.ResolveRequest) (*pbv2.ResolveResponse, error) {
			// Simulate Vertex Returning a generic response
			return &pbv2.ResolveResponse{
				Entities: []*pbv2.ResolveResponse_Entity{
					{
						Node: "health insurance",
						Candidates: []*pbv2.ResolveResponse_Entity_Candidate{
							{Dcid: "topic/NoHealthInsurance", DominantType: "Topic"},
							{Dcid: "Count_Person_NoHealthInsurance", DominantType: "StatisticalVariable"},
						},
					},
				},
			}, nil
		},
	}

	mockPlace := &MockPlaceResolver{} // Not used in this test case

	// 2. Create Dispatcher with Mocks
	d := &Dispatcher{
		vertexAI: mockVertex,
		place:    mockPlace,
	}

	// 3. Define Request with Filters (Test Phase 3 Logic)
	// We want to filter for 'StatisticalVariable' only.
	filters, _ := structpb.NewStruct(map[string]interface{}{"typeOf": "StatisticalVariable"})
	req := &pbv2.ResolveRequest{
		Nodes:               []string{"health insurance"},
		SpecializedResolver: "vertexai_embedding-statvars",
		Filters:             filters,
	}

	// 4. Run Dispatch (simulate handler call)
	// Passing nil store/maps is fine because MockVertex doesn't use them
	resp, err := d.Dispatch(context.Background(), req, &store.Store{}, nil)
	if err != nil {
		t.Fatalf("Dispatch failed: %v", err)
	}

	// 5. Verify Output (Golden Check Logic)
	// We expect only 1 candidate to remain after filtering.
	want := &pbv2.ResolveResponse{
		Entities: []*pbv2.ResolveResponse_Entity{
			{
				Node: "health insurance",
				Candidates: []*pbv2.ResolveResponse_Entity_Candidate{
					{Dcid: "Count_Person_NoHealthInsurance", DominantType: "StatisticalVariable"},
				},
			},
		},
	}

	if diff := cmp.Diff(want, resp, protocmp.Transform()); diff != "" {
		t.Errorf("Integration Test mismatch (-want +got):\n%s", diff)
	}
}
