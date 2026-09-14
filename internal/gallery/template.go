package gallery

import "html/template"

var galleryTmplt = template.Must(template.New("gallery").Parse(galleryTmpltText))

const galleryTmpltText = `{{range .Images}}<picture>
{{range .Sources}}  <source
    type="{{.Type}}"
    srcset="{{.Srcset}}"
    sizes="{{.Sizes}}">
{{end}}  <img
    src="{{.Fallback.Src}}"
    srcset="{{.Fallback.Srcset}}"
    sizes="{{.Fallback.Sizes}}"
    width="{{.Fallback.Width}}"
    height="{{.Fallback.Height}}"
    alt="{{.Fallback.Alt}}">
</picture>
{{end}}`
