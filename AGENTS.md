This directory contains Go command-line tools used to prepare image files for webpages and generate reusable HTML for a gallery of photos. All of the basic functionality has been implemented, but it has not yet been put into use.

## Application

webimage contains two small Go CLIs for Rosie the Dog’s website and for a future website, Albuquerque Dog.

cmd/prepgallery prepares JPEG photos. Its flags are -incoming, -output, -quiet, and -dir-date. The processor reads image metadata from the incoming directory using external exiftool, skips non-JPEGs and duplicate source files, creates an output subdirectory like <output>/<year>/<month>/<random-id>/, copies the original to orig.jpg, generates .jpg and .avif variants at widths 400/800/1200/1600 capped by source width using external vipsthumbnail/libvips, writes a per-image manifest.json, and updates a collection-level index.json.

cmd/gallery reads a webimage output directory and writes static HTML `<picture>` fragments. Its flags are -images, -output, -url-prefix, and -sort. It reads index.json and each image’s manifest.json, sorts by captured or processed datetime, emits AVIF `<source>` elements when available, and uses JPEG variants for the fallback `<img>` srcset.

Package layout: cmd/prepgallery and cmd/gallery parse CLI flags and orchestrate their workflows; internal/gallery loads processed image metadata and renders gallery HTML; internal/metadata wraps exiftool; internal/variants wraps vipsthumbnail; internal/image holds domain structs and format/datetime helpers; internal/manifest reads and writes per-image JSON manifests; internal/index reads and writes the collection index; internal/paths holds small validated filesystem path types.

## Coding standards

This script is going to be used only by the developer and should not be made more general than needed or incorporate unneeded features.

The code does not need to be Windows compatible and can assume that file paths have Unix-style separators, without comment in most cases.

Be reticent to add dependencies: make sure they add enough value to compensate for the extra maintenance burden.

The developer is fairly inexperienced with Go and wants to learn how to write excellent Go code, following best practices. The code should be well-organized and easy to understand, maintain, and modify. Refactor often, striving for simplicity, consistency, and well-designed code.

## Testing

The following guidelines apply to persisted, version-tracked tests. Temporary tests may be useful while developing; put them in a separate new `_test.go` file and remove them before committing unless they provide lasting value and follow these guidelines.

Prioritize maintainability and readability. Test important behavior and realistic failure paths. Full test coverage is not a goal; remember that this will be used only by the developer. Do not add low-value tests for implausible failures or misuse that only project code can introduce.

Generally, test only exported behavior. Prefer external test packages such as `package metadata_test`; use same-package tests only when an unexported helper is complicated enough to justify direct coverage.

Assume external tools, like exiftool and libvips/vipsthumbnail, are available in the testing environment. Integration-style tests for wrappers around those tools are useful, but test this project’s behavior rather than exhaustively testing the external tools.

Favor table-driven tests when cases share the same setup and assertions. Use slices of structs with clear, succinct `name` fields, `tc` for the current case, and `got`/`want` names for actual and expected values.

Put reusable fixture files in package-local `testdata` directories. Create temporary directory layouts with `t.TempDir`.

Use realistic, representative test data. For invalid cases, begin with otherwise-valid data and change only the fields relevant to the case so the failure exercises the intended rule.

For JSON output, prefer unmarshaling into small test structs and comparing values rather than comparing raw formatted JSON strings.

Use standard library comparisons such as `slices.Equal` for comparable slices and `reflect.DeepEqual` for nested structs, maps, or slices. Avoid adding assertion dependencies.

Error messages should identify the function under test and use got/want wording. Use `t.Fatalf` when later assertions depend on the failed condition.

Avoid testing exact returned error strings. If only failure matters, check that an error was returned. When error identity or type matters, use `errors.Is` or `errors.As`; when context matters, check for a relevant substring. Error text does not need to be checked in every case. Exact string comparisons are acceptable for structured user-facing output fields when that text is part of the behavior being tested.

Keep small package-local test helpers near the bottom of the file and mark them with `t.Helper()`.
