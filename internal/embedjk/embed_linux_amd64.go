//go:build embedjk && linux && amd64

package embedjk

import _ "embed"

//go:embed binaries/jk_linux_amd64
var jkBinary []byte
