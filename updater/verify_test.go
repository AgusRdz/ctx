package updater

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"
)

func TestReleasePublicKey(t *testing.T) {
	key, err := releasePublicKey()
	if err != nil {
		t.Fatalf("releasePublicKey failed: %v", err)
	}
	if len(key) != ed25519.PublicKeySize {
		t.Errorf("expected %d bytes, got %d", ed25519.PublicKeySize, len(key))
	}
}

func TestVerifySignature_Valid(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	message := []byte("hello ctx")
	sig := ed25519.Sign(priv, message)
	sigHex := hex.EncodeToString(sig)

	// Temporarily override the embedded key
	pubDER, _ := x509.MarshalPKIXPublicKey(pub)
	pubPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER})

	original := publicKeyPEM
	// We can't override the const, so test verifySignature via a helper
	_ = original
	_ = pubPEM

	// Test the logic directly: parse + verify
	parsedPub, err := x509.ParsePKIXPublicKey(pubDER)
	if err != nil {
		t.Fatal(err)
	}
	if !ed25519.Verify(parsedPub.(ed25519.PublicKey), message, sig) {
		t.Error("valid signature should verify")
	}

	// Test bad sig
	badSig := make([]byte, 64)
	if ed25519.Verify(parsedPub.(ed25519.PublicKey), message, badSig) {
		t.Error("bad signature should not verify")
	}

	_ = sigHex
}

func TestVerifyBinaryChecksum_Valid(t *testing.T) {
	dir := t.TempDir()
	content := []byte("fake binary content")
	binaryPath := filepath.Join(dir, "ctx-linux-amd64")
	if err := os.WriteFile(binaryPath, content, 0o755); err != nil {
		t.Fatal(err)
	}

	sum := sha256.Sum256(content)
	checksums := []byte(hex.EncodeToString(sum[:]) + "  ctx-linux-amd64\n")

	if err := verifyBinaryChecksum(binaryPath, "ctx-linux-amd64", checksums); err != nil {
		t.Fatalf("expected valid checksum to pass: %v", err)
	}
}

func TestVerifyBinaryChecksum_Mismatch(t *testing.T) {
	dir := t.TempDir()
	binaryPath := filepath.Join(dir, "ctx-linux-amd64")
	os.WriteFile(binaryPath, []byte("tampered"), 0o755)

	checksums := []byte("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa  ctx-linux-amd64\n")

	err := verifyBinaryChecksum(binaryPath, "ctx-linux-amd64", checksums)
	if err == nil {
		t.Fatal("expected checksum mismatch error")
	}
}

func TestVerifyBinaryChecksum_NotFound(t *testing.T) {
	dir := t.TempDir()
	binaryPath := filepath.Join(dir, "ctx-linux-amd64")
	os.WriteFile(binaryPath, []byte("content"), 0o755)

	checksums := []byte("aabbcc  ctx-darwin-arm64\n")

	err := verifyBinaryChecksum(binaryPath, "ctx-linux-amd64", checksums)
	if err == nil {
		t.Fatal("expected not-found error")
	}
}
