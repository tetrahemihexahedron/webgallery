# Development plan

These are the next items to implement from `notes/backlog.md`, listed in recommended implementation order.

Make small commits. Each numbered implementation step should be at least one independently passing commit. Run `go test ./...`, `go vet ./...`, and `staticcheck ./...` for every commit.

## 1. Rename the project and commands

Adopt `webgallery` as the repository, checkout-root, and project name; change the Go module path to `tetrahemihexahedron/webgallery`; and rename the command binaries to `prepgallery` and `rendergallery`. Keep the commands separate and preserve their existing flags and behavior.

**Packages involved:**

- All packages: update the module path and imports.
- `cmd/webimage`: rename to `cmd/prepgallery` and update command-specific references.
- `cmd/gallery`: rename to `cmd/rendergallery` and update command-specific references.
- Repository configuration and documentation: update tracked project, workspace, command, and path references.

**Implementation commits:**

1. **Rename the Go module.** Change the module path to `tetrahemihexahedron/webgallery` and update imports throughout the repository without otherwise changing package behavior.
2. **Rename the preparation command.** Move `cmd/webimage` to `cmd/prepgallery` and update its tests, help text, ignored binary name, and command-specific documentation while preserving all flags and behavior.
3. **Rename the rendering command.** Move `cmd/gallery` to `cmd/rendergallery` and update its tests, help text, and command-specific documentation while preserving all flags and behavior. Do not rename the separate `internal/gallery` package.
4. **Update the tracked project identity.** Change remaining tracked references from `webimage` to `webgallery`, including the development-container name and workspace paths, general documentation, and planning-note package headings or command references. Verify that any retained uses of the old names are intentional.

**Required user step:**

1. **Rename the GitHub repository.** Ask Sunny to rename the repository from `webimage` to `webgallery` in the GitHub repository settings, then wait for confirmation before continuing. This hosted-repository change cannot be performed with Git over SSH.

**Final local setup after the user confirms:**

- Change `origin` to `git@github.com:tetrahemihexahedron/webgallery.git` and verify fetch and push access over SSH.
- Rename the checkout root to `webgallery`, reopening or rebuilding the development container if necessary, and verify the commands and required checks from the renamed workspace.
