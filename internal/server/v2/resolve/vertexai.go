package resolve

import (
	"context"
	"fmt"
	"strings"

	discoveryengine "cloud.google.com/go/discoveryengine/apiv1"
	discoveryenginepb "cloud.google.com/go/discoveryengine/apiv1/discoveryenginepb"
	pbv2 "github.com/datacommonsorg/mixer/internal/proto/v2"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// VertexAI defines the interface for calling Vertex AI.
// It is defined here to allow mocking in tests.
type VertexAI interface {
	// Predict sends a prediction request to Vertex AI.
	// This signature is a placeholder and should be adapted to the actual Google Cloud Vertex AI SDK.
	// For now, we assume it returns a list of candidate DCIDs per input node.
	Predict(ctx context.Context, appID string, nodes []string) (map[string][]string, error)
	Close() error
}

// VertexAIClient implements VertexAI using the Google Cloud Discovery Engine API.
type VertexAIClient struct {
	client    *discoveryengine.SearchClient
	projectID string
	location  string
}

// NewVertexAIClient creates a new VertexAIClient.
func NewVertexAIClient(ctx context.Context, projectID, location string) (*VertexAIClient, error) {
	client, err := discoveryengine.NewSearchClient(ctx)
	if err != nil {
		return nil, err
	}
	return &VertexAIClient{
		client:    client,
		projectID: projectID,
		location:  location,
	}, nil
}

// Close closes the underlying client.
func (c *VertexAIClient) Close() error {
	return c.client.Close()
}

// Predict sends a search request to the Vertex AI Agent Builder (Discovery Engine).
func (c *VertexAIClient) Predict(ctx context.Context, appID string, nodes []string) (map[string][]string, error) {
	// Serving Config: projects/{project}/locations/{location}/collections/default_collection/dataStores/{data_store}/servingConfigs/default_search
	// We assume appID corresponds to the Data Store ID.
	servingConfig := fmt.Sprintf("projects/%s/locations/%s/collections/default_collection/dataStores/%s/servingConfigs/default_search", c.projectID, c.location, appID)

	results := make(map[string][]string)

	for _, node := range nodes {
		req := &discoveryenginepb.SearchRequest{
			ServingConfig: servingConfig,
			Query:         node,
			PageSize:      10, // Default limit
		}

		it := c.client.Search(ctx, req)
		var candidates []string
		for {
			resp, err := it.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				// Log error but continue with other nodes? Or fail?
				// For now, let's fail to surface issues early.
				return nil, err
			}
			
			// Extract DCID. 
			// Assuming the document ID is the DCID, or it is in the struct data.
			// Let's check struct data first, then fallback to ID.
			// The content structure depends on how data was indexed.
			// Datcom often indexes with 'dcid' field in struct_data.
			dcid := resp.Document.GetId()
			if structData := resp.Document.GetStructData(); structData != nil {
				if fields := structData.Fields; fields != nil {
					if v, ok := fields["dcid"]; ok {
						dcid = v.GetStringValue()
					}
				}
			}
			if dcid != "" {
				candidates = append(candidates, dcid)
			}
		}
		results[node] = candidates
	}

	return results, nil
}

// VertexAIResolver implements the resolution logic using Vertex AI.
type VertexAIResolver struct {
	client VertexAI
}

// NewVertexAIResolver creates a new VertexAIResolver.
func NewVertexAIResolver(client VertexAI) *VertexAIResolver {
	return &VertexAIResolver{client: client}
}

// Resolve processes the resolution request using Vertex AI.
func (r *VertexAIResolver) Resolve(ctx context.Context, in *pbv2.ResolveRequest) (*pbv2.ResolveResponse, error) {
	specializedResolver := in.GetSpecializedResolver()
	if !strings.HasPrefix(specializedResolver, "vertexai_") {
		return nil, status.Errorf(codes.InvalidArgument, "invalid specialized resolver for VertexAI: %s", specializedResolver)
	}

	appID := strings.TrimPrefix(specializedResolver, "vertexai_")
	
	// Hardcoded mapping for specializedResolver to engineId
	// Add new mappings here as needed.
	var dataStoreID string
	switch appID {
	case "embedding-statvars":
		dataStoreID = "nl-statvar-search-staging_1753469464090"
	case "all-statvars":
		// TODO: Add engine ID for all-statvars when available, or reuse same one if applicable
		return nil, status.Errorf(codes.Unimplemented, "unsupported Vertex AI app: %s", appID) // Placeholder
	default:
		return nil, status.Errorf(codes.Unimplemented, "unsupported Vertex AI app: %s", appID)
	}

	if r.client == nil {
		return nil, status.Error(codes.FailedPrecondition, "Vertex AI client not initialized")
	}

	// Call the client
	// Note: basic list of nodes from the request
	// Assuming 'nodes' in request are the descriptions/names to resolve.
	results, err := r.client.Predict(ctx, dataStoreID, in.GetNodes())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Vertex AI prediction failed: %v", err)
	}

	// Construct Response
	// We need to map the results back to the proto format.
	// ResolveResponse has 'entities' which is a list of Entity
	resp := &pbv2.ResolveResponse{
		Entities: []*pbv2.ResolveResponse_Entity{},
	}

	for node, candidates := range results {
		entity := &pbv2.ResolveResponse_Entity{
			Node:       node,
			Candidates: []*pbv2.ResolveResponse_Entity_Candidate{},
		}
		for _, dcid := range candidates {
			entity.Candidates = append(entity.Candidates, &pbv2.ResolveResponse_Entity_Candidate{
				Dcid: dcid,
				// DominantType could be inferred or returned by the model?
				// For 'embedding-statvars', we know they are StatisticalVariable
				DominantType: "StatisticalVariable",
			})
		}
		resp.Entities = append(resp.Entities, entity)
	}

	return resp, nil
}
