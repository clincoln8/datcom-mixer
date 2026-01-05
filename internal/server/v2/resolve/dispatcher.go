package resolve

import (
	"context"
	"log"
	"sync"

	pbv2 "github.com/datacommonsorg/mixer/internal/proto/v2"
	"github.com/datacommonsorg/mixer/internal/maps"
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
func (d *Dispatcher) Dispatch(ctx context.Context, in *pbv2.ResolveRequest, store *store.Store, mapsClient maps.MapsClient) (*pbv2.ResolveResponse, error) {
	resolver := in.GetSpecializedResolver()

	var resp *pbv2.ResolveResponse
	var err error

	switch {
	case resolver == "place":
		resp, err = d.place.Resolve(ctx, in, store, mapsClient)

	case len(resolver) > 9 && resolver[:9] == "vertexai_":
		resp, err = d.vertexAI.Resolve(ctx, in)

	case resolver == "embeddings":
		return nil, status.Error(codes.Unimplemented, "embeddings resolver not yet implemented")

	default:
		return nil, status.Errorf(codes.InvalidArgument, "unknown specialized resolver: %s", resolver)
	}

	if err != nil {
		return nil, err
	}

	return StandardizeResponse(in, resp), nil
}
