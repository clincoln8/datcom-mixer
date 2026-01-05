package resolve

import (
	"testing"

	pbv2 "github.com/datacommonsorg/mixer/internal/proto/v2"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestFilterCandidates(t *testing.T) {
	candidates := []*pbv2.ResolveResponse_Entity_Candidate{
		{Dcid: "A", DominantType: "City"},
		{Dcid: "B", DominantType: "Country"},
		{Dcid: "C", DominantType: "City"},
		{
			Dcid: "D",
			Properties: &structpb.Struct{
				Fields: map[string]*structpb.Value{
					"gender": structpb.NewListValue(&structpb.ListValue{
						Values: []*structpb.Value{structpb.NewStringValue("Female")},
					}),
				},
			},
		},
		{
			Dcid: "E",
			Properties: &structpb.Struct{
				Fields: map[string]*structpb.Value{
					"^containedInPlace": structpb.NewListValue(&structpb.ListValue{
						Values: []*structpb.Value{
							structpb.NewStructValue(&structpb.Struct{
								Fields: map[string]*structpb.Value{
									"dcid": structpb.NewStringValue("geoId/06"),
								},
							}),
						},
					}),
				},
			},
		},
	}

	tests := []struct {
		desc    string
		filters map[string][]string
		want    []*pbv2.ResolveResponse_Entity_Candidate
	}{
		{
			desc:    "No filters -> All candidates",
			filters: nil,
			want:    candidates,
		},
		{
			desc:    "Filter by Type City",
			filters: map[string][]string{"typeOf": {"City"}},
			want: []*pbv2.ResolveResponse_Entity_Candidate{
				{Dcid: "A", DominantType: "City"},
				{Dcid: "C", DominantType: "City"},
			},
		},
		{
			desc:    "Filter by Type Country",
			filters: map[string][]string{"typeOf": {"Country"}},
			want: []*pbv2.ResolveResponse_Entity_Candidate{
				{Dcid: "B", DominantType: "Country"},
			},
		},
		{
			desc:    "Filter by Multiple Types (OR Logic)",
			filters: map[string][]string{"typeOf": {"City", "Town"}},
			want: []*pbv2.ResolveResponse_Entity_Candidate{
				{Dcid: "A", DominantType: "City"},
				{Dcid: "C", DominantType: "City"},
			},
		},
		{
			desc:    "Filter by Arbitrary Property (Literal Match)",
			filters: map[string][]string{"gender": {"Female"}},
			want: []*pbv2.ResolveResponse_Entity_Candidate{
				{
					Dcid: "D",
					Properties: &structpb.Struct{
						Fields: map[string]*structpb.Value{
							"gender": structpb.NewListValue(&structpb.ListValue{
								Values: []*structpb.Value{structpb.NewStringValue("Female")},
							}),
						},
					},
				},
			},
		},
		{
			desc:    "Filter by Arbitrary Property (OR Logic)",
			filters: map[string][]string{"gender": {"Male", "Female"}},
			want: []*pbv2.ResolveResponse_Entity_Candidate{
				{
					Dcid: "D",
					Properties: &structpb.Struct{
						Fields: map[string]*structpb.Value{
							"gender": structpb.NewListValue(&structpb.ListValue{
								Values: []*structpb.Value{structpb.NewStringValue("Female")},
							}),
						},
					},
				},
			},
		},
		{
			desc:    "Filter by Arbitrary Property (Node Match)",
			filters: map[string][]string{"^containedInPlace": {"geoId/06"}},
			want: []*pbv2.ResolveResponse_Entity_Candidate{
				{
					Dcid: "E",
					Properties: &structpb.Struct{
						Fields: map[string]*structpb.Value{
							"^containedInPlace": structpb.NewListValue(&structpb.ListValue{
								Values: []*structpb.Value{
									structpb.NewStructValue(&structpb.Struct{
										Fields: map[string]*structpb.Value{
											"dcid": structpb.NewStringValue("geoId/06"),
										},
									}),
								},
							}),
						},
					},
				},
			},
		},
		{
			desc:    "Filter by Non-matching Type",
			filters: map[string][]string{"typeOf": {"Mountain"}},
			want:    nil,
		},
		{
			desc:    "Filter by Non-matching Property",
			filters: map[string][]string{"gender": {"Male"}},
			want:    nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			got := FilterCandidates(candidates, tc.filters)
			if diff := cmp.Diff(tc.want, got, protocmp.Transform()); diff != "" {
				t.Errorf("FilterCandidates mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
