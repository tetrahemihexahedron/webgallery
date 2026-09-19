# Development plan

This is the next item to implement from `notes/todo.md`.

The item includes small implementation steps sized for focused commits.

## 1. Reject variant overwrites

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
