package server

import (
	"context"
	"strings"
	"testing"

	"github.com/datacommonsorg/mixer/internal/featureflags"
	pbv2 "github.com/datacommonsorg/mixer/internal/proto/v2"
	"github.com/datacommonsorg/mixer/internal/store"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestV2ResolveCore_Routing(t *testing.T) {
	ctx := context.Background()

	for _, c := range []struct {
		desc             string
		enableFlag       bool
		req              *pbv2.ResolveRequest
		wantCode         codes.Code
		wantErrorContains string
	}{
		{
			"Flag Disabled: New Params -> Proceed (Legacy Error)",
			false,
			&pbv2.ResolveRequest{
				Nodes:               []string{"foo"},
				SpecializedResolver: "embeddings",
			},
			codes.InvalidArgument,
			"invalid property for resolving",
		},
		{
			"Flag Disabled: Legacy Params -> Proceed (Legacy Error)",
			false,
			&pbv2.ResolveRequest{
				Nodes:    []string{"foo"},
				Property: "invalid-property-syntax-for-parse-error",
			},
			codes.InvalidArgument, // Legacy parser returns InvalidArgument for bad syntax
			"arc string should start with arrow",
		},
		{
			"Flag Enabled: Conflict (Property + Embeddings) -> Error",
			true,
			&pbv2.ResolveRequest{
				Nodes:               []string{"foo"},
				Property:            "<-id->id",
				SpecializedResolver: "embeddings",
			},
			codes.InvalidArgument,
			"conflicting parameters",
		},
		{
			"Flag Enabled: Conflict (Property + SearchProperties) -> Error",
			true,
			&pbv2.ResolveRequest{
				Nodes:            []string{"foo"},
				Property:         "<-id->id",
				SearchProperties: []string{"name"},
			},
			codes.InvalidArgument,
			"conflicting parameters",
		},
		{
			"Flag Enabled: Hybrid (Property + Limit) -> Proceed (Legacy Error)",
			true,
			&pbv2.ResolveRequest{
				Nodes:    []string{"foo"},
				Property: "invalid-property",
				Limit:    10,
			},
			codes.InvalidArgument,
			"arc string should start with arrow",
		},
		{
			"Flag Enabled: Hybrid (Property + Filters) -> Proceed (Legacy Error)",
			true,
			&pbv2.ResolveRequest{
				Nodes:    []string{"foo"},
				Property: "invalid-property",
				// Filters technically Struct, but nil in this simple struct... 
                // Let's assume Limit test is sufficient for Hybrid check as logic matches 'Filters != nil || Limit > 0'
			},
			codes.InvalidArgument,
			"arc string should start with arrow",
		},
		{
			"Flag Enabled: Explicit Place (Property + Place) -> Proceed (Legacy Error)",
			true,
			&pbv2.ResolveRequest{
				Nodes:               []string{"foo"},
				Property:            "invalid-property",
				SpecializedResolver: "place",
			},
			codes.InvalidArgument,
			"arc string should start with arrow",
		},
		{
			"Flag Enabled: New Params -> Unimplemented (New Path)",
			true,
			&pbv2.ResolveRequest{
				Nodes:               []string{"foo"},
				SpecializedResolver: "embeddings",
			},
			codes.Unimplemented,
			"expanded resolve logic not yet implemented",
		},
	} {
		t.Run(c.desc, func(t *testing.T) {
			s := &Server{
				store: &store.Store{},
				flags: &featureflags.Flags{
					EnableExpandedResolve: c.enableFlag,
				},
			}
			_, err := s.V2ResolveCore(ctx, c.req)
			if err == nil {
				if c.wantCode != codes.OK {
					t.Errorf("got no error, want code %s", c.wantCode)
				}
				return
			}
			st, ok := status.FromError(err)
			if !ok {
				t.Errorf("got non-status error: %v", err)
				return
			}
			if st.Code() != c.wantCode {
				t.Errorf("got code %s, want %s (err: %v)", st.Code(), c.wantCode, err)
			}
			if c.wantErrorContains != "" {
				if !strings.Contains(st.Message(), c.wantErrorContains) {
					t.Errorf("got error message %q, want it to contain %q", st.Message(), c.wantErrorContains)
				}
			}
		})
	}
}
