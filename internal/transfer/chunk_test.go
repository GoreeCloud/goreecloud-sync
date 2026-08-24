package transfer

import (
	"strings"
	"testing"
)

func TestHashReturnsHexEncodedSHA256(t *testing.T) {
	const payload = "goreecloud-sync"
	const want = "d93276e24180b37be4cf6e77e46743abed13cb538d6cfccfe1289e7f0d7792ed"

	if got := Hash([]byte(payload)); got != want {
		t.Fatalf("Hash() = %q, want %q", got, want)
	}
}

func TestHashReaderMatchesChunkHash(t *testing.T) {
	const payload = "integrity-check"

	got, err := HashReader(strings.NewReader(payload))
	if err != nil {
		t.Fatalf("HashReader() error = %v", err)
	}
	if want := Hash([]byte(payload)); got != want {
		t.Fatalf("HashReader() = %q, want %q", got, want)
	}
}
