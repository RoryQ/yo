# Field list of {{ .Name }}

{{ range .Fields -}}
* {{ .Name }} {{ .SpannerDataType }} {{ .Type }}{{ if ne .Len -1 }} len={{ .Len }}{{ end }}{{ if ne .ElemLen -1 }} elemLen={{ .ElemLen }}{{ end }}
{{ end }}
# Primary Key

{{ range .PrimaryKeyFields -}}
* {{ .Name }} {{ .SpannerDataType }} {{ .Type }}
{{ end }}
# Index list of {{ .Name }}

{{ range .Indexes -}}
* {{ .IndexName }}
{{ end -}}
