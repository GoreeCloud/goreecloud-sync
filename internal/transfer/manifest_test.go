package transfer

import (
	"bytes"
	"strings"
	"testing"
)

func TestBuildManifestAndVerifyPayload(t *testing.T) {
	payload := bytes.Repeat([]byte("goreecloud-sync"), (2*DefaultChunkSize/len("goreecloud-sync"))+37)

	manifest, err := BuildManifest("example.bin", bytes.NewReader(payload), DefaultChunkSize)
	if err != nil {
		t.Fatalf("BuildManifest() error = %v", err)
	}
	if manifest.Version != ManifestVersion {
		t.Fatalf("manifest version = %d, want %d", manifest.Version, ManifestVersion)
	}
	if manifest.Size != int64(len(payload)) {
		t.Fatalf("manifest size = %d, want %d", manifest.Size, len(payload))
	}
	if len(manifest.Chunks) != 3 {
		t.Fatalf("chunk count = %d, want 3", len(manifest.Chunks))
	}
	if manifest.Chunks[0].Size != DefaultChunkSize || manifest.Chunks[1].Size != DefaultChunkSize {
		t.Fatalf("full chunks must be %d bytes", DefaultChunkSize)
	}
	if manifest.Chunks[2].Size <= 0 || manifest.Chunks[2].Size >= DefaultChunkSize {
		t.Fatalf("final chunk size = %d, want partial chunk", manifest.Chunks[2].Size)
	}
	if err := VerifyPayload(manifest, bytes.NewReader(payload)); err != nil {
		t.Fatalf("VerifyPayload() error = %v", err)
	}
}

func TestVerifyPayloadRejectsCorruption(t *testing.T) {
	payload := bytes.Repeat([]byte("a"), DefaultChunkSize+128)
	manifest, err := BuildManifest("corrupt.bin", bytes.NewReader(payload), DefaultChunkSize)
	if err != nil {
		t.Fatalf("BuildManifest() error = %v", err)
	}

	corrupt := append([]byte(nil), payload...)
	corrupt[DefaultChunkSize+5] ^= 0xff
	if err := VerifyPayload(manifest, bytes.NewReader(corrupt)); err == nil || !strings.Contains(err.Error(), "integrity verification failed") {
		t.Fatalf("VerifyPayload() error = %v, want integrity failure", err)
	}
}

func TestVerifyPayloadRejectsTruncationAndTrailingData(t *testing.T) {
	payload := bytes.Repeat([]byte("b"), DefaultChunkSize+17)
	manifest, err := BuildManifest("boundary.bin", bytes.NewReader(payload), DefaultChunkSize)
	if err != nil {
		t.Fatalf("BuildManifest() error = %v", err)
	}

	if err := VerifyPayload(manifest, bytes.NewReader(payload[:len(payload)-1])); err == nil {
		t.Fatal("VerifyPayload() accepted truncated payload")
	}

	withTrailing := append(append([]byte(nil), payload...), 0x01)
	if err := VerifyPayload(manifest, bytes.NewReader(withTrailing)); err == nil || !strings.Contains(err.Error(), "trailing data") {
		t.Fatalf("VerifyPayload() error = %v, want trailing-data rejection", err)
	}
}

func TestManifestValidateRejectsMalformedMetadata(t *testing.T) {
	payload := bytes.Repeat([]byte("c"), DefaultChunkSize+10)
	manifest, err := BuildManifest("metadata.bin", bytes.NewReader(payload), DefaultChunkSize)
	if err != nil {
		t.Fatalf("BuildManifest() error = %v", err)
	}

	tests := []struct {
		name   string
		mutate func(*Manifest)
	}{
		{
			name: "version",
			mutate: func(m *Manifest) {
				m.Version++
			},
		},
		{
			name: "uppercase digest",
			mutate: func(m *Manifest) {
				m.Hash = strings.ToUpper(m.Hash)
			},
		},
		{
			name: "chunk index",
			mutate: func(m *Manifest) {
				m.Chunks[1].Index = 7
			},
		},
		{
			name: "non-final short chunk",
			mutate: func(m *Manifest) {
				m.Chunks[0].Size--
			},
		},
		{
			name: "declared size",
			mutate: func(m *Manifest) {
				m.Size++
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			candidate := manifest
			candidate.Chunks = append([]Chunk(nil), manifest.Chunks...)
			tt.mutate(&candidate)
			if err := candidate.Validate(); err == nil {
				t.Fatalf("Validate() accepted malformed %s manifest", tt.name)
			}
		})
	}
}

func TestEmptyPayloadManifest(t *testing.T) {
	manifest, err := BuildManifest("empty.txt", bytes.NewReader(nil), DefaultChunkSize)
	if err != nil {
		t.Fatalf("BuildManifest() error = %v", err)
	}
	if manifest.Size != 0 || len(manifest.Chunks) != 0 {
		t.Fatalf("empty manifest = size %d chunks %d", manifest.Size, len(manifest.Chunks))
	}
	if manifest.Hash != Hash(nil) {
		t.Fatalf("empty manifest hash = %q, want %q", manifest.Hash, Hash(nil))
	}
	if err := VerifyPayload(manifest, bytes.NewReader(nil)); err != nil {
		t.Fatalf("VerifyPayload() error = %v", err)
	}
}

func TestBuildManifestRejectsUnboundedChunkSize(t *testing.T) {
	if _, err := BuildManifest("large.bin", bytes.NewReader([]byte("data")), DefaultChunkSize+1); err == nil {
		t.Fatal("BuildManifest() accepted chunk size above current bound")
	}
}
