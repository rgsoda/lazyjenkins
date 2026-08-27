//go:build !embedjk

package embedjk

import "errors"

// Extract reports that this build has no bundled jk (plain local/dev
// builds, or a release target we don't bundle for). Callers should fall
// back to a PATH-installed jk.
func Extract() (string, error) {
	return "", errors.New("this build has no bundled jk")
}
