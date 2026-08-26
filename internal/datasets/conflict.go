package datasets

import "strings"

// Resolution describes which record wins deterministic replication conflict
// handling. Identical records are treated as converged rather than conflicts.
type Resolution string

const (
	ResolutionLocal     Resolution = "local"
	ResolutionRemote    Resolution = "remote"
	ResolutionConverged Resolution = "converged"
)

// ResolveConflict deterministically selects one record for the same dataset and
// record identifier. Higher revision wins, then newer timestamp, then tombstone,
// then lexicographically greater origin device as a stable final tie-breaker.
func ResolveConflict(local, remote RecordEnvelope) Resolution {
	if local.Revision != remote.Revision {
		if remote.Revision > local.Revision {
			return ResolutionRemote
		}
		return ResolutionLocal
	}
	if !local.UpdatedAt.Equal(remote.UpdatedAt) {
		if remote.UpdatedAt.After(local.UpdatedAt) {
			return ResolutionRemote
		}
		return ResolutionLocal
	}
	if local.Deleted != remote.Deleted {
		if remote.Deleted {
			return ResolutionRemote
		}
		return ResolutionLocal
	}
	left := strings.TrimSpace(local.OriginDevice)
	right := strings.TrimSpace(remote.OriginDevice)
	if left != right {
		if right > left {
			return ResolutionRemote
		}
		return ResolutionLocal
	}
	return ResolutionConverged
}
