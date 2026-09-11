# libvips versions

**Important**: Use the `--output` option and absolute paths when calling `vipsthumbnail`.

We install libvips with the platform package manager, which means the devcontainer and local machine may have different versions.

The important compatibility issue is `vipsthumbnail` output handling:

- The current devcontainer libvips supports `--output`, and not `--path`.
- Newer libvips versions support `--path`; `--output` may still exist but it is deprecated/hidden.

The difference is how relative paths are handled:

- With `--output`, a relative path is interpreted relative to the input file's directory. For example, processing `testdata/duplicate.jpg` with output `tmp/.../w400.jpg` will try to write `testdata/tmp/.../w400.jpg`.
- With `--path`, a relative path is interpreted relative to the current directory.

We deal with this by using `--output` and absolute paths when calling `vipsthumbnail`.
