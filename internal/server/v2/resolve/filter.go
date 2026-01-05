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
// Supports 'typeOf' and arbitrary property matching.
func matchFilters(candidate *pbv2.ResolveResponse_Entity_Candidate, filters map[string]string) bool {
	for key, wantValue := range filters {
		// Handle 'typeOf' filter
		// We still check DominantType for typeOf as a fast path or primary check,
		// but ideally we should check the typeOf list in properties if available?
		// For backward compatibility and current logic, DominantType is reliable enough for now.
		if key == "typeOf" {
			// 1. Check DominantType
			if candidate.DominantType == wantValue {
				continue
			}
			// 2. Check Candidate.TypeOf (repeated field)
			match := false
			for _, t := range candidate.TypeOf {
				if t == wantValue {
					match = true
					break
				}
			}
			if match {
				continue
			}

			// 3. Fallback: Check candidate.Properties["typeOf"] (Legacy/Struct)
			if candidate.Properties != nil {
				if fields := candidate.Properties.Fields; fields != nil {
					if listVal, ok := fields["typeOf"]; ok {
						for _, v := range listVal.GetListValue().GetValues() {
							if v.GetStringValue() == wantValue {
								match = true
								break
							}
						}
					}
				}
			}
			if match {
				continue
			}

			return false
		}

		// Arbitrary Property Filter
		if candidate.Properties == nil || candidate.Properties.Fields == nil {
			return false
		}

		val, ok := candidate.Properties.Fields[key]
		if !ok {
			return false
		}

		// Check if any value in the list matches `wantValue`
		match := false
		if listVal := val.GetListValue(); listVal != nil {
			for _, v := range listVal.Values {
				// 1. Literal Match
				if s := v.GetStringValue(); s != "" {
					if s == wantValue {
						match = true
						break
					}
				}
				// 2. Node DCID Match
				if s := v.GetStructValue(); s != nil {
					if dcidVal := s.Fields["dcid"]; dcidVal != nil {
						if dcidVal.GetStringValue() == wantValue {
							match = true
							break
						}
					}
				}
			}
		}

		if !match {
			return false
		}
	}
	return true
}
