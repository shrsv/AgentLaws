package alaws

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/shrsv/AgentLaws/internal/provenance"
	"github.com/shrsv/AgentLaws/internal/signing"
)

// Manifest is a compiled book's provenance manifest (docs/PLAN1.md §24): a
// content hash of its semantic content, the Provenance describing how/
// when/by-whom/with-what-tool it was compiled, and - once Sign has run - a
// cryptographic Signature over ContentHash.
type Manifest = provenance.Manifest

// LawHistory is one law's Git change history (docs/PLAN1.md §37-§39).
type LawHistory = provenance.LawHistory

// LawChange describes one law whose text differs between two compiled
// Books, matched by stable identity (docs/PLAN1.md §38).
type LawChange = provenance.LawChange

// LawbookDiff is the result of comparing two compilations of the same
// lawbook (docs/PLAN1.md §38).
type LawbookDiff = provenance.LawbookDiff

// Manifest returns b's provenance manifest: a content hash of its laws and
// commentary, plus the Provenance recorded when it was compiled. Available
// on every compiled Book, whether or not it has ever been signed.
func (b *Book) Manifest() (Manifest, error) {
	return provenance.BuildManifest(b.lawbook)
}

// History returns the Git change history of the law identified by
// citation, resolved via its current source location.
func (b *Book) History(citation string) (LawHistory, error) {
	return provenance.History(b.lawbook, citation)
}

// Diff compares two compiled Books - typically the same lawbook at two
// different Git revisions - matching sections and laws by stable identity
// rather than presentation citation numbers (docs/PLAN1.md §38).
func Diff(old, new *Book) LawbookDiff {
	return provenance.Diff(old.lawbook, new.lawbook)
}

const keyFileName = "id_ed25519"

// DefaultKeyPath resolves the signing private-key path per the storage
// hierarchy in docs/PLAN1.md §5: a repository-local ./.alaws/keys/ key
// takes precedence over the global ~/.alaws/keys/ one. If neither exists
// yet, it returns the global path as where `alaws keygen` should create
// one.
func DefaultKeyPath() (string, error) {
	candidates := []string{filepath.Join(".alaws", "keys", keyFileName)}
	if home, err := os.UserHomeDir(); err == nil {
		candidates = append(candidates, filepath.Join(home, ".alaws", "keys", keyFileName))
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c, nil
		}
	}
	return candidates[len(candidates)-1], nil
}

// GenerateKey creates a new Ed25519 signing keypair at path (private key)
// and path+".pub" (public key), creating parent directories as needed.
func GenerateKey(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return signing.GenerateKey(path)
}

// Sign builds b's provenance manifest and signs its content hash with the
// private key at keyPath, returning the manifest with Signature populated.
// The signature covers only the manifest's ContentHash - the semantic
// content of the lawbook - never the volatile Provenance fields
// (docs/PLAN1.md §25, §47).
func (b *Book) Sign(keyPath string) (Manifest, error) {
	manifest, err := b.Manifest()
	if err != nil {
		return Manifest{}, err
	}
	sig, err := signing.Sign([]byte(manifest.ContentHash), keyPath)
	if err != nil {
		return Manifest{}, err
	}
	manifest.Signature = sig
	return manifest, nil
}

// Verify checks that manifest was signed for book's current compiled
// state: book's content hash must match manifest.ContentHash (a mismatch
// means the lawbook changed since it was signed - tamper detection,
// docs/PLAN1.md §49), and manifest.Signature must verify against that
// hash.
func Verify(manifest Manifest, book *Book) error {
	hash, err := provenance.HashLawbook(book.lawbook)
	if err != nil {
		return err
	}
	if hash != manifest.ContentHash {
		return fmt.Errorf("lawbook content does not match the signed manifest (modified since signing)")
	}
	if manifest.Signature == "" {
		return fmt.Errorf("manifest has no signature")
	}
	return signing.Verify([]byte(manifest.ContentHash), manifest.Signature)
}
