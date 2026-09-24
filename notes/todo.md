# Development plan

These are the next items to implement from `notes/backlog.md`, listed in recommended implementation order.

Each numbered implementation step is intended to be a focused, independently passing commit. Run `go test ./...`, `go vet ./...`, and `staticcheck ./...` for every commit.

## 1. Decouple processing options from CLI config

The CLI's `config` value contains flag-parsing and presentation concerns, while `imageProcessor` needs only the incoming root, output root, and directory-date choice. Give the processor an explicit, focused options value so quiet-mode handling remains in `main` and future CLI-only fields do not become workflow dependencies.

**Packages involved:**

- `cmd/webimage`: define processor-specific options, translate CLI config at the composition point, and update processor tests.

**Implementation commits:**

1. **Introduce focused processor options.** Add a package-private `processorOptions` type containing only the incoming directory, output directory, and directory-date source; replace `imageProcessor`'s `config` field with those options and update its call sites. Construct the options explicitly in `main` after choosing the progress reporter, and update test fixtures to build processor options directly. Preserve all flag behavior and processing behavior. No new behavioral tests are needed because the existing config tests cover parsing and the existing processor tests cover each consumed option.

## 2. Write variants deterministically

Manifest output should not inherit the order in which variant generation happened. Canonicalize variant arrays immediately before JSON conversion without mutating caller-owned `Manifest` slices. Keep the existing JSON object schema; `encoding/json` already emits string map keys in lexical order, so the package only needs to make each format's files deterministic.

**Packages involved:**

- `internal/manifest`: order variant files during conversion to the persisted JSON representation and cover the serialized behavior.
- `internal/image`: supplies the format and variant values used for ordering; no API change is expected.

**Implementation commits:**

1. **Sort variants before manifest encoding.** In `manifestToJSON`, clone each format's variant slice and sort it by width, using the relative path as a deterministic tie-breaker, before constructing `variantJSON` values. Preserve format-key spelling, manifest schema, and the in-memory order supplied by the caller. Extend the existing `WriteFile` table with a deliberately unsorted, mixed-format input and assert after decoding that each format's files are in canonical order. Do not compare the complete raw JSON solely to test map-key or indentation details, and do not add separate cases for every possible input permutation.
