### **Request Format**

| Parameter | Type | Required | Description |
| :---- | :---- | :---- | :---- |
| **nodes** | `repeated string` | **Yes** | A list of identifiers or search terms to resolve to Knowledge Graph nodes. |
| **specializedResolver** | `string` | No | Specifies the resolution engine. If empty, defaults to "embeddings" (which is not yet implemented). See Translation Tier section for valid values. |
| **filters** | `object` | No | Match constraints (e.g., `{"typeOf": "City"}`). The value for a key can be a single string or an array of strings for OR logic. Supports **Exact Match** logic only. Use the `^` prefix for incoming properties. If the object is a node, the value must be its DCID. |
| **limit** | `integer` | No | Default: 10. The maximum number of candidates to return per input node. |
| **returnedProperties** | `repeated string` | No | Specifies **additional** Knowledge Graph properties to return for each candidate. Core identity fields (name, dcid, typeOf) are always included. Use `*` prefix for property dereferencing (e.g., `*constraintProperties`). |
| **property** | `string` | No | **Deprecated.** Routes the request to the legacy "pass-through" implementation. |

### **Logic Tiers**

#### **0\. Backwards Compatibility Tier (Routing & Conflicts)**

To ensure zero regressions while encouraging adoption of the modern interface, the API uses a strict routing fork.

*   **Conflict Resolution (Strict Error)**: The API returns a **400 Bad Request** if `property` is provided alongside a `specializedResolver` other than `place`.
*   **Hybrid Support**: If `property` is present with modern `filters`, `limit`, or `returnedProperties`, the request is routed to the legacy engine. The modern parameters are ignored, as the legacy engine does not support them.

#### **1. Translation Tier (specializedResolver)**

*   **place**: A wrapper for the legacy place resolution logic. Primarily resolves place names to DCIDs.
*   **embeddings**: The default resolver if the parameter is empty. Currently **unimplemented**.
*   **`vertexai:nl_statvars`**: Targeted lookup using Vertex AI for a curated subset of Statistical Variables (primarily for Natural Language queries).
*   **`vertexai:all_statvars`**: Targeted lookup using Vertex AI for all Statistical Variables.

#### **2\. Validation Tier (filters)**

Acts as a quality gate for all modern (non-property) requests. This tier handles triple directionality and boolean logic after candidates have been resolved.

*   **Triple Directionality**:
    *   **Outgoing (Default)**: `{"prop": "val"}` matches `[Node] -> prop -> val`.
    *   **Incoming (Inverse)**: `{"^prop": "val"}` matches `val -> prop -> [Node]`.
*   **Boolean Filter Logic**:
    *   **AND (Intersection of Properties)**: Specifying multiple unique keys in the `filters` object requires a candidate to satisfy **all** conditions.
        *   *Example*: `{"typeOf": "City", "^containedInPlace": "geoId/06"}` finds nodes that are a City **AND** are contained in California.
    *   **OR (Union of Values)**: Specifying a list of string values for a single key allows a candidate to match **any** of those values.
        *   *Example*: `{"typeOf": ["City", "Town"]}` finds nodes that are either a City **OR** a Town.
*   **Post-Resolution Filtering**: Performs exact-match KG checks for candidates returned by the resolver. The API pre-fetches the properties required by the filters before applying them.

#### **3\. Response Tier (returnedProperties)**

*   **Standardization**: Regardless of the backend used, all results are mapped to the standardized v2 response structure.
*   **Dominant Type Logic**: For candidates with multiple types, the system populates this field with the *first* type returned from the database. For non-`place` resolutions, this field is cleared from the final response before being returned.
*   **Property Serialization**: Since the Knowledge Graph does not explicitly define property cardinality, **all properties nested in the `properties` object are returned as Lists.**

### **Response Structure**

#### **Legacy Format (Internal Reference)**

[docs](https://docs.datacommons.org/api/rest/v2/resolve.html#response)

```json
{
  "entities": [
    {
      "node": "Georgia",
      "candidates": [
        {
          "dcid": "geoId/13",
          "dominantType": "State"
        },
        {
          "dcid": "country/GEO",
          "dominantType": "Country"
        }
      ]
    }
  ]
}
```

#### **Standardized v2 Format (Superset)**

```json
{
  "entities": [
    {
      "node": "Springfield",
      "status": "COMPLETED",
      "candidates": [
        {
          "dcid": "geoId/1770447",
          "name": "Springfield",
          "typeOf": ["City", "AdministrativeArea"],
          "properties": {
            "containedInPlace": [
              {"dcid": "geoId/17", "name": "Illinois", "typeOf": ["State"]},
              {"dcid": "geoId/17167", "name": "Sangamon County", "typeOf": ["County"]}
            ],
            "wikidataId": ["Q28515"]
          },
          "metadata": {
            "confidenceScore": 1.0
          }
        }
      ]
    },
    {
      "node": "InvalidNode_XYZ",
      "status": "FAILED",
      "error": "An error occurred during resolution."
    }
  ]
}
```

### **Out of Scope & Future Work**

#### **Future Work (Roadmap)**

*   **`searchProperties`**: A parameter to override or restrict the set of properties a resolver searches against.
*   **Pagination**: Implementing `nextToken` or `offset` parameters.
*   **Localization**: A `lang` parameter to support multi-lingual resolution and property fetching.
*   **Dot Notation**: Support for `returnedProperties` like `parentPlace.name`.
*   **Unified 'all' Backend:** Investigate and implement a unified service layer to handle all 'all' `specializedResolver` traffic.

#### **Out of Scope (Intentional Constraints)**

*   **Range Filters**: Complex queries (e.g., `population > 100000`) belong in a Search API.
*   **Nested Resolution**: Filtering by properties of a linked node is deferred.
*   **Same-Property AND Logic**: The API will not support a native syntax for ensuring a single property contains multiple specific values (e.g., "Must be both type A and type B").

### **Example Flows**

#### **1\. Multi-Type Semantic Resolution (Topic vs. StatVar)**

**Input**: `"nodes": "health insurance", "specializedResolver": "all", "filters": {"typeOf": ["Topic", "StatisticalVariable"]}`

1.  **Resolver** finds semantic proximity for "health insurance".
2.  **Candidates Found**: `dc/topic/NoHealthInsurance` (Topic) and `Count_Person_NoHealthInsurance` (StatVar).
3.  **Validation**: Matches candidates against the OR list in `typeOf`.
4.  **Response**: Both candidates are returned with their respective types.

#### **2\. Complex Boolean Filter (AND + OR)**

**Input**: `"nodes": "Cambridge", "specializedResolver": "place", "filters": {"typeOf": "City", "^containedInPlace": ["country/USA", "country/GBR"]}`

1.  **Resolver** finds all places named "Cambridge".
2.  **Validation**: Filters for nodes that are of `typeOf` City **AND** are (`containedInPlace` `country/USA` **OR** `country/GBR`).
3.  **Result**: Returns Cambridge, MA and Cambridge, UK, but excludes Cambridge, Ontario (in Canada).

### **Open Questions for Iteration**

#### **Search & Filtering**

*   **Dot Notation**: Syntax and execution for one-level-deep lookups.
*   **Search Intent**: Weighting mechanism vs. hard filter for `searchProperties`.

#### **Response Structure & Metadata**

*   **Metadata Placement (resolver)**: Should the resolver used for an input term be included as metadata at the `entities` level?
*   **No-Match Behavior**: Should we support a `failFast` parameter to return a 404?
*   **Source Attribution (matchedProperty)**: Should we return which property actually triggered the match?

#### **Disambiguation & Vertex AI**

*   Should the `vertexai` resolver return a confidence/similarity score for each candidate by default?