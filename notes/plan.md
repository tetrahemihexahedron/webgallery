# Development plan

These are the next items to implement from `notes/todo.md`, listed in recommended implementation order.

Each numbered implementation step is intended to be a focused, independently passing commit. Run `go test ./...`, `go vet ./...`, and `staticcheck ./...` for every commit.

## 1. Strip whitespace from extracted metadata

Normalize textual metadata as it enters the application instead of making downstream packages guess whether values are already clean.

The policy is to apply `strings.TrimSpace` to every exiftool-supplied textual metadata value that is content or control data: file format, title, description, `DateTimeOriginal`, and exiftool's error text. Perform normalization before required-field checks, date parsing, problem classification, and construction of `metadata.File`. This makes whitespace-only required values missing, whitespace-only optional values empty, and returned metadata consistently normalized.

`FileName` is the exception: it is a filesystem identifier rather than descriptive metadata. Unix filenames may legally begin or end with whitespace, so changing it would make the returned path refer to a different file. Preserve it exactly and validate it as a `paths.RelPath` under task 1.

**Packages involved:**

- `internal/metadata`: define and apply the normalization policy and own all direct tests.
- `cmd/webimage`: consume already-normalized values; no additional trimming should be added here.

**Implementation commits:**

1. **Normalize exiftool records at conversion time.** Add one clear normalization step before `processOutput` validates each record, remove redundant trimming from `parseDateTimeOriginal`, and document the returned-value contract on `metadata.File` or `Result`. Add one representative metadata case using user-authored fields—for example, a padded title and a whitespace-only description—to cover both trimming and normalization to empty. Do not add separate whitespace cases for every exiftool-controlled field, and do not add a filename-with-whitespace test. Keep the existing format, date, and error tests to confirm their normal behavior is unchanged, and keep canonical captured-date formatting unchanged.

## 2. Make datetime parser normalization consistent

Both image datetime parsers represent canonical persisted formats. They should validate exact stored text rather than silently normalizing it; external-input cleanup belongs in `internal/metadata` as established by task 2.

**Packages involved:**

- `internal/image`: define and test the strict parser contract.
- `internal/gallery`: relies on these parsers when sorting persisted index data and should continue to reject malformed values.
- `internal/index` and `internal/manifest`: future validation will rely on the same strict behavior; no code change is required there yet.

**Implementation commits:**

1. **Make processed datetime parsing strict.** Remove whitespace trimming from `image.ParseProcessedAt`, add or improve exported documentation for both datetime parsers, and move the existing surrounding-whitespace case from the successful parser table to the error table. Retain the existing UTC, whole-second RFC3339 requirement and formatting behavior. Audit call sites to ensure none rely on `ParseProcessedAt` for user-input normalization; the parser test is sufficient, so do not duplicate it with gallery, index, or manifest regression tests unless one of those packages later adds behavior beyond delegating to the parser.

## 3. Minimize exported surfaces

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
