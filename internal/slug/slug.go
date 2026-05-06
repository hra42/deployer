package slug

import "strings"

// Domain returns a Docker-safe slug derived from a domain name:
// trimmed, lowercased, with dots replaced by dashes.
func Domain(d string) string {
	return strings.ReplaceAll(strings.ToLower(strings.TrimSpace(d)), ".", "-")
}
