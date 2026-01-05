package resolve

import (
	pbv2 "github.com/datacommonsorg/mixer/internal/proto/v2"
)

// StandardizeResponse processes the raw response from a resolver.
// It applies filtering, limiting, and ensures structural consistency.
func StandardizeResponse(req *pbv2.ResolveRequest, resp *pbv2.ResolveResponse) *pbv2.ResolveResponse {
	if resp == nil {
		return &pbv2.ResolveResponse{}
	}

	// Convert structpb.Struct to map[string]string
	filters := make(map[string]string)
	if f := req.GetFilters(); f != nil {
		for k, v := range f.Fields {
			if s := v.GetStringValue(); s != "" {
				filters[k] = s
			}
		}
	}
	for _, entity := range resp.Entities {
		// 1. Filter
		if len(filters) > 0 {
			entity.Candidates = FilterCandidates(entity.Candidates, filters)
		}

		// 2. Limit
		limit := int(req.GetLimit())
		if limit >= 0 && len(entity.Candidates) > limit {
			entity.Candidates = entity.Candidates[:limit]
		}
	}

	return resp
}
