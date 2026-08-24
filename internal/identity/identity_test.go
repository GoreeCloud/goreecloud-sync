package identity

import (
	"crypto/ed25519"
	"strings"
	"testing"
)

func TestNewKeyPairReturnsEd25519Material(t *testing.T) {
	publicKey, privateKey, err := NewKeyPair()
	if err != nil {
		t.Fatalf("NewKeyPair() error = %v", err)
	}
	if len(publicKey) != ed25519.PublicKeySize {
		t.Fatalf("public key length = %d, want %d", len(publicKey), ed25519.PublicKeySize)
	}
	if len(privateKey) != ed25519.PrivateKeySize {
		t.Fatalf("private key length = %d, want %d", len(privateKey), ed25519.PrivateKeySize)
	}
	if got := privateKey.Public().(ed25519.PublicKey); !got.Equal(publicKey) {
		t.Fatal("private key public component does not match returned public key")
	}
}

func TestFingerprintIsStableHexEncodedSHA256(t *testing.T) {
	publicKey := ed25519.PublicKey(strings.Repeat("\x01", ed25519.PublicKeySize))
	const want = "SHA256:72cd6e8422c407fb6d098690f1130b7ded7ec2f7f5e1d30bd9d521f015363793"

	if got := Fingerprint(publicKey); got != want {
		t.Fatalf("Fingerprint() = %q, want %q", got, want)
	}
}
