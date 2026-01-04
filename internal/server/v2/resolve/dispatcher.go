package resolve

import (
	"context"
	"log"
	"sync"

	pbv2 "github.com/datacommonsorg/mixer/internal/proto/v2"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	dispatcherInstance *Dispatcher
	dispatcherOnce     sync.Once
)

// Dispatcher routes resolution requests to the appropriate backend.
type Dispatcher struct {
	vertexAI *VertexAIResolver
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
	}
}

// Dispatch routes the request.
func (d *Dispatcher) Dispatch(ctx context.Context, in *pbv2.ResolveRequest) (*pbv2.ResolveResponse, error) {
	resolver := in.GetSpecializedResolver()

	switch {
	case resolver == "place":
		// This should technically handle the 'place' logic if we moved it here.
		// For now, if code calls this, it implies non-legacy path?
		// But in handler_core.go, we only call this if property == "".
		// Existing 'place' logic usually requires property?
		// If resolver == "place" and property == "", what should happen?
		// "Resolve 'mountain view' to Place DCID" (default behavior).
		// We can return Unimplemented for "place" here if we haven't ported the description resolution yet,
		// OR we can wrap the existing legacy place resolution.
		// For the POC, let's leave "place" as Unimplemented here or basic error,
		// focusing on vertexai.
		return nil, status.Error(codes.Unimplemented, "expanded 'place' resolution without property not yet implemented")

	case len(resolver) > 9 && resolver[:9] == "vertexai_":
		return d.vertexAI.Resolve(ctx, in)

	case resolver == "embeddings":
		return nil, status.Error(codes.Unimplemented, "embeddings resolver not yet implemented")

	default:
		return nil, status.Errorf(codes.InvalidArgument, "unknown specialized resolver: %s", resolver)
	}
}
