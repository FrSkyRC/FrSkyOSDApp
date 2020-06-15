package osdversion

import "fmt"

// Format returns a user-visible version string from
// the major, minor and patch components
func Format(major, minor, patch int) string {
	if minor == 99 {
		return fmt.Sprintf("%d.%d.%d-beta.%d", major+1, 0, 0, patch+1)
	}
	return fmt.Sprintf("%d.%d.%d", major, minor, patch)
}
