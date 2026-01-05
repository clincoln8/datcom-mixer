# Expanded Resolve API Manual Testing Guide

This guide provides step-by-step instructions for manually verifying the **Expanded Resolve API** (`/v2/resolve`).

## 1. Run mixer and esp

In separate terminals, run:

```bash
envoy -l warning --config-path esp/envoy-config.yaml
```

```bash
./run_server.sh -feature_flags_path=deploy/featureflags/expanded_resolve.yaml
```
---

## 2. Test Suite

Run these tests using either **curl (POST)** or your **Browser (GET)**.

### Backwards Compatibility (Legacy Mode)
**Goal**: Ensure existing clients strictly using `property` parameters are NOT broken.

**POST (Curl):**
```bash
curl -X POST http://localhost:8081/v2/resolve \
  -H "Content-Type: application/json" \
  -d '{
    "nodes": ["Santa Clara", "Mountain View"],
    "property": "<-description->dcid"
  }'
```

**GET (Browser):**
```
http://localhost:8081/v2/resolve?nodes=Santa%20Clara&nodes=Mountain%20View&property=<-description->dcid
```

**Expectation**:
-   Status: `200 OK`
-   Response contains legacy structure (e.g., `geoId/0669000` for Santa Clara).
-   **No** new fields like `status` or `properties` inside candidates.

### Basic Resolution (New Mode)
**Goal**: Verify routing to the new Dispatcher and Vertex AI integration.

**POST (Curl):**
```bash
curl -X POST http://localhost:8081/v2/resolve \
  -H "Content-Type: application/json" \
  -d '{
    "nodes": ["population without health insurance"],
    "specialized_resolver": "vertexai:nl_statvars"
  }'
```

**GET (Browser):**
```
http://localhost:8081/v2/resolve?nodes=population%20without%20health%20insurance&specialized_resolver=vertexai:nl_statvars
```

**Expectation**:
-   Status: `200 OK`
-   Candidates include StatVars (e.g., `Count_Person_NoHealthInsurance`) with `name` populated.
-   New structure: `entities[].status` is present.

### Limit
**Goal**: Verify `limit` parameter effectively restricts the number of candidates.

**POST (Curl):**
```bash
curl -X POST http://localhost:8081/v2/resolve \
  -H "Content-Type: application/json" \
  -d '{
    "nodes": ["population without health insurance"],
    "specialized_resolver": "vertexai:nl_statvars",
    "limit": 1
  }'
```

**GET (Browser):**
```
http://localhost:8081/v2/resolve?nodes=population%20without%20health%20insurance&specialized_resolver=vertexai:nl_statvars&limit=1
```

**Expectation**:
-   Status: `200 OK`
-   Exactly **1** candidate returned.

### Filters
**Goal**: Verify `filters` can prune candidates (e.g. restrict to `StatisticalVariable`).

**POST (Curl):**
```bash
curl -X POST http://localhost:8081/v2/resolve \
  -H "Content-Type: application/json" \
  -d '{
    "nodes": ["population without health insurance"],
    "specialized_resolver": "vertexai:nl_statvars",
    "filters": {
      "typeOf": "StatisticalVariable"
    }
  }'
```

**Expectation**:
-   Status: `200 OK`
-   All returned candidates must have `typeOf` / `dominantType` equal to `StatisticalVariable`.

### Advanced Enrichment (Returned Properties)
**Goal**: Test standard properties, dereferencing (`*`), and inverse properties (`^`).

**POST (Curl):**
```bash
curl -X POST http://localhost:8081/v2/resolve \
  -H "Content-Type: application/json" \
  -d '{
    "nodes": ["population without health insurance"],
    "specialized_resolver": "vertexai:nl_statvars",
    "limit": 2,
    "returned_properties": ["memberOf", "^relevantVariable"]
  }'
```

**GET (Browser):**
```
http://localhost:8081/v2/resolve?nodes=population%20without%20health%20insurance&specialized_resolver=vertexai:nl_statvars&limit=2&returned_properties=memberOf&returned_properties=^relevantVariable
```

**Expectation**:
-   Status: `200 OK`
-   Candidate `properties` map contains `memberOf` and `^relevantVariable`.

### Typed Inverse Selection
**Goal**: Test selecting specific types of incoming arcs.

**POST (Curl):**
```bash
curl -X POST http://localhost:8081/v2/resolve \
  -H "Content-Type: application/json" \
  -d '{
    "nodes": ["population without health insurance"],
    "specialized_resolver": "vertexai:nl_statvars",
    "returned_properties": ["^member:StatVarGroup"]
  }'
```

**GET (Browser):**
```
http://localhost:8081/v2/resolve?nodes=population%20without%20health%20insurance&specialized_resolver=vertexai:nl_statvars&returned_properties=^member:StatVarGroup
```

**Expectation**:
-   Status: `200 OK`
-   Response property is `^member` containing only `StatVarGroup` nodes.

### Conflict Detection (Error Handling)
**Goal**: Verify the API rejects invalid combinations.

**POST (Curl):**
```bash
curl -X POST http://localhost:8081/v2/resolve \
  -H "Content-Type: application/json" \
  -d '{
    "nodes": ["foo"],
    "property": "<-description->dcid",
    "specialized_resolver": "vertexai:nl_statvars"
  }'
```

**GET (Browser):**
```
http://localhost:8081/v2/resolve?nodes=foo&property=<-description->dcid&specialized_resolver=vertexai:nl_statvars
```

**Expectation**:
-   Status: `400 Bad Request`
-   Error message contains "conflicting parameters".

### Place Resolution (New Mode)
**Goal**: Verify that the explicit Place Resolver works with the new API structure.

**POST (Curl):**
```bash
curl -X POST http://localhost:8081/v2/resolve \
  -H "Content-Type: application/json" \
  -d '{
    "nodes": ["Santa Clara", "Mountain View"],
    "specialized_resolver": "place"
  }'
```

**GET (Browser):**
```
http://localhost:8081/v2/resolve?nodes=Santa%20Clara&nodes=Mountain%20View&specialized_resolver=place
```

**Expectation**:
-   Status: `200 OK`
-   Candidates should be places (e.g., `geoId/06085` or similar).

### Property Dereferencing
**Goal**: Verify that requesting `*property` returns full node details (dereferenced objects) instead of just DCIDs.

**POST (Curl):**
```bash
curl -X POST http://localhost:8081/v2/resolve \
  -H "Content-Type: application/json" \
  -d '{
    "nodes": ["females"],
    "specialized_resolver": "vertexai:nl_statvars",
    "returned_properties": ["*constraintProperties"]
  }'
```

**GET (Browser):**
```
http://localhost:8081/v2/resolve?nodes=females&specialized_resolver=vertexai:nl_statvars&returned_properties=*constraintProperties
```

**Expectation**:
-   Status: `200 OK`
-   Candidate `properties` should contain `constraintProperties`.
-   The values in `constraintProperties` should be **objects** (e.g. `{"dcid": "gender", "name": "..."}`) rather than simple strings.

### In-Arc Filtering
**Goal**: Verify filtering candidates based on incoming edges (e.g. only return variables that are members of a specific Group).

**POST (Curl):**
```bash
curl -X POST http://localhost:8081/v2/resolve \
  -H "Content-Type: application/json" \
  -d '{
    "nodes": ["population without health insurance"],
    "specialized_resolver": "vertexai:nl_statvars",
    "filters": {
      "^member": "dc/svpg/dc/topic/NoHealthInsuranceByGender_UninsuredByGender"
    }
  }'
```

**Expectation**:
-   Status: `200 OK`
-   Returned candidates (e.g. `Count_Person_NoHealthInsurance_Female_NoHealthInsurance`) must be members of the specified SVPG.
-   Returned candidates (e.g. `Count_Person_NoHealthInsurance_Female_NoHealthInsurance`) must be members of the specified SVPG.
-   Candidates *not* in that group should be filtered out.

### Hybrid Support (Legacy + Modern)
**Goal**: Verify that legacy properties can be combined with modern parameters (Limit, Filters, Enrichment).

**POST (Curl):**
```bash
curl -X POST http://localhost:8081/v2/resolve \
  -H "Content-Type: application/json" \
  -d '{
    "nodes": ["Santa Clara"],
    "property": "<-description->dcid",
    "limit": 2,
    "returned_properties": ["brockhausEncylcopediaOnlineId"]
  }' | jq
```

**Expectation**:
-   Status: `200 OK`
-   Response contains exactly **2** candidates (Limit works).
-   Candidates have `brockhausEncylcopediaOnlineId` populated (Enrichment works).

### New Resolver, old logic
**Goal**: Verify Place resolver with legacy-style property return.

**POST (Curl):**
```bash
curl -X POST http://localhost:8081/v2/resolve \
  -H "Content-Type: application/json" \
  -d '{
    "nodes": ["Santa Clara"],
    "specialized_resolver": "place",
    "returned_properties": ["containedInPlace"]
  }' | jq
```

**Expectation**:
-   Status: `200 OK`
-   Candidates have `containedInPlace` populated.

### Legacy ID Resolution (Hybrid Check)
**Goal**: Verify legacy ID-to-ID resolution works (optionally with modern params).

**POST (Curl):**
```bash
curl -X POST http://localhost:8081/v2/resolve \
  -H "Content-Type: application/json" \
  -d '{
    "nodes": ["Q30"],
    "property": "<-wikidataId->dcid"
  }' | jq
```

**Expectation**:
-   Status: `200 OK`
-   Resolves `Q30` to `country/USA`.

### Coordinate Resolution with Filter (Hybrid)
**Goal**: Verify coordinate resolution allows filtering results (e.g. only return candidates contained in a specific city).

**POST (Curl):**
```bash
curl -X POST http://localhost:8081/v2/resolve \
  -H "Content-Type: application/json" \
  -d '{
    "nodes": ["37.42#-122.08"],
    "property": "<-geoCoordinate->dcid",
    "filters": {
      "containedInPlace": "geoId/0608592830"
    }
  }' | jq
```

**Expectation**:
-   Status: `200 OK`
-   Candidates should only include those contained in Mountain View Census County Division (`geoId/0608592830`).


