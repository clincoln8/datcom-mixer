package resolve

import (
	"context"
	"strings"
	"testing"

	pbv2 "github.com/datacommonsorg/mixer/internal/proto/v2"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestDispatcher_Dispatch(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		desc             string
		req              *pbv2.ResolveRequest
		wantCode         codes.Code
		wantErrorContains string
	}{
		{
			desc: "Resolver: vertexai_embedding-statvars -> Routes to VertexAI (Client Error meant routing worked)",
			req: &pbv2.ResolveRequest{
				Nodes:               []string{"foo"},
				SpecializedResolver: "vertexai_embedding-statvars",
			},
			// Client Mock will be nil (dummy) so VertexAIResolver checks client and returns FailedPrecondition
			// This proves Dispatcher routed to VertexAIResolver
			wantCode:          codes.FailedPrecondition,
			wantErrorContains: "Vertex AI client not initialized",
		},
		{
			desc: "Resolver: place -> Unimplemented",
			req: &pbv2.ResolveRequest{
				Nodes:               []string{"foo"},
				SpecializedResolver: "place",
			},
			wantCode:          codes.Unimplemented,
			wantErrorContains: "expanded 'place' resolution",
		},
		{
			desc: "Resolver: embeddings -> Unimplemented",
			req: &pbv2.ResolveRequest{
				Nodes:               []string{"foo"},
				SpecializedResolver: "embeddings",
			},
			wantCode:          codes.Unimplemented,
			wantErrorContains: "embeddings resolver not yet implemented",
		},
		{
			desc: "Resolver: unknown -> InvalidArgument",
			req: &pbv2.ResolveRequest{
				Nodes:               []string{"foo"},
				SpecializedResolver: "unknown-resolver",
			},
			wantCode:          codes.InvalidArgument,
			wantErrorContains: "unknown specialized resolver",
		},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			// We pass nil client, which is enough to test routing because:
			// 1. If routed to VertexAI, it hits 'client not initialized' error.
			// 2. If routed elsewhere, it hits Unimplemented or InvalidArgument.
			dispatcher := NewDispatcher(nil)
			_, err := dispatcher.Dispatch(ctx, tc.req)

			if err == nil {
				t.Errorf("got success, want error code %s", tc.wantCode)
				return
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
		})
	}
}
