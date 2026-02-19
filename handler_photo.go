package main

import (
	"context"
	"log"
	"strings"
)

type PhotoHandler struct {
	vk    *VKClient
	bc    *Broadcaster
	debug bool
}

type PhotoResult struct {
	Sent          bool
	UsedTextInCap bool
}

func NewPhotoHandler(vk *VKClient, bc *Broadcaster, debug bool) *PhotoHandler {
	return &PhotoHandler{vk: vk, bc: bc, debug: debug}
}

// Вариант А: взять ссылку на фото прямо из сообщения (attachments->photo->sizes->url)
func (h *PhotoHandler) HandleVariantA(ctx context.Context, chatTitle, sender string, tsUnix int64, text string, msg *VKMessage) (PhotoResult, error) {
	_ = ctx

	if msg == nil {
		return PhotoResult{}, nil
	}

	photoURL := h.vk.ExtractFirstPhotoURL(msg)
	if photoURL == "" {
		return PhotoResult{}, nil
	}

	caption := buildPhotoCaptionHTML(chatTitle, sender, tsUnix, text)
	// caption ограничение TG
	caption = clampRunes(caption, 1024)

	if err := h.bc.SendPhotoURLHTML(photoURL, caption); err != nil {
		return PhotoResult{}, err
	}

	usedText := strings.TrimSpace(text) != ""
	if h.debug {
		log.Printf("[PHOTO] sent url=%s used_text=%v", photoURL, usedText)
	}
	return PhotoResult{Sent: true, UsedTextInCap: usedText}, nil
}
