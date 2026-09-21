# Development plan

These are the next items to implement from `notes/todo.md`, listed in recommended implementation order.

Each item includes small implementation steps sized for focused commits.

## 1. Record generated dimensions accurately

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
