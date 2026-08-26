package datasets

import (
	"sort"
	"strings"
)

// Capability describes a first-party GoreeCloud dataset that can participate
// in Sync capability negotiation. Dataset identifiers are stable protocol
// names; applications may evolve their internal storage independently.
type Capability struct {
	Dataset       string `json:"dataset"`
	Application   string `json:"application"`
	SchemaVersion int    `json:"schemaVersion"`
	Read          bool   `json:"read"`
	Write         bool   `json:"write"`
	Delete        bool   `json:"delete"`
}

var firstParty = []Capability{
	{Dataset: "browser.tabs", Application: "browser", SchemaVersion: 1, Read: true, Write: true, Delete: true},
	{Dataset: "browser.history", Application: "browser", SchemaVersion: 1, Read: true, Write: true, Delete: true},
	{Dataset: "browser.preferences", Application: "browser", SchemaVersion: 1, Read: true, Write: true, Delete: false},
	{Dataset: "search.preferences", Application: "search", SchemaVersion: 1, Read: true, Write: true, Delete: false},
	{Dataset: "search.history", Application: "search", SchemaVersion: 1, Read: true, Write: true, Delete: true},
	{Dataset: "search.sources", Application: "search", SchemaVersion: 1, Read: true, Write: true, Delete: false},
	{Dataset: "bookmarks.items", Application: "bookmarks", SchemaVersion: 1, Read: true, Write: true, Delete: true},
	{Dataset: "bookmarks.collections", Application: "bookmarks", SchemaVersion: 1, Read: true, Write: true, Delete: true},
	{Dataset: "bookmarks.assignments", Application: "bookmarks", SchemaVersion: 1, Read: true, Write: true, Delete: true},
}

// FirstParty returns an ordered copy of the canonical first-party registry.
func FirstParty() []Capability {
	out := append([]Capability(nil), firstParty...)
	sort.Slice(out, func(i, j int) bool { return out[i].Dataset < out[j].Dataset })
	return out
}

// ForApplication returns capabilities advertised by a first-party application.
func ForApplication(application string) []Capability {
	application = strings.TrimSpace(strings.ToLower(application))
	if application == "" {
		return nil
	}
	var out []Capability
	for _, capability := range FirstParty() {
		if capability.Application == application {
			out = append(out, capability)
		}
	}
	return out
}

// Negotiate returns the mutually supported datasets. The lower schema version
// wins so peers can communicate at the newest version they both understand.
func Negotiate(local, remote []Capability) []Capability {
	remoteByDataset := make(map[string]Capability, len(remote))
	for _, capability := range remote {
		remoteByDataset[capability.Dataset] = capability
	}
	var out []Capability
	for _, left := range local {
		right, ok := remoteByDataset[left.Dataset]
		if !ok || left.SchemaVersion < 1 || right.SchemaVersion < 1 {
			continue
		}
		version := left.SchemaVersion
		if right.SchemaVersion < version {
			version = right.SchemaVersion
		}
		capability := Capability{
			Dataset:       left.Dataset,
			Application:   left.Application,
			SchemaVersion: version,
			Read:          left.Read && right.Read,
			Write:         left.Write && right.Write,
			Delete:        left.Delete && right.Delete,
		}
		if capability.Read || capability.Write || capability.Delete {
			out = append(out, capability)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Dataset < out[j].Dataset })
	return out
}
