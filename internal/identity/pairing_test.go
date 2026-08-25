package identity

import "testing"

func TestPairingProofRoundTrip(t *testing.T) {
	publicKey, privateKey, err := NewKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	proof, err := NewPairingProof("11111111-1111-1111-1111-111111111111", "challenge", publicKey, privateKey)
	if err != nil {
		t.Fatal(err)
	}
	fingerprint, err := proof.Verify()
	if err != nil {
		t.Fatal(err)
	}
	if fingerprint != Fingerprint(publicKey) {
		t.Fatalf("fingerprint=%q", fingerprint)
	}
}

func TestPairingProofRejectsTampering(t *testing.T) {
	publicKey, privateKey, err := NewKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	proof, err := NewPairingProof("11111111-1111-1111-1111-111111111111", "challenge", publicKey, privateKey)
	if err != nil {
		t.Fatal(err)
	}
	proof.Nonce = "different"
	if _, err := proof.Verify(); err == nil {
		t.Fatal("expected tampered pairing proof rejection")
	}
}
