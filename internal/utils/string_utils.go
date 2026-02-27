package utils

import "strings"

// TruncateString truncates a string to maxLen characters
func TruncateString(s string, maxLen int) string {
    if len(s) <= maxLen {
        return s
    }
    return s[:maxLen] + "..."
}

// ContainsAny returns true if s contains any of the substrings
func ContainsAny(s string, substrings []string) bool {
    for _, sub := range substrings {
        if strings.Contains(s, sub) {
            return true
        }
    }
    return false
}
