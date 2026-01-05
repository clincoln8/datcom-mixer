#!/bin/bash
set -e

# Base URL
URL="http://localhost:8081/v2/resolve"

echo "========================================================"
echo "Running Manual Curl Tests for Expanded Resolve API"
echo "Server must be running on $URL"
echo "========================================================"
echo ""

run_test() {
    local name="$1"
    local data="$2"
    
    echo "--------------------------------------------------------"
    echo "TEST: $name"
    echo "Payload: $data"
    echo "Response:"
    curl -s -X POST "$URL" \
      -H "Content-Type: application/json" \
      -d "$data" | jq . || echo " (jq not installed, raw output above)"
    echo ""
}

run_test "Backwards Compatibility (Legacy Mode)" '{
    "nodes": ["Santa Clara", "Mountain View"],
    "property": "<-description->dcid"
}'

run_test "Basic Resolution (New Mode)" '{
    "nodes": ["population without health insurance"],
    "specialized_resolver": "vertexai:nl_statvars"
}'

run_test "Limit (Limit=1)" '{
    "nodes": ["population without health insurance"],
    "specialized_resolver": "vertexai:nl_statvars",
    "limit": 1
}'

run_test "Filters (typeOf: StatisticalVariable)" '{
    "nodes": ["population without health insurance"],
    "specialized_resolver": "vertexai:nl_statvars",
    "filters": {
      "typeOf": "StatisticalVariable"
    }
}'

run_test "Advanced Enrichment (Returned Properties: memberOf, ^relevantVariable)" '{
    "nodes": ["population without health insurance"],
    "specialized_resolver": "vertexai:nl_statvars",
    "limit": 2,
    "returned_properties": ["memberOf", "^relevantVariable"]
}'

run_test "Typed Inverse Selection (^member:StatVarGroup)" '{
    "nodes": ["population without health insurance"],
    "specialized_resolver": "vertexai:nl_statvars",
    "returned_properties": ["^member:StatVarGroup"]
}'

echo "--------------------------------------------------------"
echo "TEST: Conflict Detection (Error Handling)"
echo "Payload: { ... conflicting params ... }"
echo "Response (Expect 400 Bad Request):"
curl -s -X POST "$URL" \
  -H "Content-Type: application/json" \
  -d '{
    "nodes": ["foo"],
    "property": "<-description->dcid",
    "specialized_resolver": "vertexai:nl_statvars"
  }' | jq . || echo " (jq not installed)"
echo ""

run_test "Place Resolution" '{
    "nodes": ["Santa Clara", "Mountain View"],
    "specialized_resolver": "place"
}'

run_test "Property Dereferencing (*constraintProperties)" '{
    "nodes": ["females"],
    "specialized_resolver": "vertexai:nl_statvars",
    "returned_properties": ["*constraintProperties"]
}'

run_test "In-Arc Filtering (^member: ...)" '{
    "nodes": ["population without health insurance"],
    "specialized_resolver": "vertexai:nl_statvars",
    "filters": {
       "^member": "dc/svpg/dc/topic/NoHealthInsuranceByGender_UninsuredByGender"
    }
}'

echo "========================================================"
echo "All tests executed."
