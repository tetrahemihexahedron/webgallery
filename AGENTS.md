This directory contains the code for a command-line script written in Go used to prepare image files for webpages. It is a work in progress, so some features may be only partially implemented.

## Application

webimage is a small Go CLI for preparing JPEG photos for Rosie the Dog’s website and for a future website, Albuquerque Dog. Entry point is cmd/webimage; flags are -incoming, -output, and -quiet.

The processor reads image metadata from the incoming directory using external exiftool, skips non-JPEGs, creates an output subdirectory like <output>/<year>/<month>/<random-id>/, copies the original to orig.jpg, generates .jpg and .avif variants at widths 400/800/1200/1600 capped by source width using external vipsthumbnail/libvips, then writes a per-image manifest.json.

Package layout: internal/config parses CLI flags; internal/metadata wraps exiftool; internal/variants wraps vipsthumbnail; internal/image holds domain structs/format parsing; internal/manifest writes JSON manifests; cmd/webimage/processor.go orchestrates the workflow.

## Coding standards

This script is going to be used only by the developer and should not be made more general than needed or incorporate unneeded features.

The code does not need to be Windows compatible and can assume that file paths have Unix-style separators, without comment in most cases.

Be reticent to add dependencies: make sure they add enough value to compensate for the extra maintenance burden.

The developer is fairly inexperienced with Go and wants to learn how to write excellent Go code, following best practices. The code should be well-organized and easy to understand, maintain, and modify. Refactor often, striving for simplicity, consistency, and well-designed code.

## Testing

Any helpful tests can be written when developing, but they maybe temporary. The following guidelines apply to persisted, version-tracked tests.

Generally, test only exported behavior. Prefer external test packages such as `package metadata_test`; use same-package tests only when an unexported helper is complicated enough to justify direct coverage.

Full test coverage is not a goal; remember that this will be used only by the developer, and don't add tests for unlikely errors.

Assume external tools, like exiftool and libvips/vipsthumbnail, are available in the testing environment. Integration-style tests for wrappers around those tools are useful, but test this project’s behavior rather than exhaustively testing the external tools.

Favor table-driven tests when cases share the same setup and assertions. Use slices of structs with clear, succinct `name` fields, `tc` for the current case, and `got`/`want` names for actual and expected values.

Put reusable fixture files in package-local `testdata` directories. Create temporary directory layouts with `t.TempDir`.

For JSON output, prefer unmarshaling into small test structs and comparing values rather than comparing raw formatted JSON strings.

Use standard library comparisons such as `slices.Equal` for comparable slices and `reflect.DeepEqual` for nested structs, maps, or slices. Avoid adding assertion dependencies.

Error messages should identify the function under test and use got/want wording. Use `t.Fatalf` when later assertions depend on the failed condition.

Avoid testing exact returned error strings. Prefer checking that an error was returned, using `errors.Is`/`errors.As` or checking that the message contains relevant text. Exact string comparisons are acceptable for structured user-facing output fields when that text is part of the behavior being tested.

Keep small test helpers near the bottom of the file.
