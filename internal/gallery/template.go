package gallery

import "html/template"

var galleryTemplate = template.Must(template.New("gallery").Parse(galleryTemplateText))

type templateData struct {
	Images []templateImage
}

type templateImage struct {
	Sources  []templateSource
	Fallback templateFallback
	LazyLoad bool
}

type templateSource struct {
	Type   string
	Srcset string
	Sizes  string
}

type templateFallback struct {
	Src    string
	Srcset string
	Sizes  string
	Width  int
	Height int
	Alt    string
}

const galleryTemplateText = `{{range .Images}}<picture>
{{range .Sources}}  <source
    type="{{.Type}}"
    srcset="{{.Srcset}}"
    sizes="{{.Sizes}}">
{{end}}  <img
{{if .LazyLoad}}    loading="lazy"
{{end}}    src="{{.Fallback.Src}}"
    srcset="{{.Fallback.Srcset}}"
    sizes="{{.Fallback.Sizes}}"
    width="{{.Fallback.Width}}"
    height="{{.Fallback.Height}}"
    alt="{{.Fallback.Alt}}">
</picture>
{{end}}`
