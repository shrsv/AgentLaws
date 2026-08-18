// Package compiler drives the compilation pipeline described in
// docs/PLAN1.md §18: discover -> load -> validate -> parse -> number ->
// construct the Lawbook IR -> run diagnostics.
package compiler

import (
	"errors"

	"github.com/athreyac4/agentlaws/internal/model"
	"github.com/athreyac4/agentlaws/internal/validator"
)

// ErrNotImplemented is returned by every stub in this package until the
// compiler is implemented per PLAN1 §64 Milestone 2-3.
var ErrNotImplemented = errors.New("compiler: not implemented")

// Options configures a single Compile call.
type Options struct {
	// Strict causes any warning-level diagnostic to be treated as an error.
	Strict bool
}

// Result is the outcome of compiling one lawbook cluster.
type Result struct {
	Lawbook     model.Lawbook
	Diagnostics []validator.Diagnostic
}

// Compile compiles the lawbook cluster rooted at path.
func Compile(path string, opts Options) (Result, error) {
	return Result{}, ErrNotImplemented
}
