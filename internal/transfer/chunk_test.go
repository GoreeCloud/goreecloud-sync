package transfer

import (
	"strings"
	"testing"
)

func TestHashReturnsHexEncodedSHA256(t *testing.T) {
	const payload = "goreecloud-sync"
	const want = "606f215ec0bc7d4cec2500e9fe5b8edd923efd690a9f82996b10b6451efd7bb5"

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
