package bridge

import (
	"context"
	"strings"

	"github.com/iRelphy/vk2tg/internal/tg"
	"github.com/iRelphy/vk2tg/internal/util"
	"github.com/iRelphy/vk2tg/internal/vk"
)

// PhotoHandler tries to forward photo attachments from VK to Telegram.
// Right now we implement a simple strategy:
// - take first photo from VK message attachments
// - choose the biggest size URL
// - send it to Telegram with caption (header + "Photo" + optional message text)
type PhotoHandler struct {
	vk    *vk.Client
	bc    *tg.Broadcaster
	debug bool
}

type PhotoResult struct {
	Sent          bool // did we send a photo?
	UsedTextInCap bool // did we include message text inside the caption?
}

func NewPhotoHandler(vkClient *vk.Client, bc *tg.Broadcaster, debug bool) *PhotoHandler {
	return &PhotoHandler{vk: vkClient, bc: bc, debug: debug}
}

func (h *PhotoHandler) HandleVariantA(
	_ context.Context,
	chatTitle, sender string,
	tsUnix int64,
	text string,
	msg *vk.Message,
) (PhotoResult, error) {
	if msg == nil {
		return PhotoResult{}, nil
	}

	photoURLs := h.vk.ExtractPhotoURLs(msg)
	if len(photoURLs) == 0 {
		return PhotoResult{}, nil
	}

	text = strings.TrimSpace(text)

	// Пытаемся вставить текст в caption, но не терять его из-за лимита 1024
	captionWithText := buildPhotoCaptionHTML(chatTitle, sender, tsUnix, text)
	captionWithTextClamped := util.ClampRunes(captionWithText, 1024)

	usedText := false
	caption := captionWithTextClamped

	// Если текст был и caption “обрезался”, лучше НЕ включать текст в caption, а отправить его отдельно
	if text != "" && captionWithText != captionWithTextClamped {
		caption = util.ClampRunes(buildPhotoCaptionHTML(chatTitle, sender, tsUnix, ""), 1024)
		usedText = false
	} else if text != "" {
		usedText = true
	}

	// 1 фото -> обычная отправка
	if len(photoURLs) == 1 {
		if err := h.bc.SendPhotoURLHTML(photoURLs[0], caption); err != nil {
			return PhotoResult{}, err
		}
		return PhotoResult{Sent: true, UsedTextInCap: usedText}, nil
	}

	// 2+ фото -> альбом (media group)
	if err := h.bc.SendMediaGroupPhotosURLHTML(photoURLs, caption); err != nil {
		return PhotoResult{}, err
	}

	return PhotoResult{Sent: true, UsedTextInCap: usedText}, nil
}
