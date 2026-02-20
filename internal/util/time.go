package util

import "time"

// FormatTime converts a VK Unix timestamp (seconds since 1970) into a readable local time string.
func FormatTime(tsUnix int64) string {
	if tsUnix <= 0 {
		return ""
	}
	return time.Unix(tsUnix, 0).In(time.Local).Format("02.01.2006 15:04:05")
}
