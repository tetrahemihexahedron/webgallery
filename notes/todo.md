# Development plan

These are the next items to implement from `notes/backlog.md`, listed in recommended implementation order.

Make small commits. Each numbered implementation step should be at least one independently passing commit. Run `go test ./...`, `go vet ./...`, and `staticcheck ./...` for every commit.

## 1. Lazy-load later gallery images

Add native lazy loading to images that are unlikely to be visible initially while leaving the first 15 images at the browser's default eager behavior.

1. Add a package constant for the eager-image count and a `LazyLoad` field to `templateImage`; in `newTemplateData`, set the field from each image's position in the already-sorted rendering order.
2. Update the single gallery template to conditionally render `loading="lazy"` on the `<img>` element when `LazyLoad` is true, without duplicating the picture markup.
