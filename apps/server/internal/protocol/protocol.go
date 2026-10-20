// Package protocol holds the sync wire contract shared with the desktop client.
//
// The authoritative definition lives in the desktop repository half, at
// apps/desktop/src/common/sync-protocol.ts. Both sides are pinned together by
// the shared fixtures under packages/sync-fixtures (work item #9); when the two
// disagree, that is a bug in one of them, not a negotiation.
package protocol

// Supported wire versions, advertised by GET /health as protocol.min/max.
// The desktop sends its version in X-Croffle-Sync-Version and checks this range
// once on connect, so an incompatible pair is reported in the settings screen
// instead of failing later during a background sync.
const (
	Min = 1
	Max = 1
)

// Header names the desktop transport sets on every sync request.
const (
	HeaderVersion  = "X-Croffle-Sync-Version"
	HeaderClientID = "X-Croffle-Client"
)
