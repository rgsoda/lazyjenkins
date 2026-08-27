//go:build embedjk && darwin && amd64

package embedjk

import _ "embed"

//go:embed binaries/jk_darwin_amd64
var jkBinary []byte
