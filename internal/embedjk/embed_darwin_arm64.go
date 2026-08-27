//go:build embedjk && darwin && arm64

package embedjk

import _ "embed"

//go:embed binaries/jk_darwin_arm64
var jkBinary []byte
