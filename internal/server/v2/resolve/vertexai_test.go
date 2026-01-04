package resolve

import (
	"context"
	"fmt"
	"strings"
	"testing"

	pbv2 "github.com/datacommonsorg/mixer/internal/proto/v2"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/testing/protocmp"
)

// MockVertexAIClient is a mock implementation of VertexAI.
type MockVertexAIClient struct {
	PredictFunc func(ctx context.Context, appID string, nodes []string) (map[string][]string, error)
}

func (m *MockVertexAIClient) Close() error {
	return nil
}

func (m *MockVertexAIClient) Predict(ctx context.Context, appID string, nodes []string) (map[string][]string, error) {
	if m.PredictFunc != nil {
		return m.PredictFunc(ctx, appID, nodes)
	}
	return nil, nil
}

func TestVertexAIResolver_Resolve(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		desc             string
		req              *pbv2.ResolveRequest
		mockClient       *MockVertexAIClient
		want             *pbv2.ResolveResponse
		wantCode         codes.Code
		wantErrorContains string
	}{
		{
			desc: "Valid App ID (embedding-statvars) -> Success",
			req: &pbv2.ResolveRequest{
				Nodes:               []string{"foo"},
				SpecializedResolver: "vertexai_embedding-statvars",
			},
			mockClient: &MockVertexAIClient{
				PredictFunc: func(ctx context.Context, appID string, nodes []string) (map[string][]string, error) {
					if appID != "nl-statvar-search-staging_1753469464090" {
						return nil, fmt.Errorf("unexpected appID: %s", appID)
					}
					return map[string][]string{
						"foo": {"sv_1", "sv_2"},
					}, nil
				},
			},
			want: &pbv2.ResolveResponse{
				Entities: []*pbv2.ResolveResponse_Entity{
					{
						Node: "foo",
						Candidates: []*pbv2.ResolveResponse_Entity_Candidate{
							{Dcid: "sv_1", DominantType: "StatisticalVariable"},
							{Dcid: "sv_2", DominantType: "StatisticalVariable"},
						},
					},
				},
			},
			wantCode: codes.OK,
		},
		{
			desc: "Invalid App ID (unknown-app) -> Error",
			req: &pbv2.ResolveRequest{
				Nodes:               []string{"foo"},
				SpecializedResolver: "vertexai_unknown-app",
			},
			mockClient: &MockVertexAIClient{},
			wantCode:   codes.Unimplemented,
			wantErrorContains: "unsupported Vertex AI app",
		},
		{
			desc: "Client Error -> Internal Error",
			req: &pbv2.ResolveRequest{
				Nodes:               []string{"foo"},
				SpecializedResolver: "vertexai_embedding-statvars",
			},
			mockClient: &MockVertexAIClient{
				PredictFunc: func(ctx context.Context, appID string, nodes []string) (map[string][]string, error) {
					return nil, fmt.Errorf("vertex error")
				},
			},
			wantCode:          codes.Internal,
			wantErrorContains: "Vertex AI prediction failed",
		},
		{
			desc: "Client Not Initialized -> FailedPrecondition",
			req: &pbv2.ResolveRequest{
				Nodes:               []string{"foo"},
				SpecializedResolver: "vertexai_embedding-statvars",
			},
			mockClient: nil, // Trigger r.client == nil check
			wantCode:   codes.FailedPrecondition,
			wantErrorContains: "Vertex AI client not initialized",
		},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			var client VertexAI
			if tc.mockClient != nil {
				client = tc.mockClient
			}
			// If mockClient is nil, NewVertexAIResolver gets nil client, which is what we want for that test case
			// But NewVertexAIResolver accepts interface.
			resolver := NewVertexAIResolver(client)
			resp, err := resolver.Resolve(ctx, tc.req)

			if err != nil {
				if tc.wantCode == codes.OK {
					t.Errorf("got error %v, want success", err)
				}
				s, ok := status.FromError(err)
				if !ok {
					t.Errorf("got non-status error: %v", err)
				}
				if s.Code() != tc.wantCode {
					t.Errorf("got code %s, want %s", s.Code(), tc.wantCode)
				}
				if tc.wantErrorContains != "" {
					if !strings.Contains(s.Message(), tc.wantErrorContains) {
						t.Errorf("got message %q, want to contain %q", s.Message(), tc.wantErrorContains)
					}
				}
				return
			}

			if tc.wantCode != codes.OK {
				t.Errorf("got success, want error code %s", tc.wantCode)
			}

			if diff := cmp.Diff(tc.want, resp, protocmp.Transform()); diff != "" {
				t.Errorf("Resolve() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
