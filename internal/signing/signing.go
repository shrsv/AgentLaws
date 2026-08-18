// Package signing signs and verifies the canonical representation of a
// compiled lawbook, independent of any particular renderer (PLAN1 §25).
package signing

import "errors"

// ErrNotImplemented is returned by every stub in this package until signing
// is implemented per PLAN1 §64 Milestone 7.
var ErrNotImplemented = errors.New("signing: not implemented")

// ErrVerificationFailed indicates a signature did not match the provided
// canonical representation - the tamper-detection case in PLAN1 §49.
var ErrVerificationFailed = errors.New("signing: verification failed")

// Sign signs the canonical (already-serialized) representation of a
// compiled lawbook and returns an opaque signature string.
func Sign(canonical []byte, key string) (string, error) {
	return "", ErrNotImplemented
}

// Verify checks that signature was produced for canonical by a trusted key.
func Verify(canonical []byte, signature string) error {
	return ErrNotImplemented
}
