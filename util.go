package main

import (
	"fmt"
	"html"
	"strings"
	"time"
	"unicode/utf8"
)

const Divider = "────────────"

func escHTML(s string) string { return html.EscapeString(s) }

func clampRunes(s string, max int) string {
	if max <= 0 || s == "" {
		return s
	}
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	r := []rune(s)
	return string(r[:max])
}

func formatTime(tsUnix int64) string {
	if tsUnix <= 0 {
		return ""
	}
	return time.Unix(tsUnix, 0).In(time.Local).Format("02.01.2006 15:04:05")
}

func emojiFor(kind string) string {
	switch kind {
	case "text":
		return "✍️"
	case "photo":
		return "📷"
	case "voice":
		return "🎤"
	default:
		return "💬"
	}
}

func headerHTML(chatTitle, sender string, tsUnix int64) string {
	ct := strings.TrimSpace(chatTitle)
	sn := strings.TrimSpace(sender)
	if ct == "" {
		ct = "Диалог"
	}
	if sn == "" {
		sn = "Unknown"
	}
	t := formatTime(tsUnix)
	if t == "" {
		return fmt.Sprintf("🗨️ <b>%s</b>\n👤 %s", escHTML(ct), escHTML(sn))
	}
	return fmt.Sprintf("🗨️ <b>%s</b>\n👤 %s\n🕒 <i>%s</i>", escHTML(ct), escHTML(sn), escHTML(t))
}

// Полный HTML-текст для TG-сообщения
func buildMessageHTML(chatTitle, sender string, tsUnix int64, kind string, text string) string {
	h := headerHTML(chatTitle, sender, tsUnix)
	em := emojiFor(kind)

	text = strings.TrimSpace(text)
	if text == "" {
		// даже если текста нет — покажем реакцию и подпись
		label := ""
		if kind == "photo" {
			label = " Фото"
		}
		return h + "\n" + Divider + "\n" + escHTML(em+label)
	}
	return h + "\n" + Divider + "\n" + escHTML(em+" "+text)
}

// Для caption под фото: можно добавить вторую строку с текстом
func buildPhotoCaptionHTML(chatTitle, sender string, tsUnix int64, text string) string {
	h := headerHTML(chatTitle, sender, tsUnix)
	text = strings.TrimSpace(text)

	if text == "" {
		return h + "\n" + Divider + "\n" + escHTML("📷 Фото")
	}
	// Фото + текст отдельно (чтобы красиво)
	return h + "\n" + Divider + "\n" + escHTML("📷 Фото") + "\n\n" + escHTML("✍️ "+text)
}
