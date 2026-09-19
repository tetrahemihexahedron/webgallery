# Development plan

These are the next six items to implement from `notes/todo.md`, listed in recommended implementation order.

Each item includes small implementation steps sized for focused commits.

## 1. Reject incomplete variant generation

**Type:** Fix
**Package(s):** `cmd/webimage`, `internal/variants`

`processImage` currently treats a partially successful generation result as success and can write an incomplete manifest. Processing should be all-or-nothing for each source image.

Small implementation steps:

1. **Add focused processor coverage.**
   - Use one small package-local fake generator rather than introducing a reusable test framework.
   - Cover a partial result containing a generated JPEG and a failed variant.
   - Cover an otherwise successful result containing only AVIF output.
   - Assert processing fails and removes the incomplete image directory; do not add separate index-level infrastructure for these cases.

2. **Fail on both error levels.**
   - Treat a non-nil error returned by `Generate` as a request-level processing failure.
   - Check `result.Err()` for per-variant failures and fail even when other files were generated.
   - Include the generated and failed counts in per-variant failure context.

3. **Require a usable JPEG fallback.**
   - Preserve the existing failure when no variants were generated.
   - Reject a result that has generated variants but no JPEG.

4. **Preserve cleanup behavior.**
   - Ensure files created for the failed image are removed through the existing cleanup path.
   - Keep the implementation and assertions at the image-processing boundary.

## 2. Replace path-based variant specifications

**Type:** Refactor
**Package(s):** `internal/variants`, `internal/image`, `cmd/webimage`

Move output-path construction and format handling behind a package-level request API. Keep this task behavior-preserving: request validation policy, cancellation, output verification, and actual-dimension reporting remain separate work.

Small implementation steps:

1. **Introduce the request and result types.**
   - Export a `Request` containing `SourcePath`, `OutputDir`, `Widths`, and `Formats []image.Format`.
   - Change `Result.Generated` to `[]image.Variant` with paths relative to `OutputDir`.
   - Change `Failure` to identify the requested format and width without exposing a complete output path.
   - Keep the existing request-level versus per-variant error contract.

2. **Move output planning into `internal/variants`.**
   - Replace extension-inferred `Spec` values with typed format and width combinations from `Request`.
   - Construct output filenames and absolute command arguments inside the package.
   - Keep the current filenames, iteration order, encoder settings, and validation behavior.

3. **Expose the package-level function.**
   - Implement `func Generate(req Request) (Result, error)` using the existing `exec.Command` behavior.
   - Do not add `context.Context`, cancellation handling, or new request validation in this task.

4. **Simplify the command integration.**
   - Use a command-local function type for dependency injection.
   - Build one `Request` from the source path, image directory, selected widths, and JPEG/AVIF formats.
   - Remove `Vipsthumbnail`, `Spec`, `variantSpecs`, `variantFilename`, and `identifyVariants` rather than maintaining both APIs.

5. **Adapt existing regression tests.**
   - Update the current package and processor tests for the request/result API.
   - Preserve their existing assertions for generated files, result ordering, encoder behavior, and error classification.
   - Add no new test harness solely for this refactor.

## 3. Default gallery sorting to processed dates

**Type:** Fix
**Package(s):** `cmd/gallery`, `internal/gallery`

Every processed image has a `processedAt` value, while `capturedAt` is optional. Make processed-date sorting the CLI default so the default workflow can render every valid collection, while keeping captured-date sorting strict when explicitly requested.

Small implementation steps:

1. **Update focused policy coverage.**
   - Update the existing argument-parsing expectation so omitting `-sort` selects `processed`.
   - Retain the existing sort tests showing that processed sorting permits an empty `capturedAt` and captured sorting rejects one.
   - Add a new rendering fixture only if those existing tests do not cover the implementation change.

2. **Change the CLI default.**
   - Default `cmd/gallery -sort` to `processed`.
   - Keep explicit `-sort captured` and `-sort processed` behavior unchanged.

3. **Document the default.**
   - State the default and the stricter captured-date requirement in the README or command help.

## 4. Reject overlapping input and output roots

**Type:** Fix
**Package(s):** `cmd/webimage`

Overlapping roots can cause generated output to be consumed as future input or place source files inside a tree the processor mutates. Reject unsafe layouts before reading metadata or creating output.

Small implementation steps:

1. **Add one table of root relationships.**
   - Cover equal roots, output beneath input, input beneath output, siblings, and merely prefix-similar names.
   - Keep the cases at the path-validation boundary without faking metadata or filesystem tools.

2. **Validate roots before processing.**
   - Use cleaned absolute paths and path-aware relative checks rather than string-prefix checks.
   - Return a direct error before reading metadata, loading the index, or creating files.

3. **Keep the boundary narrow.**
   - Apply the check in `cmd/webimage`, where both configured roots are available.
   - Leave symlink-resolved containment to the existing unplanned paths-policy item.

## 5. Create image directories exclusively

**Type:** Fix
**Package(s):** `cmd/webimage`

A random image-directory collision is very unlikely, but it must not cause the processor to reuse or later remove a pre-existing directory. Prefer a direct error over retry machinery that exists mainly to support an implausible test case.

Small implementation steps:

1. **Separate directory selection from creation.**
   - Keep date-based relative path construction separate from filesystem mutation.
   - Create the year/month parents with `os.MkdirAll`, then create only the random leaf with `os.Mkdir`.

2. **Fail directly on a collision.**
   - Return a clear error when the random leaf already exists.
   - Do not make random ID generation injectable or add collision retries.

3. **Protect pre-existing directories.**
   - Run cleanup only after the current attempt successfully created the leaf directory.
   - Rely on the existing processor integration test for normal directory creation; do not add persisted collision tests unless the implementation develops nontrivial collision handling.

## 6. Reject variant overwrites

**Type:** Fix
**Package(s):** `internal/variants`

Variant generation should never replace an existing output file implicitly, even when the package is called outside the normal fresh-directory workflow.

Small implementation steps:

1. **Detect existing destinations.**
   - Check each output path before invoking `vipsthumbnail`.
   - Treat an existing file as a per-variant failure with a direct error.

2. **Preserve existing files.**
   - Skip generation for an existing destination and leave its contents unchanged.
   - Continue attempting other requested variants.

3. **Add one focused integration case.**
   - Pre-create one destination and request it alongside one variant that can be generated normally.
   - Assert the existing contents are unchanged and the result records one failed and one generated variant.
   - Do not introduce a command-runner abstraction solely for this assertion.
