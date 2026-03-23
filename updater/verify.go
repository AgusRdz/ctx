package updater

import (
	"crypto/ed25519"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"io"
	"os"
	"strings"
)

// publicKeyPEM is the Ed25519 public key compiled into the binary.
// It is used to verify release signatures and cannot be tampered with
// at runtime — the key is the trust anchor for self-updates.
const publicKeyPEM = `-----BEGIN PUBLIC KEY-----
MCowBQYDK2VwAyEAg5vYSYU/3PpUttY1giCdutB+pX0e6hQWlGjNy8cFWJQ=
-----END PUBLIC KEY-----`

// releasePublicKey parses the embedded Ed25519 public key.
func releasePublicKey() (ed25519.PublicKey, error) {
	block, _ := pem.Decode([]byte(publicKeyPEM))
	if block == nil {
		return nil, fmt.Errorf("ctx: failed to decode public key PEM")
	}
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("ctx: failed to parse public key: %w", err)
	}
	key, ok := pub.(ed25519.PublicKey)
	if !ok {
		return nil, fmt.Errorf("ctx: public key is not Ed25519")
	}
	return key, nil
}

// verifySignature verifies that sig (base64-encoded) is a valid Ed25519 signature
// over message using the embedded public key.
func verifySignature(message []byte, sigB64 string) error {
	pub, err := releasePublicKey()
	if err != nil {
		return err
	}
	sig, err := base64.StdEncoding.DecodeString(strings.TrimSpace(sigB64))
	if err != nil {
		return fmt.Errorf("ctx: failed to decode signature: %w", err)
	}
	if !ed25519.Verify(pub, message, sig) {
		return fmt.Errorf("ctx: signature verification failed — checksums.txt may be tampered")
	}
	return nil
}

// verifyBinaryChecksum checks that the SHA256 of the file at binaryPath
// matches the entry in checksumsTxt for the given binaryName.
func verifyBinaryChecksum(binaryPath, binaryName string, checksumsTxt []byte) error {
	// Parse "sha256hex  filename\n" lines
	for _, line := range strings.Split(strings.TrimSpace(string(checksumsTxt)), "\n") {
		parts := strings.Fields(line)
		if len(parts) != 2 {
			continue
		}
		if parts[1] != binaryName {
			continue
		}
		expected := strings.ToLower(parts[0])
		actual, err := sha256File(binaryPath)
		if err != nil {
			return fmt.Errorf("ctx: failed to hash binary: %w", err)
		}
		if actual != expected {
			return fmt.Errorf("ctx: checksum mismatch — expected %s, got %s", expected, actual)
		}
		return nil
	}
	return fmt.Errorf("ctx: binary %q not found in checksums.txt", binaryName)
}

// sha256File returns the lowercase hex SHA256 of a file.
func sha256File(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
