# TODO

## Review Hybrid Flow User Experience

**Task:** Review the user experience for the "hybrid flow" of the `v2/resolve` endpoint and define its intended behavior.

**Current Inconsistency Notes (Design vs. Implementation):**

*   **Design Expectation**: The original design suggested that "hybrid mode" would allow users to leverage the legacy `property` parameter for initial resolution, with the results then being processed by modern features like `filters`, `limit`, and `returnedProperties`. This implied a pipeline where legacy resolution fed into the new logic tiers.
*   **Current Implementation (`internal/server/handler_core.go`)**: The current implementation behaves as a strict "hard fork". If the `property` parameter is present in the request (and there are no conflicts that trigger an error), the request is *exclusively* routed down the legacy resolution path. Any modern parameters (`filters`, `limit`, `specializedResolver`, `returnedProperties`) provided alongside `property` are **ignored** because the new logic tiers are never executed.

**Action Required:**
1.  Determine the desired user experience for requests containing the `property` field along with new parameters.
2.  Align the documentation (`dd.md`, `expanded_resolve.md`) with the chosen user experience.
3.  If the desired experience is a pipeline (legacy resolution feeding into new logic tiers), implement the necessary changes to `handler_core.go` and the `Dispatcher` to enable this behavior.
