// Package version identifies the alaws binary itself - which build produced
// a given compiled lawbook, independent of the lawbook's own Git history
// (docs/PLAN1.md §24-§25).
package version

import "runtime/debug"

// Version and BuildTime are set at release-build time via -ldflags (see
// Makefile's build/build-go targets). They stay "dev"/"" for `go run` or an
// ad-hoc `go build` with no ldflags.
var (
	Version   = "dev"
	BuildTime = ""
)

// Info returns the alaws binary's version and build time. If Version wasn't
// set via -ldflags, it falls back to Go's own automatic VCS build stamping
// (runtime/debug.ReadBuildInfo, populated by the toolchain whenever the
// binary was built from inside a Git checkout - e.g. `go install
// .../alaws@latest`), so there is always something usable even for an
// unreleased build.
func Info() (version, buildTime string) {
	if Version != "dev" || BuildTime != "" {
		return Version, BuildTime
	}

	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return "dev", ""
	}

	var rev, t string
	for _, s := range bi.Settings {
		switch s.Key {
		case "vcs.revision":
			rev = s.Value
		case "vcs.time":
			t = s.Value
		}
	}

	v := bi.Main.Version
	if v == "" || v == "(devel)" {
		// Older toolchains (or a build outside a Git checkout) don't stamp
		// Main.Version with VCS info the way newer ones do - synthesize
		// something useful from the revision ourselves rather than
		// reporting a bare "(devel)".
		v = "dev"
		if rev != "" {
			short := rev
			if len(short) > 12 {
				short = short[:12]
			}
			v = "dev+" + short
		}
	}
	return v, t
}
