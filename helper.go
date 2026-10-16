package main

import "strings"

func CleanInput(text string) (result []string) {
	if text == "" {
		return []string{""}
	}
	lower := strings.ToLower(text)
	// Fields splits the string by any type/amount of whitespace
	result = strings.Fields(lower)
	// // Join stitches them back together with a single space
	// trimmed := strings.Join()
	// result := strings.Split(trimmed, " ")
	if len(result) <= 0 {
		return []string{""}
	}
	return result
}
