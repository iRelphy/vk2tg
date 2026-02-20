package util

// This package contains small helper functions that are used in several parts of the program.
// Think of it as a "toolbox": text trimming, safe HTML escaping, time formatting, etc.

import (
	"html"
	"unicode/utf8"
)

// Divider is used as a visual separator inside Telegram messages.
const Divider = "────────────"

// EscapeHTML makes text safe for Telegram HTML parse mode.
// Telegram supports a subset of HTML tags; if we do not escape user text,
// symbols like < and > could break formatting.
func EscapeHTML(s string) string { return html.EscapeString(s) }

// ClampRunes truncates a string to at most max Unicode characters (runes).
// Telegram has strict limits:
// - 4096 chars for message text
// - 1024 chars for photo caption
func ClampRunes(s string, max int) string {
	if max <= 0 || s == "" {
		return s
	}
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	r := []rune(s)
	return string(r[:max])
}
