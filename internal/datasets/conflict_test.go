package datasets

import (
	"testing"
	"time"
)

func TestResolveConflictPrefersHigherRevision(t *testing.T) {
	when := time.Unix(100, 0).UTC()
	local := RecordEnvelope{Revision: 2, UpdatedAt: when, OriginDevice: "device-a"}
	remote := RecordEnvelope{Revision: 3, UpdatedAt: when.Add(-time.Hour), OriginDevice: "device-b"}
	if got := ResolveConflict(local, remote); got != ResolutionRemote {
		t.Fatalf("resolution = %q, want remote", got)
	}
}

func TestResolveConflictPrefersNewerTimestampThenTombstone(t *testing.T) {
	when := time.Unix(100, 0).UTC()
	local := RecordEnvelope{Revision: 3, UpdatedAt: when, OriginDevice: "device-a"}
	remote := RecordEnvelope{Revision: 3, UpdatedAt: when.Add(time.Second), OriginDevice: "device-b"}
	if got := ResolveConflict(local, remote); got != ResolutionRemote {
		t.Fatalf("timestamp resolution = %q, want remote", got)
	}
	remote.UpdatedAt = when
	remote.Deleted = true
	if got := ResolveConflict(local, remote); got != ResolutionRemote {
		t.Fatalf("tombstone resolution = %q, want remote", got)
	}
}

func TestResolveConflictUsesStableDeviceTieBreaker(t *testing.T) {
	when := time.Unix(100, 0).UTC()
	local := RecordEnvelope{Revision: 3, UpdatedAt: when, OriginDevice: "device-a"}
	remote := RecordEnvelope{Revision: 3, UpdatedAt: when, OriginDevice: "device-b"}
	if got := ResolveConflict(local, remote); got != ResolutionRemote {
		t.Fatalf("device resolution = %q, want remote", got)
	}
	remote.OriginDevice = local.OriginDevice
	if got := ResolveConflict(local, remote); got != ResolutionConverged {
		t.Fatalf("identical metadata resolution = %q, want converged", got)
	}
}
