# Development plan

These are the next items planned to be done from the todo list notes/todo.md.

Each item should include detailed small implementation steps sized for focused commits.

## Refactoring

### 1. Improve `cmd/webimage` names and function signatures

**Package(s):** `cmd/webimage`

Several names are generic or misleading. Some function signatures pass loosely related values separately. Clear names and small grouping types will make the command easier to understand.

Small implementation steps:

1. **Rename workflow types.**
   - `processor` -> `imageProcessor`
   - `result` -> `processResult`
   - `fileProblem` -> `imageProblem`
   - Update tests in the same commit.

2. **Rename misleading fields.**
   - Rename `result.dirProcessed` to something like `incomingDir` if it is still useful.
   - Or remove the field if no caller needs it.

3. **Rename workflow methods and helpers.**
   - `processDir` -> `processIncomingDir`
   - `processFile` -> `processImage`
   - `hashFile` -> `fileSHA256`
   - `imgDirRelPath` -> `newImageDirRelPath`
   - `filename` -> `variantFilename`
   - `deleteRemnants` -> `removeImageDir`

4. **Rename local variables for clarity.**
   - Change the loop variable `metadata` to `meta` so it does not shadow the imported `metadata` package.
   - Rename `imageProcessed` to `processedImg` consistently.

5. **Introduce a source-image grouping type.**
   - Add a small struct such as:
     ```go
     type sourceImage struct {
         metadata image.Metadata
         path     paths.AbsPath
         sha256   string
     }
     ```
   - Build this once after path creation and hashing.

6. **Simplify `processImage` signature.**
   - Change from `processFile(metadata image.Metadata, sourceHash string, sourceAbsPath paths.AbsPath)` to `processImage(source sourceImage)`.
   - Keep the old behavior unchanged.

7. **Simplify variant planning signature.**
   - Change `variantSpecs(imgDir paths.AbsPath, img image.Processed)` to take only what it uses, such as `variantSpecs(imgDir paths.AbsPath, sourceWidth int)`.
   - Or move the planning into `internal/variants` later if that package owns more of the generation request.

### 2. Improve gallery data flow and option parsing

**Package(s):** `internal/gallery`, `cmd/gallery`

Gallery rendering has a reasonable shape already, but a few changes would make it clearer where parsing, loading, sorting, and template conversion happen.

Small implementation steps:

1. **Add `gallery.ParseSortField`.**
   - Implement `func ParseSortField(s string) (SortField, error)`.
   - Move valid-value error wording into `internal/gallery`.
   - Update `cmd/gallery` to use it.

2. **Rename internal loaded-image types.**
   - `galleryImage` -> `loadedImage` or `galleryItem`.
   - `indexImage` field -> `entry` or `indexEntry`.
   - Keep this as a pure rename commit.

3. **Improve `Render` documentation.**
   - Expand the comment to explain that `Render` reads `index.json`, loads manifests, sorts images, and writes HTML fragments.
   - Mention that `Options.ImagesRoot` must point at a webimage output root.

4. **Make sort handling more explicit.**
   - Add a helper that returns a parsed sort key, such as `sortKey(img loadedImage, field SortField) (time.Time, error)`.
   - Sort by `time.Time` rather than validating timestamps and then comparing strings.

5. **Consider avoiding in-place sorting.**
   - Either rename `sortImages` to make mutation obvious, or change it to return a sorted copy.
   - Choose the simpler call site.

6. **Rename template DTOs if they still feel unclear.**
   - Possible names: `templateData` -> `galleryHTMLData`, `templateImage` -> `pictureData`, `templateSource` -> `sourceData`, `templateFallback` -> `imgData`.
   - Do this only if the new names improve readability at call sites.

7. **Normalize options in one place.**
   - Add a small helper if needed, such as `normalizeOptions(opts Options) (Options, error)`.
   - This is a good place for future defaults like `sizes` or fallback width.

### 3. Revisit the variants API shape

**Package(s):** `internal/variants`, `cmd/webimage`

The caller currently constructs detailed output paths, the variants package infers encoding from path extensions, and `cmd/webimage` later reparses the paths to identify generated variants. A clearer API would put variant-generation knowledge in one place.

Small implementation steps:

1. **Export `Result.Err`.**
   - Rename `func (r Result) err() error` to `func (r Result) Err() error`.
   - Keep `Generate`'s return signature unchanged initially.
   - Update callers/tests to use the exported method where helpful.

2. **Document partial-result semantics.**
   - Add comments to `Generate` and `Result` explaining whether callers should inspect both `Generated` and `Failed`.
   - This can be done before behavior changes.

3. **Rename `Vipsthumbnail`.**
   - Consider `VipsThumbnail` for Go readability.
   - Update command construction in `cmd/webimage` and tests.

4. **Rename command-local variables.**
   - In `generateVariant`, rename `width` -> `sizeArg`, `path` -> `outputArg`, and `out` -> `cmdOutput`.
   - This clarifies the difference between image dimensions, filesystem paths, and command arguments.

5. **Introduce a format/config type.**
   - Add an internal or exported type for generated formats and encoder options.
   - Move `determineEncoderOptions(path)` toward a format-driven helper.

6. **Add a higher-level request type experimentally.**
   - Consider:
     ```go
     type Request struct {
         SourcePath paths.AbsPath
         OutputDir  paths.AbsPath
         Widths     []int
         Formats    []FormatConfig
     }
     ```
   - Keep the existing `Spec` API until the request type proves simpler.

7. **Move variant identification closer to generation.**
   - Have the variants package return enough information for manifests, such as format, relative path, requested width, and maybe actual dimensions.
   - Then remove or simplify `cmd/webimage.identifyVariants`.

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
