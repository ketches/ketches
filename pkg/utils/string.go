package utils

import "strings"

func FormatHost(host string) string {
	return strings.ReplaceAll(host, ".", "-")
}
