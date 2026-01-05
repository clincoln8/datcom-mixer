package resolve

import (
	"testing"

	pbv2 "github.com/datacommonsorg/mixer/internal/proto/v2"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestStandardizeResponse(t *testing.T) {
	filters, _ := structpb.NewStruct(map[string]interface{}{"typeOf": "City"})
	limit := int32(2)
	req := &pbv2.ResolveRequest{
		Limit:   &limit,
		Filters: filters,
	}

	rawResp := &pbv2.ResolveResponse{
		Entities: []*pbv2.ResolveResponse_Entity{
			{
				Node: "foo",
				Candidates: []*pbv2.ResolveResponse_Entity_Candidate{
					{Dcid: "A", DominantType: "City"},
					{Dcid: "B", DominantType: "Country"},
					{Dcid: "C", DominantType: "City"},
					{Dcid: "D", DominantType: "City"},
				},
			},
		},
	}

	wantResp := &pbv2.ResolveResponse{
		Entities: []*pbv2.ResolveResponse_Entity{
			{
				Node: "foo",
				Candidates: []*pbv2.ResolveResponse_Entity_Candidate{
					{Dcid: "A", DominantType: "City"},
					{Dcid: "C", DominantType: "City"},
				},
			},
		},
	}

	gotResp := StandardizeResponse(req, rawResp)

	if diff := cmp.Diff(wantResp, gotResp, protocmp.Transform()); diff != "" {
		t.Errorf("StandardizeResponse mismatch (-want +got):\n%s", diff)
	}
}

func TestStandardizeResponse_NoLimit(t *testing.T) {
	limit := int32(-1)
	req := &pbv2.ResolveRequest{
		Limit: &limit, // Explicitly -1 (Unlimited)
	}

	// Create 15 candidates
	candidates := []*pbv2.ResolveResponse_Entity_Candidate{}
	for i := 0; i < 15; i++ {
		candidates = append(candidates, &pbv2.ResolveResponse_Entity_Candidate{Dcid: "id"})
	}

	rawResp := &pbv2.ResolveResponse{
		Entities: []*pbv2.ResolveResponse_Entity{
			{Node: "foo", Candidates: candidates},
		},
	}

	gotResp := StandardizeResponse(req, rawResp)

	// Expect ALL 15 candidates because StandardizeResponse no longer defaults to 10.
	if len(gotResp.Entities[0].Candidates) != 15 {
		t.Errorf("Expected 15 candidates, got %d", len(gotResp.Entities[0].Candidates))
	}
}

func TestStandardizeResponse_ExplicitZeroLimit(t *testing.T) {
	limit := int32(0)
	req := &pbv2.ResolveRequest{
		Limit: &limit, // Explicitly 0
	}

	// Create candidates
	candidates := []*pbv2.ResolveResponse_Entity_Candidate{{Dcid: "A"}}
	rawResp := &pbv2.ResolveResponse{
		Entities: []*pbv2.ResolveResponse_Entity{{Node: "foo", Candidates: candidates}},
	}

	gotResp := StandardizeResponse(req, rawResp)

	if len(gotResp.Entities[0].Candidates) != 0 {
		t.Errorf("Expected 0 candidates, got %d", len(gotResp.Entities[0].Candidates))
	}
}

func TestStandardizeResponse_EmptyPropertiesCleanup(t *testing.T) {
	req := &pbv2.ResolveRequest{}

	// Candidate with empty Struct properties
	candidates := []*pbv2.ResolveResponse_Entity_Candidate{
		{
			Dcid:       "A",
			Properties: &structpb.Struct{Fields: map[string]*structpb.Value{}},
		},
	}
	rawResp := &pbv2.ResolveResponse{
		Entities: []*pbv2.ResolveResponse_Entity{{Node: "foo", Candidates: candidates}},
	}

	gotResp := StandardizeResponse(req, rawResp)

	if gotResp.Entities[0].Candidates[0].Properties != nil {
		t.Errorf("Expected nil Properties, got %v", gotResp.Entities[0].Candidates[0].Properties)
	}
}
