# Development plan

These are the next items to implement from `notes/todo.md`, listed in recommended implementation order.

Each numbered implementation step is intended to be a focused, independently passing commit. Run `go test ./...`, `go vet ./...`, and `staticcheck ./...` for every commit.

## 1. Minimize exported surfaces

Remove exports that are implementation details while preserving the small APIs actually used between packages. Do this package by package so each rename remains easy to review.

**Packages involved:**

- `cmd/gallery`: command-only configuration type and fields.
- `cmd/webimage`: command-only configuration, directory-date enum, constants, and fields.
- `internal/gallery`: sort-field validation helper.
- `internal/index`: index path helper.

**Implementation commits:**

1. **Unexport `cmd/gallery` configuration.** Rename `Config` to `config`, make its fields package-private, and update parsing, rendering, and tests. This package is a command and exposes no reusable Go API.
2. **Unexport `cmd/webimage` configuration.** Rename `Config` and its fields, updating `main`, processor construction, and tests. Keep this commit limited to configuration visibility rather than changing processor options.
3. **Unexport the webimage directory-date enum.** Rename `DirDate` and its constants to package-private names, choosing a name such as `dirDateSource` to avoid colliding with the existing `dirDate` function. Update validation, processor helpers, and tests without changing accepted flag values or error text.
4. **Unexport gallery's validity helper.** Change `SortField.IsValid` to an unexported helper or method. Keep `SortField`, its constants, and `ParseSortField` exported because `cmd/gallery` legitimately consumes them; retain validation inside `gallery.Render` for directly constructed options.
5. **Unexport the index path helper.** Rename `index.IndexPath` to `indexPath` because production callers use `ReadDir` and `Index.Update`, not the filename helper. Update package internals and have external-package tests construct fixture paths with `filepath.Join` rather than expanding the production API for test setup.
6. **Audit the resulting API.** Review `go doc` output and cross-package references for the touched packages. Remove any newly dead comments or helpers, but record additional nontrivial API redesigns in `notes/todo.md` instead of expanding this cleanup.
