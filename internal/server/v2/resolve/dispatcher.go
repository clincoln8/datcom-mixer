package resolve

import (
	"context"
	"log"
	"sync"

	"github.com/datacommonsorg/mixer/internal/maps"
	pbv2 "github.com/datacommonsorg/mixer/internal/proto/v2"
	"github.com/datacommonsorg/mixer/internal/server/resource"
	"github.com/datacommonsorg/mixer/internal/store"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	dispatcherInstance *Dispatcher
	dispatcherOnce     sync.Once
)

// VertexResolution defines the interface for Vertex AI resolution.
type VertexResolution interface {
	Resolve(ctx context.Context, in *pbv2.ResolveRequest) (*pbv2.ResolveResponse, error)
}

// PlaceResolution defines the interface for Place resolution.
type PlaceResolution interface {
	Resolve(ctx context.Context, in *pbv2.ResolveRequest, store *store.Store, mapsClient maps.MapsClient) (*pbv2.ResolveResponse, error)
}

// Dispatcher routes resolution requests to the appropriate backend.
type Dispatcher struct {
	vertexAI VertexResolution
	place    PlaceResolution
}

// GetDispatcher returns the singleton Dispatcher instance.
func GetDispatcher() *Dispatcher {
	dispatcherOnce.Do(func() {
		// Hardcoded values for now as requested
		client, err := NewVertexAIClient(context.Background(), "datcom-nl", "global")
		if err != nil {
			log.Printf("Failed to create Vertex AI client: %v", err)
			// Proceed with nil client; Resolver will return FailedPrecondition if called
			dispatcherInstance = NewDispatcher(nil)
		} else {
			dispatcherInstance = NewDispatcher(client)
		}
	})
	return dispatcherInstance
}

// NewDispatcher creates a new Dispatcher.
func NewDispatcher(client VertexAI) *Dispatcher {
	return &Dispatcher{
		vertexAI: NewVertexAIResolver(client),
		place:    NewPlaceResolver(),
	}
}

// Dispatch routes the request.
func (d *Dispatcher) Dispatch(
	ctx context.Context,
	in *pbv2.ResolveRequest,
	store *store.Store,
	mapsClient maps.MapsClient,
	metadata *resource.Metadata,
) (*pbv2.ResolveResponse, error) {
	resolver := in.GetSpecializedResolver()
	if resolver == "" {
		resolver = "embeddings"
	}

	var resp *pbv2.ResolveResponse
	var err error

	switch {
	case resolver == "place":
		resp, err = d.place.Resolve(ctx, in, store, mapsClient)

	case len(resolver) > 9 && resolver[:9] == "vertexai:":
		resp, err = d.vertexAI.Resolve(ctx, in)

	case resolver == "embeddings":
		return nil, status.Error(codes.Unimplemented, "embeddings resolver not yet implemented")

	default:
		return nil, status.Errorf(codes.InvalidArgument, "unknown specialized resolver: %s", resolver)
	}

	if err != nil {
		return nil, err
	}

	// 1. Pre-fetch properties needed for filtering
	// Optimization: Only fetch properties used in filters for ALL candidates.
	if filters := in.GetFilters(); filters != nil && len(filters.Fields) > 0 {
		var filterProps []string
		for k := range filters.Fields {
			filterProps = append(filterProps, k)
		}
		// Only enrich if we actually have property filters (typeOf is handled via DominantType if prop missing, but good to have)
		if len(filterProps) > 0 {
			if err := EnrichResponse(ctx, store, metadata, resp, filterProps); err != nil {
				return nil, err
			}
		}
	}

	// 2. Standardize (Filter + Limit)
	// This reduces the candidate list significantly.
	resp = StandardizeResponse(in, resp)

	// 3. Post-fetch requested properties for the remaining candidates
	// We run this even if returned_properties is empty to ensure default properties (like typeOf) are populated.
	// Optimization: We accept slight redundancy (re-fetching filter props) for the small list of survivors
	// in exchange for not fetching returned_properties for the dropped candidates.
	err = EnrichResponse(ctx, store, metadata, resp, in.GetReturnedProperties())
	if err != nil {
		return nil, err
	}

	// Final Cleanup: Remove empty properties maps to avoid "{}" in JSON
	if resp != nil {
		for _, entity := range resp.Entities {
			for _, candidate := range entity.Candidates {
				if candidate.Properties != nil && len(candidate.Properties.Fields) == 0 {
					candidate.Properties = nil
				}
			}
		}
	}

	return resp, nil
}
