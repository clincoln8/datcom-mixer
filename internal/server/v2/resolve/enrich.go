package resolve

import (
	"context"
	"strings"

	pbv2 "github.com/datacommonsorg/mixer/internal/proto/v2"
	"github.com/datacommonsorg/mixer/internal/server/resource"
	v2pv "github.com/datacommonsorg/mixer/internal/server/v2/propertyvalues"
	"github.com/datacommonsorg/mixer/internal/store"
	"google.golang.org/protobuf/types/known/structpb"
)

// EnrichResponse fetches and populates returned_properties for candidates.
func EnrichResponse(
	ctx context.Context,
	store *store.Store,
	metadata *resource.Metadata,
	resp *pbv2.ResolveResponse,
	properties []string,
) error {
	// 1. Collect all candidate DCIDs
	var dcids []string
	dcidSet := make(map[string]struct{})

	// If store is nil (e.g. tests), skip enrichment.
	if store == nil {
		return nil
	}

	if resp != nil {
		for _, entity := range resp.Entities {
			for _, candidate := range entity.Candidates {
				if candidate.Dcid != "" {
					if _, ok := dcidSet[candidate.Dcid]; !ok {
						dcidSet[candidate.Dcid] = struct{}{}
						dcids = append(dcids, candidate.Dcid)
					}
				}
			}
		}
	}

	if len(dcids) == 0 {
		return nil
	}

	// Prepare properties to fetch
	// typeOf is ALWAYS fetched.
	// name, dcid, dominantType are core and ignored if requested.
	// Splitting properties into directions
	var outProps []string
	var inProps []string
	// Map to handle type filtering for IN properties: prop -> requiredType
	inTypeFilters := make(map[string]string)

	ignoredProps := map[string]struct{}{
		"name":         {},
		"dcid":         {},
		"dominantType": {},
	}

	derefProps := make(map[string]struct{})

	// Track fetched properties to avoid re-fetching in Phase 2
	fetchedProps := make(map[string]struct{})

	for _, p := range properties {
		// Handle dereferencing prefix '*'
		cleanedProp := p
		if strings.HasPrefix(p, "*") {
			cleanedProp = strings.TrimPrefix(p, "*")
			derefProps[cleanedProp] = struct{}{} // Keep track of base props to dereference
		}

		// Handle Inverse prefix '^'
		if strings.HasPrefix(cleanedProp, "^") {
			base := strings.TrimPrefix(cleanedProp, "^")
			// Handle Type filter :Type
			if parts := strings.Split(base, ":"); len(parts) > 1 {
				base = parts[0]
				inTypeFilters[base] = parts[1]
			}

			if _, ignored := ignoredProps[base]; !ignored {
				inProps = append(inProps, base)
				fetchedProps[base] = struct{}{}
			}
		} else {
			if _, ignored := ignoredProps[cleanedProp]; !ignored {
				outProps = append(outProps, cleanedProp)
				fetchedProps[cleanedProp] = struct{}{}
			}
		}
	}

	// Always fetch typeOf and name in OUT direction
	outProps = append(outProps, "typeOf")
	outProps = append(outProps, "name")
	fetchedProps["typeOf"] = struct{}{}
	fetchedProps["name"] = struct{}{}

	if len(outProps) == 0 && len(inProps) == 0 {
		return nil
	}

	// Helper to populate candidate properties
	populateFn := func(nodeResp *pbv2.NodeResponse, directionPrefix string) {
		for _, entity := range resp.Entities {
			for _, candidate := range entity.Candidates {
				if candidate.Dcid == "" {
					continue
				}

				if candidate.Properties == nil {
					candidate.Properties = &structpb.Struct{
						Fields: make(map[string]*structpb.Value),
					}
				}

				if lg, ok := nodeResp.Data[candidate.Dcid]; ok {
					for prop, nodes := range lg.Arcs {
						// Apply keys with direction prefix (e.g. ^containedInPlace)
						// Exception: typeOf is always typeOf (no prefix)
						responseKey := directionPrefix + prop
						if prop == "typeOf" {
							responseKey = "typeOf"
						}

						// Special handling for name
						if prop == "name" {
							if candidate.Name == "" {
								for _, n := range nodes.Nodes {
									if n.Value != "" {
										candidate.Name = n.Value
										break
									}
								}
							}
							continue
						}

						// Special handling for typeOf (Always Strings)
						if prop == "typeOf" {
							var typeValues []string
							for _, n := range nodes.Nodes {
								if n.Dcid != "" {
									typeValues = append(typeValues, n.Dcid)
								} else if n.Value != "" {
									typeValues = append(typeValues, n.Value)
								} else if n.Name != "" {
									typeValues = append(typeValues, n.Name)
								}
							}
							if len(typeValues) > 0 {
								candidate.TypeOf = typeValues
								if len(typeValues) == 1 {
									candidate.DominantType = ""
								}
							}
							continue
						}

						// Polymorphic handling for other properties
						var values []interface{}
						for _, n := range nodes.Nodes {
							// Filter Logic
							if reqType, hasFilter := inTypeFilters[prop]; hasFilter && directionPrefix == "^" {
								// Basic check against n.Types if present
								if len(n.Types) > 0 {
									match := false
									for _, t := range n.Types {
										if t == reqType {
											match = true
											break
										}
									}
									if !match {
										continue
									}
								}
							}

							if n.Value != "" {
								values = append(values, n.Value)
							} else if n.Dcid != "" {
								// Node object
								nodeObj := map[string]interface{}{
									"dcid": n.Dcid,
								}
								if n.Name != "" {
									nodeObj["name"] = n.Name
								}
								if len(n.Types) > 0 {
									tIf := make([]interface{}, len(n.Types))
									for i, t := range n.Types {
										tIf[i] = t
									}
									nodeObj["typeOf"] = tIf
								}
								values = append(values, nodeObj)
							} else if n.Name != "" {
								values = append(values, n.Name)
							}
						}

						if len(values) > 0 {
							listValue, err := structpb.NewList(values)
							if err == nil {
								candidate.Properties.Fields[responseKey] = structpb.NewListValue(listValue)
							}
						}
					}
				}
			}
		}
	}

	// 2. Fetch OUT Properties
	if len(outProps) > 0 {
		nodeResp, err := v2pv.PropertyValues(
			ctx,
			store,
			metadata,
			dcids,
			outProps,
			"out",
			0,
			"",
		)
		if err != nil {
			return err
		}

		populateFn(nodeResp, "")
	}

	// 3. Fetch IN Properties
	if len(inProps) > 0 {
		nodeResp, err := v2pv.PropertyValues(
			ctx,
			store,
			metadata,
			dcids,
			inProps,
			"in",
			0,
			"", // token
		)
		if err != nil {
			return err
		}
		populateFn(nodeResp, "^")
	}

	// 3. Handle Dereferencing (Phase 2)
	if len(derefProps) > 0 {
		secondPassProps := map[string]struct{}{}

		// Collect values from the base properties to use as new properties to fetch
		for _, entity := range resp.Entities {
			for _, candidate := range entity.Candidates {
				if candidate.Properties == nil {
					continue
				}
				for baseProp := range derefProps {
					if val, ok := candidate.Properties.Fields[baseProp]; ok {
						// Extract values from list helper
						if listVal := val.GetListValue(); listVal != nil {
							for _, v := range listVal.Values {
								var propName string
								// Handle Polymorphic (String or Struct)
								if str := v.GetStringValue(); str != "" {
									propName = str
								} else if s := v.GetStructValue(); s != nil {
									if dcidVal := s.Fields["dcid"]; dcidVal != nil {
										propName = dcidVal.GetStringValue()
									}
								}

								if propName != "" {
									if _, ignored := ignoredProps[propName]; !ignored {
										// Avoid re-fetching things we already have
										if _, alreadyFetched := fetchedProps[propName]; !alreadyFetched {
											secondPassProps[propName] = struct{}{}
										}
									}
								}
							}
						}
					}
				}
			}
		}

		if len(secondPassProps) > 0 {
			var nextProps []string
			for p := range secondPassProps {
				nextProps = append(nextProps, p)
			}

			nodeResp2, err := v2pv.PropertyValues(
				ctx,
				store,
				metadata,
				dcids,
				nextProps,
				"out",
				0,
				"",
			)
			if err != nil {
				return err
			}
			populateFn(nodeResp2, "")
		}
	}

	return nil
}
