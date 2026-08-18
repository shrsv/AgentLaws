// Package web embeds the built Preact UI (web/dist) so the Go binary can
// serve it without any external files at runtime. Run `npm run build` in
// this directory before `go build`/`go run` - dist/ is a build artifact and
// is not committed to source control.
package web

import "embed"

//go:embed all:dist
var DistFS embed.FS
