package module

import "strings"

func TextTokenizer(text string) []string {
	return strings.Fields(text)
}