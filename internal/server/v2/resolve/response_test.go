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
	req := &pbv2.ResolveRequest{
		Limit:   2,
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
