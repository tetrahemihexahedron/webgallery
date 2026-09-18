# TODO / review notes

These notes collect possible maintainability/readability improvements, potential bugs, and feature ideas discovered during package-by-package review. Items are grouped using the branch types from `.pi/prompts/implement-plan-item.md`: `refactor`, `fix`, `feature`, `docs`, `test`, and `chore`.

At the time of the original review, `go test ./...`, `go vet ./...`, and `staticcheck ./...` passed.

## Represented in `plan.md`

These items have been promoted to `notes/plan.md`. The plan is the authoritative source for their scope and implementation steps.

### Refactor

- **Improve `cmd/webimage` names and signatures (`cmd/webimage`):** Rename ambiguous workflow types, fields, helpers, and the shadowing `metadata` variable; group source-image data into a small type; and narrow `processImage` and `variantSpecs` to their actual inputs.
- **Clarify manifest and index APIs (`internal/manifest`, `internal/index`):** Use clearer DTO and exported type names, add explicit processed-image conversion and symmetric read/write APIs, clarify or split index updates, and divide `index.go` by responsibility.
- **Improve gallery data flow and option parsing (`cmd/gallery`, `internal/gallery`):** Centralize sort parsing, parse sort keys once, clarify mutation and rendering behavior, improve internal/template names, document `Render`, and normalize options in one place.
- **Revisit the variants API (`internal/variants`, `cmd/webimage`):** Export and document result-error handling, improve Vips-related naming, move from path-inferred encoders toward typed format/config and request data, clarify partial results, and return enough information to avoid reparsing generated paths.

### Fix

- **Choose a missing-`capturedAt` policy (`cmd/webimage`, `internal/gallery`, `internal/image`):** Processing can currently produce images that the default gallery sort rejects. Choose and document one policy, cover processing and rendering with missing-date tests, and make the smallest consistent behavior change.
- **Handle partial variant generation (`cmd/webimage`, `internal/variants`, `internal/manifest`):** Partial failures can be treated as success and leave no JPEG fallback. Test this path, require a JPEG, choose an all-or-nothing or warning policy, preserve cleanup, and improve progress messages.
- **Make output creation safer (`cmd/webimage`, `internal/variants`):** Random directory collisions and existing variant files can lead to accidental writes. Create random leaf directories exclusively with bounded retries, make ID generation testable, and reject variant overwrites.
- **Clean up CLI parsing and validation (`cmd/webimage`, `cmd/gallery`):** Use `flag.ContinueOnError`, reject positional arguments, validate input directories, decide whether output roots are created or required, and clarify gallery output targets, with focused parser tests.
- **Fix user-facing output generation (`internal/gallery`, `internal/variants`):** Preserve full URL prefixes when joining paths and sanitize raw external-command output by trimming, limiting, and replacing non-printable content, with regression tests.

## Unplanned items

Packages are sorted by path. Within each package, items use the type order `refactor`, `fix`, `feature`, `docs`, `test`, then `chore`; empty type sections are omitted.

### All packages

#### Refactor

- **Centralize enum parsing:** Enum-like values still use scattered conversion, validation, defaults, and error wording. Add parser functions where repetition warrants them, such as a future `ParseDirDate`; `ParseSortField` is already represented in the plan.

#### Fix

- **Make external commands cancellable:** `exiftool` and `vipsthumbnail` have no context, cancellation, or timeout, so a stuck process can hang a CLI indefinitely. Thread `context.Context` into the wrappers and use `exec.CommandContext`, adding focused cancellation tests if this behavior is implemented.
- **Record generated dimensions accurately:** Raw metadata dimensions can disagree with auto-rotated or rounded output. Have variant generation verify and return actual dimensions, write those values to manifests, and cover orientation and rounding with image fixtures.
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
- **Keep the development container complete:** Ensure it installs tools actually used by development and tests, including `exiftool`, libvips/`vipsthumbnail`, and `file` if workflows continue to invoke it.

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

- **Keep consumer interfaces local:** Keep small interfaces such as `metadataReader` and `variantGenerator` near the command code that consumes them; share them only if multiple consumers genuinely require the same contract.
- **Simplify error cleanup:** After the partial-result policy in the plan is settled, consider a deferred cleanup guard in `processImage` to reduce repeated cleanup branches without hiding which partial files are retained.

#### Fix

- **Harden source copying:** `copyFile` can miss delayed close errors and does not guard against source and destination being the same file. Check close errors, reject same-file copies, and decide deliberately whether permissions and modification time should be preserved.
- **Define empty-run index behavior:** Empty or all-skipped first runs may leave no `index.json`. Decide whether these runs create or refresh an empty index, then test empty directories, skipped-only directories, and the resulting index contents.
- **Reject overlapping input and output roots:** Overlapping paths can cause generated output to be consumed as input or otherwise put source data at risk. Validate the roots before processing.

#### Feature

- **Add processing controls only when needed:** Possible controls include dry-run, configurable widths/formats/quality, moving successful inputs to a done directory, and explicit force/reprocess behavior. Keep each option separate unless they share a clear workflow.
- **Support recursive incoming directories:** If nested input is needed, define path and duplicate-basename behavior together with the metadata changes needed to retain source directories.
- **Make output layout configurable:** Allow omission or customization of the `yyyy/mm` prefix when pages need flatter image URLs, with tests for each supported layout.
- **Preserve source extensions for new formats:** `orig.jpg` is correct while inputs are JPEG-only. If additional source formats are accepted, derive the original-copy extension from the validated source format.

#### Test

- **Keep processor tests readable:** If the processor integration test becomes difficult to scan, introduce a small scenario builder or named assertion helpers rather than one large setup block.

### `internal/gallery`

#### Refactor

- **Reduce duplicated index and manifest data:** Fields such as title, dates, and SHA-256 can diverge between the index and manifest. Consider keeping only stable lookup data in the index and loading display metadata from manifests.

#### Fix

- **Validate loaded gallery data:** `loadImages` does not reconcile index and manifest fields, and rendering trusts dimensions. Validate consistency and positive dimensions at the loading boundary, with mismatch and invalid-dimension tests.
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

- **Validate index data at its boundary:** `ReadDir` and direct `UpdateFile` use can admit malformed dates and hashes, duplicate directories or manifest paths, paths outside the image directory, and incomplete processed images. Add an `Index.Validate` path used before writing and after reading, enforce `<dir>/manifest.json` relationships, and cover malformed and duplicate fixtures.
- **Report non-directory inputs clearly:** `ReadDir` currently returns a lower-level path error when given a file. Detect this case and add a focused test.
- **Define missing-output-root behavior:** `UpdateFile` assumes the root exists. Either create it or return a direct validation error, and test the selected contract.
- **Detect unknown JSON fields if useful:** Use `json.Decoder.DisallowUnknownFields` if catching hand-edited schema mistakes is more valuable than forward compatibility, with an unknown-field fixture.
- **Decide whether rename-level durability is sufficient:** Temp-file-and-rename is atomic but not explicitly `fsync`ed. Add file and directory syncs only if crash durability matters for this personal tool.

#### Feature

- **Add index maintenance commands only when needed:** Possible operations include rebuilding from manifests, pruning missing image directories, and reporting duplicate source hashes.

### `internal/manifest`

#### Refactor

- **Write variants deterministically:** Sort variants by format and width before encoding so output does not depend on generation order.
- **Use descriptive local names:** Rename short parameters such as `i image.Processed` to `img` or `processed` when touching the surrounding code.

#### Fix

- **Validate manifests consistently:** Validate positive source dimensions, non-empty variant paths, positive widths, dates, hashes, supported formats, required JPEG fallbacks, extension/format agreement, and duplicate paths or width descriptors on read and write. Add focused invalid fixtures and reject unknown JSON fields only if strict schema checking is desired.

### `internal/metadata`

#### Refactor

- **Request only needed exiftool tags:** Fetching all metadata is slower and leaves behavior exposed to unrelated tags. Limit the query to consumed fields and evaluate `-G1`, `-a`, and `-s` if duplicate tag names need disambiguation.
- **Split command, decoding, and conversion stages:** If metadata handling grows, replace `fetchExiftoolOutput`/`processOutput` with clearer `runExiftool`, `decodeExiftoolJSON`, and record-conversion steps.
- **Improve metadata names:** Consider `Result.Images` or `Files`, `Problems` or `FileErrors`, `ExifTool` or `Reader`, and `exiftoolRecord` instead of names that obscure whether a value represents one file or a collection.

#### Fix

- **Skip unsupported files before requiring dimensions:** Non-images and PDFs can be reported as missing metadata before the processor has a chance to skip them. Filter by file type first and test non-JPEG inputs without dimensions.
- **Treat empty exiftool output consistently:** A directory containing only subdirectories currently becomes an error. Treat successful no-output runs as empty after validating the input directory, with a focused fixture.
- **Make result order deterministic:** Sort metadata and file problems by filename rather than relying on filesystem or exiftool order, and test both slices.
- **Choose a whitespace policy:** Titles and descriptions pass through unchanged, including accidental surrounding spaces. Decide whether to preserve or normalize them, and test the chosen behavior.

#### Feature

- **Support nested source paths:** Preserve `SourceFile` or `Directory` when recursive processing is added so duplicate basenames remain distinguishable.
- **Improve capture-date extraction:** If `DateTimeOriginal` is absent or incomplete, define an explicit priority among subsecond/original, XMP creation, EXIF creation/modification, and file modification dates. Preserve timezone offsets when useful, consider exiftool's `-d` formatting, and add representative fixtures.
- **Define description-field precedence:** If `Description` is insufficient, choose an order among `ImageDescription`, XMP description, `Caption-Abstract`, `Headline`, `Title`, and `ObjectName` rather than accepting whichever tag happens to appear.

#### Test

- **Handle optional external tooling in tests:** If lean environments are supported, add an exiftool availability helper and skip only integration tests that genuinely require it.
- **Keep external-tool assertions resilient:** Prefer error types or relevant substrings over exact comparisons of long Go/exiftool messages.

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

#### Refactor

- **Keep execution injectable only when tests need it:** A small command runner can make warnings and missing outputs deterministic to test without over-generalizing the wrapper.

#### Fix

- **Handle successful command output deliberately:** Any `vipsthumbnail` output currently causes an error even with a zero exit status. Decide whether warnings are acceptable and test warning-on-success behavior.
- **Verify generated files:** A zero exit status does not prove that output exists, is non-empty, or has the requested format. Validate generated files and test missing or malformed outputs; actual dimensions are covered by the cross-package item above.
- **Define output-directory responsibility:** `Generate` requires parent directories to exist without documenting that contract. Either create them or return a direct error, with a test for missing parents.
- **Strengthen same-file protection if needed:** Lexical path comparison misses hard links and symlinks. Compare file identities when the added safety justifies the complexity, and add corresponding tests.
- **Define source-format validation:** Decide whether the wrapper validates supported inputs or deliberately delegates that responsibility to callers and libvips.
- **Cover cleaned parent-path output roots:** Add a regression check for roots containing `..` so `vipsthumbnail` receives cleaned absolute paths and diagnostics remain readable.

#### Feature

- **Add bounded parallel generation:** If generation speed matters, run variants concurrently with a small limit and preserve complete error reporting.
- **Support additional output formats:** Add WebP or other formats only when a website needs them, with encoder and output-validation tests.
- **Add crop/cover generation:** Provide an optional crop mode and decide whether centered crops are sufficient or need a focus position, with image-level tests.

#### Test

- **Handle optional libvips tooling in tests:** If lean environments are supported, add a `vipsthumbnail` availability helper and skip only integration tests that require it.
