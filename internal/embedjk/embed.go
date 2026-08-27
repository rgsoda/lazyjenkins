// Package embedjk bundles the jk binary into release builds of lazyjenkins,
// so installing lazyjenkins doesn't require separately installing jk.
//
// The actual embedding only happens in builds compiled with the "embedjk"
// build tag (see embed_embedded.go) — that's what goreleaser's release
// builds use, after its pre-build hook fetches jk's release binaries into
// binaries/. Plain local builds (`go build .`, no tags) use embed_stub.go
// instead, so hacking on lazyjenkins never requires fetching or keeping
// those binaries around.
package embedjk

// Version is the pinned jk release bundled into lazyjenkins. Bump this and
// scripts/fetch-jk.sh together when updating.
const Version = "0.0.36"
