package resolve

import (
	"testing"

	pbv2 "github.com/datacommonsorg/mixer/internal/proto/v2"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/testing/protocmp"
)

func TestFilterCandidates(t *testing.T) {
	candidates := []*pbv2.ResolveResponse_Entity_Candidate{
		{Dcid: "A", DominantType: "City"},
		{Dcid: "B", DominantType: "Country"},
		{Dcid: "C", DominantType: "City"},
	}

	tests := []struct {
		desc    string
		filters map[string]string
		want    []*pbv2.ResolveResponse_Entity_Candidate
	}{
		{
			desc:    "No filters -> All candidates",
			filters: nil,
			want:    candidates,
		},
		{
			desc:    "Filter by Type City",
			filters: map[string]string{"typeOf": "City"},
			want: []*pbv2.ResolveResponse_Entity_Candidate{
				{Dcid: "A", DominantType: "City"},
				{Dcid: "C", DominantType: "City"},
			},
		},
		{
			desc:    "Filter by Type Country",
			filters: map[string]string{"typeOf": "Country"},
			want: []*pbv2.ResolveResponse_Entity_Candidate{
				{Dcid: "B", DominantType: "Country"},
			},
		},
		{
			desc:    "Filter by Non-matching Type",
			filters: map[string]string{"typeOf": "Mountain"},
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
