// Package signing signs and verifies the content hash of a compiled
// lawbook using a self-contained Ed25519 keypair - no external gpg/ssh
// dependency (docs/PLAN1.md §25, §49).
package signing

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"strings"
)

// ErrVerificationFailed indicates a signature did not match the provided
// content hash - the tamper-detection case in PLAN1 §49.
var ErrVerificationFailed = errors.New("signing: verification failed")

const (
	privatePEMType = "AGENTLAWS PRIVATE KEY"
	publicPEMType  = "AGENTLAWS PUBLIC KEY"
	sigPrefix      = "ed25519:"
)

// GenerateKey creates a new Ed25519 keypair, writing the private key to
// path (mode 0600) and the public key to path + ".pub".
func GenerateKey(path string) error {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return err
	}
	privPEM := pem.EncodeToMemory(&pem.Block{Type: privatePEMType, Bytes: priv})
	if err := os.WriteFile(path, privPEM, 0600); err != nil {
		return err
	}
	pubPEM := pem.EncodeToMemory(&pem.Block{Type: publicPEMType, Bytes: pub})
	return os.WriteFile(path+".pub", pubPEM, 0644)
}

func loadPrivateKey(path string) (ed25519.PrivateKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(data)
	if block == nil || block.Type != privatePEMType {
		return nil, fmt.Errorf("signing: %s is not an AgentLaws private key", path)
	}
	if len(block.Bytes) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("signing: %s has an invalid key size", path)
	}
	return ed25519.PrivateKey(block.Bytes), nil
}

// Sign signs contentHash (a compiled lawbook's content hash, see
// internal/provenance.HashLawbook) with the private key at keyPath, and
// returns a self-describing signature string embedding the public key, so
// Verify needs no out-of-band key lookup.
func Sign(contentHash []byte, keyPath string) (string, error) {
	priv, err := loadPrivateKey(keyPath)
	if err != nil {
		return "", err
	}
	sig := ed25519.Sign(priv, contentHash)
	pub := priv.Public().(ed25519.PublicKey)
	return sigPrefix + base64.StdEncoding.EncodeToString(sig) + ":" + base64.StdEncoding.EncodeToString(pub), nil
}

// Verify checks that signature was produced for contentHash by the private
// key matching the public key embedded in signature.
func Verify(contentHash []byte, signature string) error {
	if !strings.HasPrefix(signature, sigPrefix) {
		return fmt.Errorf("signing: unrecognized signature format")
	}
	parts := strings.SplitN(strings.TrimPrefix(signature, sigPrefix), ":", 2)
	if len(parts) != 2 {
		return fmt.Errorf("signing: malformed signature")
	}
	sig, err := base64.StdEncoding.DecodeString(parts[0])
	if err != nil {
		return fmt.Errorf("signing: malformed signature: %w", err)
	}
	pub, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return fmt.Errorf("signing: malformed public key: %w", err)
	}
	if len(pub) != ed25519.PublicKeySize {
		return fmt.Errorf("signing: invalid public key size")
	}
	if !ed25519.Verify(ed25519.PublicKey(pub), contentHash, sig) {
		return ErrVerificationFailed
	}
	return nil
}
