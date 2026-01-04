package resolve

import (
	pbv2 "github.com/datacommonsorg/mixer/internal/proto/v2"
)

// FilterCandidates applies filtering logic to a list of candidates.
// It returns a filtered list of candidates.
func FilterCandidates(candidates []*pbv2.ResolveResponse_Entity_Candidate, filters map[string]string) []*pbv2.ResolveResponse_Entity_Candidate {
	if len(filters) == 0 {
		return candidates
	}

	var filtered []*pbv2.ResolveResponse_Entity_Candidate
	for _, candidate := range candidates {
		if matchFilters(candidate, filters) {
			filtered = append(filtered, candidate)
		}
	}
	return filtered
}

// matchFilters checks if a candidate matches all filters.
// Currently supports 'typeOf' filter.
func matchFilters(candidate *pbv2.ResolveResponse_Entity_Candidate, filters map[string]string) bool {
	for key, wantValue := range filters {
		// Handle 'typeOf' filter
		if key == "typeOf" {
			// Candidate DominantType should match.
			// TODO: Handle inheritance or multiple types if candidate has full property map?
			// For now, check DominantType.
			if candidate.DominantType != wantValue {
				return false
			}
		}
		// Add other property filters here if Candidate has properties map populated
	}
	return true
}
