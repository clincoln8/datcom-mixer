package resolve

import (
	pbv2 "github.com/datacommonsorg/mixer/internal/proto/v2"
)

// FilterCandidates applies filtering logic to a list of candidates.
// It returns a filtered list of candidates.
func FilterCandidates(candidates []*pbv2.ResolveResponse_Entity_Candidate, filters map[string][]string) []*pbv2.ResolveResponse_Entity_Candidate {
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
// For a single filter key, if any of the provided values match, it returns true (OR logic).
func matchFilters(candidate *pbv2.ResolveResponse_Entity_Candidate, filters map[string][]string) bool {
	for key, wantValues := range filters {
		matched := false

		// 1. Handle typeOf specially (check DominantType + list)
		if key == "typeOf" {
			// Check if candidate has ANY of the wanted types
			for _, wantValue := range wantValues {
				if candidate.DominantType == wantValue {
					matched = true
					break
				}
				for _, t := range candidate.TypeOf {
					if t == wantValue {
						matched = true
						break
					}
				}
				if matched {
					break
				}
				// Fallback: Check candidate.Properties["typeOf"]
				if candidate.Properties != nil {
					if fields := candidate.Properties.Fields; fields != nil {
						if listVal, ok := fields["typeOf"]; ok {
							for _, v := range listVal.GetListValue().GetValues() {
								if v.GetStringValue() == wantValue {
									matched = true
									break
								}
							}
						}
					}
				}
				if matched {
					break
				}
			}
		} else {
			// 2. Arbitrary Property Filter
			if candidate.Properties != nil && candidate.Properties.Fields != nil {
				if val, ok := candidate.Properties.Fields[key]; ok {
					if listVal := val.GetListValue(); listVal != nil {
						// Check if ANY candidate value matches ANY wantValue
						for _, v := range listVal.Values {
							var candidateVal string
							if s := v.GetStringValue(); s != "" {
								candidateVal = s
							} else if s := v.GetStructValue(); s != nil {
								if dcidVal := s.Fields["dcid"]; dcidVal != nil {
									candidateVal = dcidVal.GetStringValue()
								}
							}

							if candidateVal != "" {
								for _, wantValue := range wantValues {
									if candidateVal == wantValue {
										matched = true
										break
									}
								}
							}
							if matched {
								break
							}
						}
					}
				}
			}
		}

		if !matched {
			return false
		}
	}
	return true
}
