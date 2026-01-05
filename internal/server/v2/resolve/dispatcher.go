package resolve

import (
	"context"
	"log"
	"strings"
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
	// We filter based on the properties fetched in Phase 1 (if any).
	// This reduces the candidate list significantly before the expensive full enrichment.
	resp = StandardizeResponse(in, resp)

	// 3. Post-fetch requested properties for the remaining candidates
	// We run this to populate 'returned_properties' and ensure names/types are present for the final result.
	err = EnrichResponse(ctx, store, metadata, resp, in.GetReturnedProperties())
	if err != nil {
		return nil, err
	}

	// Final Cleanup: Remove empty properties maps to avoid "{}" in JSON
	if resp != nil {
		baseAllowed := make(map[string]bool)
		var derefBases []string

		for _, p := range in.GetReturnedProperties() {
			cleanP := p
			isDeref := false
			// Handle Dereferencing prefix '*'
			if len(cleanP) > 1 && cleanP[0] == '*' {
				cleanP = cleanP[1:]
				isDeref = true
			}

			// Allow exact match of cleaned property
			baseAllowed[cleanP] = true

			// Handle Type filter suffix ':' (e.g. "^member:StatVarGroup" -> "^member")
			if idx := strings.Index(cleanP, ":"); idx != -1 {
				base := cleanP[:idx]
				baseAllowed[base] = true
				if isDeref {
					derefBases = append(derefBases, base)
				}
			} else {
				if isDeref {
					derefBases = append(derefBases, cleanP)
				}
			}
		}

		for _, entity := range resp.Entities {
			for _, candidate := range entity.Candidates {
				// Prune unrequested properties
				if candidate.Properties != nil {
					// 1. Identify dynamic properties from dereferencing
					dynamicAllowed := make(map[string]bool)
					for _, base := range derefBases {
						if val, ok := candidate.Properties.Fields[base]; ok {
							if listVal := val.GetListValue(); listVal != nil {
								for _, v := range listVal.Values {
									if s := v.GetStringValue(); s != "" {
										dynamicAllowed[s] = true
									} else if st := v.GetStructValue(); st != nil {
										if d := st.Fields["dcid"]; d != nil {
											dynamicAllowed[d.GetStringValue()] = true
										}
									}
								}
							}
						}
					}

					// 2. Delete if not allowed
					for key := range candidate.Properties.Fields {
						keep := baseAllowed[key] || dynamicAllowed[key]
						// Allow typeOf if it's there? Usually explicitly requested or auto-fetched?
						// "typeOf" is automatically fetched by EnrichResponse. 
						// If user didn't ask for it, should we keep it?
						// Most legacy clients expect typeOf?
						// User said: "just beacuase a prop is listed in filtering does not mean it should be retured"
						// But typeOf is special.
						// Let's assume if it wasn't requested, and not dereferenced, it goes.
						// UNLESS it's required for the candidate struct itself (candidate.TypeOf slice).
						// The Properties["typeOf"] is the legacy one.
						if !keep {
							delete(candidate.Properties.Fields, key)
						}
					}
				}

				if candidate.Properties != nil && len(candidate.Properties.Fields) == 0 {
					candidate.Properties = nil
				}
				// User Request: Never return DominantType for resolver != place
				if resolver != "place" {
					candidate.DominantType = ""
				}
				// User Request: Only return name if it differs from DCID
				if candidate.Name == candidate.Dcid {
					candidate.Name = ""
				}
			}
		}
	}

	return resp, nil
}
