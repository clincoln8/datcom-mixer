package resolve

import (
	"context"

	pbv2 "github.com/datacommonsorg/mixer/internal/proto/v2"
	"github.com/datacommonsorg/mixer/internal/maps"
	"github.com/datacommonsorg/mixer/internal/store"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// PlaceResolver implements the resolution logic using legacy Place resolution.
type PlaceResolver struct{}

// NewPlaceResolver creates a new PlaceResolver.
func NewPlaceResolver() *PlaceResolver {
	return &PlaceResolver{}
}

// Resolve processes the resolution request using legacy logic.
// Currently maps strictly to Description resolution (Name -> DCID).
func (r *PlaceResolver) Resolve(
	ctx context.Context,
	in *pbv2.ResolveRequest,
	store *store.Store,
	mapsClient maps.MapsClient,
) (*pbv2.ResolveResponse, error) {
	// Check for nil dependencies to prevent panics in tests or misconfiguration
	if store == nil || store.BtGroup == nil {
		return nil, status.Error(codes.FailedPrecondition, "store or BigTable client not initialized")
	}

	// TODO: Add detection for Coordinate logic if needed, or separate specialized_resolver
	// For now, default to Description (Name resolution)
	// We pass nil for typeOfs to match generic behavior.
	return Description(ctx, store, mapsClient, in.GetNodes(), nil)
}
