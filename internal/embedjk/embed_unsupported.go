//go:build embedjk && !((darwin && arm64) || (darwin && amd64) || (linux && arm64) || (linux && amd64))

package embedjk

// jkBinary stays nil: we don't bundle jk for this platform. Extract()
// (embed_embedded.go) reports that cleanly rather than failing to compile.
var jkBinary []byte
