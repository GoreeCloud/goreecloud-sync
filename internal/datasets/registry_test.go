package datasets

import "testing"

func TestFirstPartyRegistryIncludesCoreApplications(t *testing.T) {
	for _, app := range []string{"browser", "search", "bookmarks"} {
		if got := len(ForApplication(app)); got == 0 {
			t.Fatalf("expected capabilities for %s", app)
		}
	}
}

func TestNegotiateUsesMutualPermissionsAndLowestSchema(t *testing.T) {
	local := []Capability{{
		Dataset: "browser.tabs", Application: "browser", SchemaVersion: 2,
		Read: true, Write: true, Delete: true,
	}}
	remote := []Capability{{
		Dataset: "browser.tabs", Application: "browser", SchemaVersion: 1,
		Read: true, Write: false, Delete: true,
	}}

	got := Negotiate(local, remote)
	if len(got) != 1 {
		t.Fatalf("negotiated capability count = %d, want 1", len(got))
	}
	if got[0].SchemaVersion != 1 {
		t.Fatalf("schema version = %d, want 1", got[0].SchemaVersion)
	}
	if !got[0].Read || got[0].Write || !got[0].Delete {
		t.Fatalf("unexpected negotiated permissions: %+v", got[0])
	}
}

func TestNegotiateDropsUnknownDatasets(t *testing.T) {
	got := Negotiate(
		[]Capability{{Dataset: "bookmarks.items", SchemaVersion: 1, Read: true}},
		[]Capability{{Dataset: "search.history", SchemaVersion: 1, Read: true}},
	)
	if len(got) != 0 {
		t.Fatalf("negotiated capabilities = %+v, want none", got)
	}
}
