# TODO

## For AI

Sometimes the agent runs `file`, which isn't installed. Maybe it should be installed in the container.

## Go style

1. Learn about Go docs and add appropriate comments where missing.

## main

1. ChatGPT suggests that interfaces, like `metadataReader` should live here, with the explanation "In Go, interfaces usually belong near the code that consumes them, not near the code that implements them." What if I want to share interfaces, such as a MetadataReader, between different main packages in cmd? Should I not do that?

## webimage

1. Maybe add an option to not include the `/yyyy/mm` prefix on the output directory. I don't think I want it for all the pages in Rosie's website, or at least not for the home page.

1. gpt-5.5: `parseArgs` still uses `flag.ExitOnError`. `flag.ContinueOnError` would be safer and easier to test because parsing errors would return normally instead of potentially exiting. (Also true in `cmd/gallery`.)

## processor

1. While unlikely, one should still check for duplicate random id's and regenerate.

1. Check the source format and determine the extension of the moved source, instead of always assuming it is jpg.

1. Can I handle the clean-up when there is an error better, maybe with `defer`? Do I always want to delete all the generated variants or should I keep partial results?

## image

1. Consider changing `image.Metadata.Filename` to a `paths.RelPath`.

## metadata

1. Is the exiftool call what we want. ChatGPT suggested a number of flags that our call isn't using: `-G1`, `-a`, and `-s`.

1. Consider having `Read` accept a `context.Context` and using `exec.CommandContext` instead of `exec.Command`. This would allow callers to cancel the operation if it took too long, though I'm not sure it's necessary for my use case.

1. See what happens when incoming directory has subdirectories with image files. Think about what should happen.

1. There might be several different fields with a 'description'. ChatGPT suggests this command to extract and normalize the description:

   ```sh
   exiftool -json -G1 -s \
      -Description \
      -ImageDescription \
      -XMP-dc:Description \
      -Caption-Abstract \
      image.jpg
   ```

   and lists these as possible fields:

   ```
   Description
   ImageDescription
   XMP-dc:Description
   Caption-Abstract
   Headline
   Title
   ObjectName
   ```

1. Consider checking other date fields to determine the `CapturedAt` datetime, if `DateTimeOriginal` isn't found.

   ChatGPT suggested
   ```
   Composite:SubSecDateTimeOriginal
   EXIF:DateTimeOriginal
   XMP:DateCreated
   XMP:CreateDate
   EXIF:CreateDate
   EXIF:ModifyDate
   File:FileModifyDate
   ```

1. Perhaps add a test helper that checks for exiftool and skips the tests if it isn't found.

1. Consider formatting the date returned by exiftool with the `-d` flag instead of parsing and then formatting the value.

## variants

1. Crop photos when generating smaller sizes. This should presumably be an option. I don't know if one needs finer control, say, on how much to crop. Most of Rosie's photos could handle being cropped.

1. Tune libvips options. The formatting options depend on the resulting format and are sometimes vague, like `effort` and `quality`. I'm not sure I've landed on the best combo. Should the caller be allowed to change some of the options used?

1. Consider returning an error if a file already exists at the output file path, instead of overwriting the file. It could be overwritten with, say, an '--unsafe' or '--force' flag.

1. Consider having `Generate` accept a `context.Context`.

1. Consider generating each variant in a separate Go routine. AI suggested I'd still want to limit concurrency to something quite small, like 2, and that I'd have to use "a semaphore plus `sync.WaitGroup`" if I wanted to try all variants and join errors. The suggested code:

   ```go
   func Generate(ctx context.Context, source string, specs []Spec) error {
       if source == "" {
           return errors.New("source path is empty")
       }

       const concurrency = 2

       sem := make(chan struct{}, concurrency)

       var wg sync.WaitGroup
       var mu sync.Mutex
       var errs []error

       for _, spec := range specs {
           spec := spec
           wg.Add(1)

           go func() {
               defer wg.Done()

               sem <- struct{}{}
               defer func() { <-sem }()

               if err := generateVariant(ctx, source, spec); err != nil {
                   mu.Lock()
                   errs = append(errs, fmt.Errorf(
                       "generate %q at width %d: %w",
                       spec.OutPath,
                       spec.Width,
                       err,
                   ))
                   mu.Unlock()
               }
           }()
       }

       wg.Wait()
       return errors.Join(errs...)
   }
   ```

1. As ChatGPT pointed out, the returned error duplicates 'result state'. It suggests returning only a `Result` and having the calling code compute the error:

   ```go
   result := variants.Generate(source, specs)
   if err := result.Err(); err != nil {
     ...
   }
   ```

   [Some changes have been made here...]

1. Consider passing Generate a `Request` struct instead of a source and a slice of `Spec`. This seems much better: now, main needs to know how to form the output paths, which doesn't really seem like its business. ChatGPT's suggestions:

   ```go
   type Request struct {
       SourcePath string
       OutputDir  string
       BaseName   string
       Widths     []int
       Formats    []Format
   }

   type Variant struct {
       Format string `json:"format"`
       Width  int    `json:"width"`
       Path   string `json:"path"`
       URL    string `json:"url"`
   }
   ```

1. A fuller version from ChatGPT:

   ```go
   package variants

   type Format string

   const (
       AVIF Format = "avif"
       WebP Format = "webp"
       JPEG Format = "jpeg"
       PNG  Format = "png"
   )

   type FormatConfig struct {
       Format  Format
       Ext     string
       Quality int
   }

   type Request struct {
       SourcePath string
       OutputDir  string
       PublicDir  string
       BaseName   string
       Widths     []int
       Formats    []FormatConfig
       MaxWidth   int
   }

   type Variant struct {
       Format Format `json:"format"`
       Width  int    `json:"width"`
       Path   string `json:"-"`
       URL    string `json:"url"`
   }

   type VIPSThumbnail struct {
       Path string
   }
   ```

## manifest

1. Change it so that it writes first to a temp file and then copies, perhaps, like done in index.

## index

1. Read should probably error if it isn't given an actual directory.

1. Consider splitting files: index.go now contains types, reading, duplicate lookup, update/write, DTO conversion, and path helpers.
  ```text
  index.go   // public types + IndexPath + ImageDirsBySHA256
  read.go    // Read + file-to-typed conversion
  update.go  // UpdateFile + typed-to-file conversion + writing
  ```

## paths

1. Consider making use of `os.Root`. See https://go.dev/blog/osroot.

1. I'm not sure how much benefit this package is adding. Might be worth reviewing whether these types are useful and are defined appropriately.

## gallery

1. Improve description of Render.

1. gpt-5.5: A `gallery.ParseSortField(string)` helper may be nicer than converting and calling `IsValid` in cmd/gallery. A parser could centralize valid values and error wording.

1. gpt-5.5: `galleryImage` stores both index and manifest data. Later we may want to reconcile duplicate fields like `capturedAt`, `processedAt`, `title`, and `sha256`, because they exist in both the index and manifests. For now, sorting should use the index fields as planned. Me: Perhaps the index should contain only the sha and directory, to minimize the possibility of the data diverging.

1. Make sizes configurable. gpt-5.5: `imageSizes` and `fallbackDisplayWidth` are fine for now, but they are likely the first template-related values that will want a flag or template override.

1. This is unlikely to be a problem for my use, but as is, one can't use full URL prefixes. gpt-5.5: `publicURL` uses `path.Join`, which is good for `/images`, but bad for `https://example.com/images` because it can collapse the `//` after `https:`. If full URL prefixes are needed later, use `net/url` or a custom join helper.

1. Figure out what values of `loading` and `decoding` I want and include them in the template. Possibly make them configurable.

1. If the upload site later needs SQLite-controlled display order, add an input option to `gallery` for an ordered export.

1. Add the ability to render the html for a single image, given by the directory, I guess.

## Bugs and problems

1. When attempting to run `webimage` on a directory in `project/rosie-the-dog`, I got the following error:
   ```txt
   Error processing "IMG_1983.jpeg": no variants were generated: generating "../rosie-the-dog/public/images/gallery/2026/08/LHjHDcpLwg/w400.jpg" with width 400: unexpected output from image generation: ????C
   ```
   This was followed by a bunch of gobbledy-gook. I think that the output file was unrecognized because of my use of `..`:
   ```
   ~/projects/webimage% ./webimage -incoming ../rosie-the-dog/incoming -output ../rosie-the-dog/public/images/gallery
   ```
   The error message needs to be fixed.
