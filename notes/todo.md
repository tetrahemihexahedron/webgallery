# Development plan

These are the next items to implement from `notes/backlog.md`, listed in recommended implementation order.

Make small commits. Each numbered implementation step should be at least one independently passing commit. Run `go test ./...`, `go vet ./...`, and `staticcheck ./...` for every commit.

## 1. Write variants deterministically

Manifest output should not inherit the order in which variant generation happened. Canonicalize variant arrays immediately before JSON conversion without mutating caller-owned `Manifest` slices. Keep the existing JSON object schema; `encoding/json` already emits string map keys in lexical order, so the package only needs to make each format's files deterministic.

**Packages involved:**

- `internal/manifest`: order variant files during conversion to the persisted JSON representation and cover the serialized behavior.
- `internal/image`: supplies the format and variant values used for ordering; no API change is expected.

**Implementation commits:**

1. **Sort variants before manifest encoding.** In `manifestToJSON`, clone each format's variant slice and sort it by width, using the relative path as a deterministic tie-breaker, before constructing `variantJSON` values. Preserve format-key spelling, manifest schema, and the in-memory order supplied by the caller. Extend the existing `WriteFile` table with a deliberately unsorted, mixed-format input and assert after decoding that each format's files are in canonical order. Do not compare the complete raw JSON solely to test map-key or indentation details, and do not add separate cases for every possible input permutation.
