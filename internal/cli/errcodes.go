package cli

import (
	"errors"

	"github.com/shrsv/AgentLaws/internal/resolver"
)

// errNotFound is returned by CLI-local lookups (e.g. section show) that
// don't go through internal/resolver but should still map to ExitNotFound.
var errNotFound = errors.New("not found")

// ExitCode maps an error returned by Execute to a process exit code, per
// the convention documented in PLAN1 §32.
func ExitCode(err error) int {
	if err == nil {
		return ExitOK
	}
	if errors.Is(err, resolver.ErrNotFound) || errors.Is(err, errNotFound) {
		return ExitNotFound
	}
	var usageErr *UsageError
	if errors.As(err, &usageErr) {
		return ExitUsage
	}
	return ExitError
}

// UsageError signals a CLI usage problem (bad flag combination, etc.)
// distinct from a validation/compile failure.
type UsageError struct {
	Msg string
}

func (e *UsageError) Error() string { return e.Msg }
