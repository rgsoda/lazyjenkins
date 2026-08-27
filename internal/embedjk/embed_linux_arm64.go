//go:build embedjk && linux && arm64

package embedjk

import _ "embed"

//go:embed binaries/jk_linux_arm64
var jkBinary []byte
