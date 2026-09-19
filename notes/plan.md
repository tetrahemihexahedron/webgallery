# Development plan

These are the next two items to implement from `notes/todo.md`, listed in recommended implementation order.

Each item includes small implementation steps sized for focused commits.

## 1. Create image directories exclusively

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

## 2. Reject variant overwrites

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
