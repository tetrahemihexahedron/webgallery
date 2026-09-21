# Development plan

These are the next four items to implement from `notes/todo.md`, listed in recommended implementation order.

Each item includes small implementation steps sized for focused commits.

## 1. Limit metadata validation to extraction concerns

**Type:** Fix
**Package(s):** `internal/metadata`

Once the processor enforces JPEG requirements, metadata reading can remain format-neutral. It should report exiftool and conversion failures, but it should not reject records merely because fields required by the current JPEG workflow are absent.

Small implementation steps:

1. **Pass through incomplete records.**
   - Continue giving exiftool-reported errors precedence and require `FileName` and `FileType`, because callers cannot identify or classify a record without them.
   - Stop treating missing or non-positive dimensions as metadata-layer problems; copy width and height values as reported so the processor can decide whether they matter for the selected format.
   - Keep title and description optional and unchanged.

2. **Keep capture-date normalization at the extraction boundary.**
   - Treat a missing or whitespace-only `DateTimeOriginal` as an empty optional `capturedAt`.
   - When `DateTimeOriginal` is present, continue trimming only as part of decoding the external value, parse exiftool's date layout, and use `image.FormatCapturedAt` to produce the canonical value required by `image.Metadata`; never copy exiftool's date text directly into the domain struct.
   - Keep malformed non-empty dates as per-file metadata problems: decoding and normalizing external metadata belongs here, while validating the internal contract and deciding whether a missing capture date is acceptable belongs in the processor.

3. **Update exported integration coverage.**
   - Through `Exiftool.Read` in external package `metadata_test`, change the existing missing-dimensions expectation from a problem to structured metadata with zero values.
   - Add a small reusable non-JPEG fixture without dimensions and locate records by filename rather than assuming order, because deterministic ordering is a later plan item.
   - Retain the malformed-date case using relevant error context rather than an exact parser message, and compare structured `image.Metadata` values instead of raw exiftool JSON.

## 2. Treat no-record directories as empty

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

## 3. Make metadata result order deterministic

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

## 4. Record generated dimensions accurately

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
