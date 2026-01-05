package resolve

import (
	"context"

	pbv2 "github.com/datacommonsorg/mixer/internal/proto/v2"
	"github.com/datacommonsorg/mixer/internal/maps"
	"github.com/datacommonsorg/mixer/internal/store"
)

// MockVertexResolver implements VertexResolution for testing.
type MockVertexResolver struct {
	ResolveFunc func(ctx context.Context, in *pbv2.ResolveRequest) (*pbv2.ResolveResponse, error)
}

func (m *MockVertexResolver) Resolve(ctx context.Context, in *pbv2.ResolveRequest) (*pbv2.ResolveResponse, error) {
	if m.ResolveFunc != nil {
		return m.ResolveFunc(ctx, in)
	}
	return &pbv2.ResolveResponse{}, nil
}

// MockPlaceResolver implements PlaceResolution for testing.
type MockPlaceResolver struct {
	ResolveFunc func(ctx context.Context, in *pbv2.ResolveRequest, store *store.Store, mapsClient maps.MapsClient) (*pbv2.ResolveResponse, error)
}

func (m *MockPlaceResolver) Resolve(ctx context.Context, in *pbv2.ResolveRequest, store *store.Store, mapsClient maps.MapsClient) (*pbv2.ResolveResponse, error) {
	if m.ResolveFunc != nil {
		return m.ResolveFunc(ctx, in, store, mapsClient)
	}
	return &pbv2.ResolveResponse{}, nil
}
