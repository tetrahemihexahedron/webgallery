# Development plan

These are the next items planned to be done from the todo list notes/todo.md.

Each item should include detailed small implementation steps sized for focused commits.

## Refactoring

### 1. Clarify variant generator results and names

**Package(s):** `internal/variants`, `cmd/webimage`

Variant generation can succeed for some specifications and fail for others, so returning both a `Result` and an error duplicates failure state. Make `Result` the sole generation outcome, clarify its error contract, and improve the command-argument names without changing generation behavior or redesigning the request API.

Small implementation steps:

1. **Export `Result.Err` and return only `Result` from `Generate`.**
   - Rename `func (r Result) err() error` to `func (r Result) Err() error`.
   - Add an unexported result-level error so `Err()` can report source-path validation failures as well as aggregate `Failure.Err` values.
   - Change the existing `Generate(source, specs)` method to return only `Result`, then update the consumer interface, callers, and tests to obtain the error from `result.Err()`.
   - Add focused coverage for successful results, multiple per-spec failures, and source-path validation failures.

2. **Document partial-result semantics.**
   - Document that, after source-path validation, `Generate` attempts every specification and records each attempt in either `Generated` or `Failed`.
   - Explain that `Result.Err()` reports source-path validation errors and aggregates per-spec failures.
   - Keep generation and partial-result behavior unchanged apart from the return-signature refactor.

3. **Clarify command-local variable names.**
   - In `generateVariant`, rename `width` to `sizeArg`, `path` to `outputArg`, and `out` to `cmdOutput`.
   - Keep this as a pure naming change.

## Behavior improvements

### 1. Choose and implement a consistent missing `capturedAt` policy

**Package(s):** `cmd/webimage`, `internal/gallery`, `internal/image`

Current behavior lets `webimage` produce images that `gallery` cannot render with its default `-sort captured`.

Small implementation steps:

1. **Choose the policy and document it.**
   - Options:
     - require captured dates during processing,
     - default gallery sorting to processed date,
     - or sort missing captured dates last.

2. **Add tests for the chosen policy.**
   - Include an image with missing `DateTimeOriginal` or empty `CapturedAt`.
   - Test both processing and gallery rendering behavior.

3. **Implement the smallest behavior change.**
   - If requiring captured dates, reject/skip files earlier.
   - If changing gallery sorting, update defaults and sorting behavior.

4. **Update README/help text.**
   - Make the behavior visible to the developer using the tools.

### 2. Handle partial variant generation correctly

**Package(s):** `cmd/webimage`, `internal/variants`, `internal/manifest`

Current behavior may write a manifest after some variant generation failures, as long as at least one file was generated.

Small implementation steps:

1. **Add a failing test in `cmd/webimage`.**
   - Use a fake variant generator that returns some generated specs and some failed specs.
   - Assert the image is not written as successfully processed, or assert the chosen warning behavior.

2. **Require JPEG fallback variants.**
   - Before writing a manifest, confirm generated variants include at least one JPEG.
   - Treat no JPEG fallback as a processing failure.

3. **Decide all-or-nothing versus warnings.**
   - The simplest safe policy is all-or-nothing: any failed variant means the image processing fails and cleanup runs.

4. **Implement the chosen policy.**
   - If all-or-nothing, check `result.Err()` or `len(result.Failed)` after generation and return an error.
   - Keep cleanup behavior consistent.

5. **Improve progress messages.**
   - If partial failures are reported, include how many variants succeeded and failed.

### 3. Make image directory creation and output writes safer

**Package(s):** `cmd/webimage`, `internal/variants`

Random ID collisions are unlikely, but the current code could write into an existing image directory. Variant output overwrites are also currently allowed by default.

Small implementation steps:

1. **Split directory path generation from directory creation.**
   - Keep `newImageDirRelPath` responsible for the relative path only.
   - Add a helper such as `createImageDir(outRoot, date)` that creates the directory.

2. **Use exclusive directory creation.**
   - Use `os.Mkdir` for the final random directory rather than `os.MkdirAll` for the full image directory.
   - Create year/month parents first, then create the random leaf exclusively.

3. **Retry on collision.**
   - If `os.Mkdir` returns `fs.ErrExist`, generate another ID and retry a bounded number of times.

4. **Add deterministic collision tests.**
   - Make ID generation injectable or pass in a test generator that returns a duplicate once and then a unique ID.

5. **Add no-overwrite behavior for variants.**
   - Check for an existing output file before running `vipsthumbnail`.
   - Return a clear error unless a future force mode is explicitly added.

### 4. Clean up CLI parsing and input validation

**Package(s):** `cmd/webimage`, `cmd/gallery`

Better parsing and validation will make the tools easier to test and less surprising to use.

Small implementation steps:

1. **Switch to `flag.ContinueOnError`.**
   - Do this separately for each command.
   - Keep tests focused on unknown flags and invalid values.

2. **Reject positional arguments.**
   - After `flags.Parse(args)`, check `flags.NArg()`.
   - Return a clear error if extra args are present.

3. **Validate `webimage -incoming`.**
   - Check that it exists and is a directory.
   - Return a direct error before calling exiftool.

4. **Validate or create `webimage -output`.**
   - Decide whether the command creates the output root or requires it.
   - Implement and document one behavior.

5. **Validate `gallery -images`.**
   - Check that it is a directory containing `index.json`, or rely on `index.ReadDir` but improve the error.

6. **Clarify gallery output destination.**
   - Replace `UseStdout` plus zero `OutFile` with a small output target representation if it improves readability.

### 5. Fix user-facing output generation issues

**Package(s):** `internal/gallery`, `internal/variants`

Two user-visible issues stand out: full URL prefixes are broken, and raw command output can make error messages unreadable.

Small implementation steps:

1. **Add URL prefix tests.**
   - Cover empty prefix, `/images`, `/images/`, and `https://example.com/images`.

2. **Replace `path.Join` URL building.**
   - Use a URL-aware helper or simple slash trimming that preserves `https://`.
   - Keep generated relative paths unchanged.

3. **Add command-output sanitizing helper.**
   - Add a helper in `internal/variants`, such as `formatCommandOutput([]byte) string`.
   - It should trim whitespace, cap length, and replace non-printable bytes.

4. **Use sanitized output in variant errors.**
   - Update `generateVariant` errors to include sanitized command output.
   - Avoid dumping binary or huge output blobs.

5. **Add focused tests for sanitized errors.**
   - Unit test the sanitizer directly.
   - If command execution is later injectable, test `generateVariant` error wrapping too.
