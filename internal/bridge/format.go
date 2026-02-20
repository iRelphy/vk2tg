package bridge

// This file contains formatting functions for Telegram messages.
// We generate HTML (Telegram parse mode = HTML) and ensure that all user text is escaped,
// so it cannot break formatting.

import (
	"fmt"
	"strings"

	"github.com/iRelphy/vk2tg/internal/util"
)

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

// headerHTML builds the message header:
// chat title, sender name, and time.
func headerHTML(chatTitle, sender string, tsUnix int64) string {
	ct := strings.TrimSpace(chatTitle)
	sn := strings.TrimSpace(sender)

	if ct == "" {
		ct = "Диалог"
	}
	if sn == "" {
		sn = "Unknown"
	}

	t := util.FormatTime(tsUnix)
	if t == "" {
		return fmt.Sprintf("🗨️ <b>%s</b>\n👤 %s", util.EscapeHTML(ct), util.EscapeHTML(sn))
	}

	return fmt.Sprintf(
		"🗨️ <b>%s</b>\n👤 %s\n🕒 <i>%s</i>",
		util.EscapeHTML(ct),
		util.EscapeHTML(sn),
		util.EscapeHTML(t),
	)
}

// buildMessageHTML is used for a normal Telegram text message.
func buildMessageHTML(chatTitle, sender string, tsUnix int64, kind string, text string) string {
	h := headerHTML(chatTitle, sender, tsUnix)
	em := emojiFor(kind)

	text = strings.TrimSpace(text)
	if text == "" {
		// Even without text show that something happened.
		label := ""
		if kind == "photo" {
			label = " Фото"
		}
		return h + "\n" + util.Divider + "\n" + util.EscapeHTML(em+label)
	}

	return h + "\n" + util.Divider + "\n" + util.EscapeHTML(em+" "+text)
}

// buildPhotoCaptionHTML is used as a caption under a photo.
func buildPhotoCaptionHTML(chatTitle, sender string, tsUnix int64, text string) string {
	h := headerHTML(chatTitle, sender, tsUnix)
	text = strings.TrimSpace(text)

	if text == "" {
		return h + "\n" + util.Divider + "\n" + util.EscapeHTML("📷 Фото")
	}

	// Show "Photo" and then the message text in a separate paragraph.
	return h + "\n" + util.Divider + "\n" + util.EscapeHTML("📷 Фото") + "\n\n" + util.EscapeHTML("✍️ "+text)
}
