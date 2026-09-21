# TODO / review notes

These notes collect possible maintainability/readability improvements, potential bugs, and feature ideas discovered during package-by-package review. Items are grouped using the branch types from `.pi/prompts/implement-plan-item.md`: `refactor`, `fix`, `feature`, `docs`, `test`, and `chore`.

At the time of the original review, `go test ./...`, `go vet ./...`, and `staticcheck ./...` passed.

## Represented in `plan.md`

These items have been promoted to `notes/plan.md`. The plan is the authoritative source for their scope and implementation steps.

### Fix

- **Validate JPEG processing requirements in the processor (`cmd/webimage`):** After format dispatch, require processable JPEG dimensions and enforce the configured capture-date requirement before hashing or creating output.
- **Limit metadata validation to extraction concerns (`internal/metadata`):** Pass through missing dimensions for processor policy while retaining exiftool errors, record identity requirements, and normalization of present capture dates.
- **Treat no-record directories as empty (`internal/metadata`):** Treat successful empty exiftool output as an empty result for directories, including directories containing only ignored subdirectories.
- **Make metadata result order deterministic (`internal/metadata`):** Sort usable metadata and per-file problems independently by filename before returning them.
- **Record generated dimensions accurately (`internal/image`, `internal/variants`, `internal/manifest`, `cmd/webimage`):** Measure successful outputs and carry their actual width and height through processed-image data and manifests instead of estimating from source metadata.

## Unplanned items

Packages are sorted by path. Within each package, items use the type order `refactor`, `fix`, `feature`, `docs`, `test`, then `chore`; empty type sections are omitted.

### All packages

#### Refactor

- **Centralize enum parsing:** Enum-like values still use scattered conversion, validation, defaults, and error wording. Add parser functions where repetition warrants them, such as a future `ParseDirDate`.

#### Fix

- **Make external commands cancellable:** Thread `context.Context` through the command workflows and both the metadata and variants APIs, use `exec.CommandContext` for `exiftool` and `vipsthumbnail`, treat cancellation as a request-level error, and add focused cancellation coverage for both tools.
- **Make CLI parsing non-exiting:** Both commands use `flag.ExitOnError` and accept positional arguments. Use `flag.ContinueOnError`, return parse and help results to the caller, reject positional arguments, and add focused argument tests without invoking subprocesses.
- **Validate CLI filesystem roots before work:** Confirm the `webimage` incoming root and gallery images root exist and are directories before external commands or index reads, and create the `webimage` output root after overlap validation so its missing-root behavior is intentional.
- **Use consistent atomic output writes:** Index writes use temp-file-and-rename while manifests and gallery output do not. Consider a small shared atomic-write helper, apply it where interrupted writes could corrupt output, and test that render failures do not replace an existing file.
- **Prevent concurrent output mutation:** Two `webimage` processes can race while updating the output root and index. Add a lightweight lock only if accidental concurrent runs are plausible.
- **Clean up command-line error output:** The `main` packages use `log.Fatal`, which adds timestamps to user-facing errors. Prefer explicit stderr output and exit status handling, then add small smoke tests for the resulting CLI messages.

#### Feature

- **Version generated JSON schemas:** Add schema/version fields to index and manifest files if migrations are likely enough to justify maintaining versioned formats.
- **Add richer image data end to end:** Add location, keywords, captions, original-file references, dominant colors, or blur placeholders only for concrete consumers, updating metadata extraction, domain types, manifests, and gallery rendering together.

#### Docs

- **Expand the README:** Add representative commands, required external tools, the output layout, and important assumptions.
- **Improve Go documentation:** Review exported identifiers and package comments, documenting semantics that are not obvious from names and types.

#### Test

- **Add an end-to-end workflow test:** Run `webimage` on fixture images and then run `gallery` against the generated output.
- **Reduce repeated test setup:** Introduce shared helpers for patterns such as `mustAbs`, `mustRel`, and fixture copying only if the duplication starts obscuring tests.

#### Chore

- **Add a standard verification command:** Provide a small script, `just` target, or CI job that runs `go test`, `go vet`, and `staticcheck`.
- **Keep development-tool installation explicit:** The container setup installs exiftool and libvips but relies on other features for tools such as `staticcheck`, and `file` is absent. Install or verify every tool used by documented checks and workflows, adding `file` only if the developer workflow continues to require it.

### `cmd/gallery`

#### Refactor

- **Avoid unnecessary output buffering:** `renderGallery` buffers the complete HTML document. Stream to the selected writer if galleries become large and doing so preserves the current no-partial-output guarantees.
- **Clarify option names:** Consider names such as `ImagesDir` or `WebimageRoot` instead of `ImagesRoot`, and `OutputFile` instead of `OutFile`, when related code is next changed.

#### Fix

- **Define output-parent behavior:** Writing to an output file fails when its parent directory is absent. Either create the parent or return an intentional, direct error, and test writing to a temporary output path.

#### Feature

- **Accept externally controlled ordering:** If an upload site later controls display order in SQLite, accept a simple ordered export rather than coupling the gallery command to the database.

### `cmd/webimage`

#### Refactor

- **Consolidate image-directory cleanup:** `processImage` repeats cleanup after exclusive leaf creation. Use a deferred guard that is armed only after the current attempt creates the leaf and disarmed after the manifest succeeds, while preserving joined cleanup errors and the current all-or-nothing image policy.

#### Fix

- **Define empty-run index behavior:** Empty or all-skipped first runs may leave no `index.json`. Decide whether these runs create or refresh an empty index, then test empty directories, skipped-only directories, and the resulting index contents.

#### Feature

- **Add processing controls only when needed:** Possible controls include dry-run, configurable widths/formats/quality, moving successful inputs to a done directory, and explicit force/reprocess behavior. Keep each option separate unless they share a clear workflow.
- **Support recursive incoming directories:** If nested input is needed, define path and duplicate-basename behavior together with the metadata changes needed to retain source directories.
- **Make output layout configurable:** Allow omission or customization of the `yyyy/mm` prefix when pages need flatter image URLs, with tests for each supported layout.
- **Preserve source extensions for new formats:** `orig.jpg` is correct while inputs are JPEG-only. If additional source formats are accepted, derive the original-copy extension from the validated source format.

#### Test

- **Keep processor tests readable:** If the processor integration test becomes difficult to scan, introduce a small scenario builder or named assertion helpers rather than one large setup block.

### `internal/gallery`

#### Refactor

- **Parse sort keys once:** Sorting currently validates datetime strings and then compares their original string values. Parse the selected field into `time.Time` keys and sort by those values instead.
- **Make sorting mutation explicit:** `sortImages` sorts its input slice in place. Either rename it to make that mutation clear or return a sorted copy, choosing whichever keeps the call site simpler.
- **Reduce duplicated index and manifest data:** Fields such as title, dates, and SHA-256 can diverge between the index and manifest. Consider keeping only stable lookup data in the index and loading display metadata from manifests.

#### Fix

- **Validate loaded gallery data:** `loadImages` does not reconcile index and manifest fields, and rendering trusts dimensions. Validate consistency and positive dimensions at the loading boundary, with mismatch and invalid-dimension tests.
- **Preserve full URL prefixes:** `publicURL` uses `path.Join`, which corrupts prefixes such as `https://example.com/images`. Preserve empty and relative-prefix behavior while joining full `http://` or `https://` prefixes without rewriting their scheme separators.
- **Escape all generated URLs:** `srcset` assembly assumes generated filenames need no escaping. Centralize public URL construction and escaping before accepting arbitrary paths or prefixes, and test unusual valid path characters.
- **Choose an alt-text policy:** Empty title and description produce empty alt text, which may incorrectly mark gallery photos as decorative. Choose whether to allow, warn, fail, or supply a fallback, and test that policy.

#### Feature

- **Make picture markup configurable:** Potential options include `sizes`, fallback width, sort direction, loading/decoding attributes, CSS classes, captions, `<figure>` wrappers, original-image links, and filtering by date. Add focused command and rendering tests with each supported option.
- **Render a single image:** Add a single-image rendering path that reuses the full gallery template-data conversion; expose it through `cmd/gallery` only if needed.

#### Test

- **Cover gallery loading and fallback behavior:** Add tests for a missing manifest, HTML escaping in titles and descriptions, fallback selection when all JPEG variants are smaller than the preferred width, and a multiple-image load/sort/render fixture.

### `internal/image`

#### Refactor

- **Introduce validation only where it removes repetition:** Domain structs use plain strings, so invalid dates, hashes, and empty required fields are easy to construct. Add small typed wrappers or `Validate` methods only when multiple callers need the same rules, with focused tests.
- **Centralize format knowledge:** Add format-to-extension and format-to-MIME helpers, and consider richer parse results when callers need to distinguish missing, unknown, and unsupported formats.
- **Use a path type for metadata filenames:** Change `Metadata.FileName` to `paths.RelPath` once the metadata package can guarantee safe relative paths.
- **Shorten redundant path field names:** Consider `Processed.Dir` or `ImageDir` instead of `DirRelPath` because the type already communicates relativity.

#### Fix

- **Clarify capture-time semantics:** `capturedAt` has no timezone, so it represents camera wall time rather than an absolute instant. Document that distinction from UTC `processedAt`; capture offsets if absolute cross-time-zone ordering becomes necessary, and test date round trips.
- **Normalize format parsing deliberately:** `ParseFormat` does not trim whitespace and maps unknown, missing, and unsupported values to `FormatOther`. Decide which distinctions callers need and add cases such as whitespace and `.jpeg`.

#### Docs

- **Document exported domain data:** Explain source versus variant dimensions and capture versus processing timestamps where the types alone are insufficient.

#### Test

- **Document processed-time precision:** Add a test for nanosecond truncation in `FormatProcessedAt` if callers rely on it.

### `internal/index`

#### Refactor

- **Centralize relative-path conversion:** Move `relPathFromAbs` into `internal/paths` if another package needs the same operation.
- **Make index ordering deliberate:** Either preserve and document insertion order or sort/rebuild deterministically so output does not vary accidentally.

#### Fix

- **Validate index-level metadata:** Add an `Index.Validate` path used after reading and before writing, require a valid `generatedAt`, and define the valid empty-index representation with focused read and update cases.
- **Validate index entries and uniqueness:** Validate entry dates and SHA-256 values, require each manifest to equal `<dir>/manifest.json`, and reject duplicate directories, manifest paths, and hashes with malformed and duplicate fixtures.
- **Report non-directory inputs clearly:** `ReadDir` currently returns a lower-level path error when given a file. Detect this case and add a focused test.
- **Define missing-output-root behavior:** `Index.Update` assumes the root exists. Either create it or return a direct validation error for direct package callers, and test the selected contract.
- **Detect unknown JSON fields if useful:** Use `json.Decoder.DisallowUnknownFields` if catching hand-edited schema mistakes is more valuable than forward compatibility, with an unknown-field fixture.
- **Decide whether rename-level durability is sufficient:** Temp-file-and-rename is atomic but not explicitly `fsync`ed. Add file and directory syncs only if crash durability matters for this personal tool.

#### Feature

- **Add index maintenance commands only when needed:** Possible operations include rebuilding from manifests, pruning missing image directories, and reporting duplicate source hashes.

### `internal/manifest`

#### Refactor

- **Write variants deterministically:** Sort variants by format and width before encoding so output does not depend on generation order.

#### Fix

- **Validate manifest metadata:** Use one validation path from conversion, read, and write to require positive source dimensions, valid processed and optional captured dates, and a valid SHA-256 value, with focused otherwise-valid fixtures for each field.
- **Validate manifest variant sets:** Require non-empty relative paths, positive dimensions, supported formats, a JPEG fallback, matching file extensions, and unique paths and width descriptors within each format on conversion, read, and write.

### `internal/metadata`

#### Refactor

- **Request only needed exiftool tags:** Fetching all metadata is slower and leaves behavior exposed to unrelated tags. Limit the query to consumed fields and evaluate `-G1`, `-a`, and `-s` if duplicate tag names need disambiguation.
- **Split command, decoding, and conversion stages:** If metadata handling grows, replace `fetchExiftoolOutput`/`processOutput` with clearer `runExiftool`, `decodeExiftoolJSON`, and record-conversion steps.
- **Improve metadata names:** Consider `Result.Images` or `Files`, `Problems` or `FileErrors`, `ExifTool` or `Reader`, and `exiftoolRecord` instead of names that obscure whether a value represents one file or a collection.

#### Fix

- **Choose a whitespace policy:** Titles and descriptions pass through unchanged, including accidental surrounding spaces. Decide whether to preserve or normalize them, and test the chosen behavior.

#### Feature

- **Support nested source paths:** Preserve `SourceFile` or `Directory` when recursive processing is added so duplicate basenames remain distinguishable.
- **Improve capture-date extraction:** If `DateTimeOriginal` is absent or incomplete, define an explicit priority among subsecond/original, XMP creation, EXIF creation/modification, and file modification dates. Preserve timezone offsets when useful, consider exiftool's `-d` formatting, and add representative fixtures.
- **Define description-field precedence:** If `Description` is insufficient, choose an order among `ImageDescription`, XMP description, `Caption-Abstract`, `Headline`, `Title`, and `ObjectName` rather than accepting whichever tag happens to appear.

### `internal/paths`

#### Refactor

- **Centralize repeated path operations:** Add `Base`, `Ext`, or `RelFrom` only if they replace repeated direct `filepath` calls in multiple packages.
- **Separate filesystem and URL paths:** `RelPath` is used for both, although URL escaping follows different rules. Add a URL conversion helper or distinct type if URL handling expands.
- **Reassess the package's value periodically:** Keep the path types if they prevent real mistakes; otherwise simplify them rather than accumulating conversion noise.
- **Prefer concise APIs where clear:** Methods such as `root.Join(rel)` or helpers such as `RelFrom` may read better than longer names when path-heavy call sites are next revised.

#### Fix

- **Distinguish relative files from directory roots:** `RelPath` accepts `.`, which is useful for a root but invalid for manifest file values. Add file- and directory-specific validation, plus cleaning cases such as `a/../../b`, `./..`, and `a/..`.
- **Handle zero values consistently:** Add `IsZero` or boundary validation where zero `AbsPath` and `RelPath` values currently fail late or inconsistently.
- **Reject clearly invalid filesystem characters:** Detect NUL at construction time if an earlier error is useful, with a focused test.
- **Decide the symlink containment policy:** Lexical validation cannot prevent symlinks escaping a root. If containment becomes a requirement, evaluate resolved file identities or Go's `os.Root` and add symlink tests; otherwise document that it is out of scope.

### `internal/variants`

#### Fix

- **Define variant request validation:** Decide which malformed shared request fields reject the whole request and which format/width-specific problems remain per-variant failures. Cover missing paths, a missing or non-directory output location, non-positive or duplicate widths, empty or duplicate format lists, unsupported formats, and source-format responsibility with focused tests.
- **Accept diagnostics from successful variant commands:** Do not fail solely because `vipsthumbnail` writes output with a zero exit status; rely on destination validation for success and test the result-classification logic without adding a command-runner abstraction unless it becomes necessary.
- **Sanitize command output in errors:** Raw `vipsthumbnail` output can be binary or very large. Trim whitespace, replace non-printable bytes, cap retained output, and use the sanitized text in generation errors, with direct helper tests.
- **Verify generated files:** A zero exit status does not prove that output exists, is non-empty, or has the requested format. Validate generated files and test missing or malformed outputs; dimension measurement is covered by the planned cross-package item.
- **Strengthen same-file protection if needed:** Lexical path comparison misses hard links and symlinks. Compare file identities when the added safety justifies the complexity, and add corresponding tests.
- **Cover cleaned parent-path output roots:** Add a regression check for roots containing `..` so `vipsthumbnail` receives cleaned absolute paths and diagnostics remain readable.

#### Feature

- **Add bounded parallel generation:** If generation speed matters, run variants concurrently with a small limit and preserve complete error reporting.
- **Support additional output formats:** Add WebP or other formats only when a website needs them, with encoder and output-validation tests.
- **Add crop/cover generation:** Provide an optional crop mode and decide whether centered crops are sufficient or need a focus position, with image-level tests.
