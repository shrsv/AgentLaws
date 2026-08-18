package signing

import (
	"os"
	"testing"
)

func TestSignVerifyRoundTrip(t *testing.T) {
	dir := t.TempDir()
	keyPath := dir + "/testkey"

	if err := GenerateKey(keyPath); err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}

	content := []byte("test content hash to sign")
	sig, err := Sign(content, keyPath)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	if sig == "" {
		t.Fatal("Sign returned empty signature")
	}

	if err := Verify(content, sig); err != nil {
		t.Fatalf("Verify: %v", err)
	}
}

func TestVerifyDetectsTampering(t *testing.T) {
	dir := t.TempDir()
	keyPath := dir + "/testkey"

	if err := GenerateKey(keyPath); err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}

	content := []byte("original content")
	sig, err := Sign(content, keyPath)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}

	tampered := []byte("original contexx")
	if err := Verify(tampered, sig); err != ErrVerificationFailed {
		t.Fatalf("expected ErrVerificationFailed for tampered content, got: %v", err)
	}
}

func TestVerifyRejectsMalformedSignature(t *testing.T) {
	content := []byte("test")
	if err := Verify(content, "not-a-valid-sig"); err == nil {
		t.Fatal("expected error for malformed signature")
	}
}

func TestGenerateKeyCreatesFiles(t *testing.T) {
	dir := t.TempDir()
	keyPath := dir + "/mykey"

	if err := GenerateKey(keyPath); err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}

	if _, err := os.ReadFile(keyPath); err != nil {
		t.Fatalf("private key file missing: %v", err)
	}
	if _, err := os.ReadFile(keyPath + ".pub"); err != nil {
		t.Fatalf("public key file missing: %v", err)
	}
}
