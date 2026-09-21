# Development plan

These are the next six items to implement from `notes/todo.md`, listed in recommended implementation order.

Each item includes small implementation steps sized for focused commits.

## 1. Harden source copying

**Type:** Fix
**Package(s):** `cmd/webimage`

`copyFile` opens its destination with truncation and relies on deferred closes, so a mistaken destination can destroy an existing file and delayed write errors can be lost. Make copying fail safely while preserving the current policy of creating a plain site artifact rather than cloning source permissions or timestamps.

Small implementation steps:

1. **Add focused copy-policy coverage.**
   - Add a small same-package test for `copyFile`; direct coverage is justified here because its destructive-open and resource-lifecycle behavior is difficult to observe cleanly through the processor workflow.
   - Verify a successful copy with `bytes.Equal`, then use a table for same path, hard-link alias, symlink alias, and unrelated pre-existing destination cases that share setup and assertions.
   - For rejected destinations, check that an error occurred and that original contents are unchanged; use `errors.Is` only if the implementation exposes a meaningful filesystem error, and do not compare complete error strings.
   - Keep source permissions and timestamps out of the tested contract: standardize new destination files on mode `0644` subject to the process umask, and do not copy source metadata.

2. **Reject destructive destinations before writing.**
   - Open and inspect the source before opening the destination.
   - When the destination resolves to the source, including through a hard link or symlink, return a direct same-file error using file identity rather than lexical path comparison.
   - Create the destination exclusively so any other pre-existing file, directory, or dangling symlink is rejected instead of truncated.

3. **Report copy completion errors.**
   - Check the `io.Copy` result and explicitly close the destination so delayed write or close failures are returned.
   - Close every successfully opened file on all paths and remove only a partial destination created by the current attempt when copying does not complete.
   - Do not introduce a filesystem abstraction or persisted tests solely to force implausible close failures; cover the realistic success and destructive-destination paths instead.

## 2. Validate JPEG processing requirements in the processor

**Type:** Fix
**Package(s):** `cmd/webimage`

`internal/metadata` currently prevents incomplete JPEG records from reaching the processor, so `cmd/webimage` does not enforce all of the invariants needed before hashing and generating files. Establish those checks at the processing boundary before relaxing metadata extraction in the following item.

Small implementation steps:

1. **Validate after format dispatch.**
   - Keep the existing format check first so unsupported records are skipped without requiring image dimensions or capture metadata.
   - For JPEG records, require positive width and height before building the source path, hashing the file, or creating output.
   - Allow an empty `capturedAt` with processed-date directories, require it when `--dir-date=captured` is selected, and reject any non-empty value that is not in the canonical captured-date format.

2. **Report processing problems consistently.**
   - Return validation failures through the existing per-file `imageProblem` path and progress reporting rather than as request-level errors.
   - Keep title and description optional and do not add source-format policy to lower-level image or path types.
   - Reuse `internal/image` date parsing instead of introducing another datetime parser.

3. **Add focused processor coverage.**
   - Add a table to the existing same-package processor tests; package-local coverage is appropriate because the command workflow has no exported API and the validation must be observed before filesystem work begins.
   - Start each invalid case from otherwise-valid JPEG metadata and change only width, height, or capture data; include an unsupported format with missing dimensions to prove format skipping still happens first.
   - Check whether a problem was returned and match only relevant message substrings, then retain the existing processor integration test as coverage for a valid JPEG flowing through the complete workflow.

## 3. Limit metadata validation to extraction concerns

**Type:** Fix
**Package(s):** `internal/metadata`

Once the processor enforces JPEG requirements, metadata reading can remain format-neutral. It should report exiftool and conversion failures, but it should not reject records merely because fields required by the current JPEG workflow are absent.

Small implementation steps:

1. **Pass through incomplete records.**
   - Continue giving exiftool-reported errors precedence and require `FileName` and `FileType`, because callers cannot identify or classify a record without them.
   - Stop treating missing or non-positive dimensions as metadata-layer problems; copy width and height values as reported so the processor can decide whether they matter for the selected format.
   - Keep title and description optional and unchanged.

2. **Keep capture-date normalization at the extraction boundary.**
   - Treat a missing `DateTimeOriginal` as an empty optional `capturedAt`.
   - When `DateTimeOriginal` is present, continue parsing exiftool's external representation and formatting the canonical value expected by `image.Metadata`.
   - Keep malformed non-empty dates as per-file metadata problems: decoding external metadata belongs here, while deciding whether a valid or missing capture date is required belongs in the processor.

3. **Update exported integration coverage.**
   - Through `Exiftool.Read` in external package `metadata_test`, change the existing missing-dimensions expectation from a problem to structured metadata with zero values.
   - Add a small reusable non-JPEG fixture without dimensions and locate records by filename rather than assuming order, because deterministic ordering is a later plan item.
   - Retain the malformed-date case using relevant error context rather than an exact parser message, and compare structured `image.Metadata` values instead of raw exiftool JSON.

## 4. Treat no-record directories as empty

**Type:** Fix
**Package(s):** `internal/metadata`

A successful exiftool invocation can return no records for a directory that contains only subdirectories because reads are intentionally non-recursive. That should have the same result as an empty directory instead of becoming a request-level error.

Small implementation steps:

1. **Define successful no-output behavior.**
   - When exiftool succeeds with empty stdout, stat the requested path.
   - Return an empty `Result` for any directory, regardless of whether it contains ignored subdirectories.
   - Retain a direct error for a non-directory path that unexpectedly produces no output and preserve command failures unchanged.

2. **Simplify the filesystem check.**
   - Replace the current entry-count helper with a directory-type check; do not recurse or inspect subdirectory contents.
   - Keep all filesystem errors wrapped with the requested path for context.

3. **Extend directory integration coverage.**
   - Extend the exported `Exiftool.Read` integration test in `package metadata_test` with a table of temporary directory layouts for an empty directory and a directory containing only one or more subdirectories.
   - Create layouts with `t.TempDir` and assert both cases return empty metadata and problem slices without placing files inside the nested directories.
   - Keep command-failure assertions type- or substring-based; do not add exact comparisons of exiftool diagnostics.

## 5. Make metadata result order deterministic

**Type:** Fix
**Package(s):** `internal/metadata`

Exiftool and filesystem enumeration order are not part of the package contract, so otherwise-identical runs can return metadata and problems in different orders. Return both result slices in a documented, predictable order.

Small implementation steps:

1. **Sort both result channels.**
   - Sort `Result.Metadata` and `Result.FileProblems` independently by `FileName` in ascending lexical order after record conversion.
   - Use a stable comparison so duplicate filenames, if exiftool ever reports them, retain their original relative order.
   - Preserve the existing distinction between usable metadata and per-file problems.

2. **Add mixed-result coverage.**
   - Exercise the exported `Exiftool.Read` API from `package metadata_test`, building a temporary directory from reusable valid and invalid fixtures created in deliberately non-lexical order.
   - Compare the resulting metadata slice with `slices.Equal` and explicit `got`/`want` values; check problem filenames in order while matching only relevant message substrings so external-tool wording does not make the test brittle.
   - Keep any fixture-copying helper near the bottom of the test file, mark it with `t.Helper`, and avoid direct tests of sorting helpers unless their logic becomes independently nontrivial.

3. **Document the result contract.**
   - Update the exported `Result` documentation to state that both slices are ordered by filename.
   - Keep sorting inside `internal/metadata` rather than relying on callers to normalize external-tool output.

## 6. Record generated dimensions accurately

**Type:** Fix
**Package(s):** `internal/image`, `internal/variants`, `internal/manifest`, `cmd/webimage`

Manifests currently derive variant heights from source metadata and requested widths. Auto-rotation, shrink-only generation, and encoder rounding can make those estimates differ from the files that were actually written, causing incorrect intrinsic dimensions in gallery HTML. Carry measured output dimensions through the processing pipeline instead.

Small implementation steps:

1. **Make variant dimensions explicit.**
   - Add `Height` to `image.Variant` and define its `Width` and `Height` as the generated file's actual pixel dimensions rather than the requested resize descriptor.
   - Keep the requested width separately in `plannedVariant` and `variants.Failure` so command arguments, filenames, and error reporting retain their current meaning.
   - Update representative literals and structured assertions throughout the processor, variants, and manifest tests without adding a second public variant type or an assertion dependency.

2. **Measure successful outputs.**
   - After `vipsthumbnail` succeeds, invoke `vipsheader` from the installed libvips tools to read the output file's `width` and `height` fields in one command.
   - Parse exactly two positive integers and treat missing, malformed, or non-positive dimensions as a per-variant failure rather than reporting the variant as generated.
   - Wrap inspection failures with the output path and requested format/width, but test only error identity or relevant context rather than complete command output; leave broader output-format validation to the separate generated-file verification item.
   - Use the real installed tools for integration coverage and do not introduce command injection solely to manufacture malformed `vipsheader` output.

3. **Persist measured dimensions.**
   - Populate each successful `image.Variant` with the measured width and height.
   - Change `manifest.FromProcessed` to copy those values directly instead of recomputing height from the source aspect ratio, and remove the now-unused floating-point calculation.
   - Keep source dimensions unchanged because they describe the original input metadata, not an auto-oriented derivative.

4. **Cover rotation, rounding, and shrink-only behavior.**
   - Put one small reusable EXIF-orientation fixture in package-local `testdata`, reuse the existing 800x1067 image for rounding coverage, and extend the table-driven `variants.Generate` integration test in external package `variants_test`.
   - Compare returned dimensions with values read independently by the existing exiftool-based test helper for actual JPEG and AVIF files, including a shrink-only request larger than the source.
   - Update the processor integration test to assert exact variant structs, and test manifest conversion with structured `Manifest` values or unmarshaled JSON fields rather than formatted JSON text.
