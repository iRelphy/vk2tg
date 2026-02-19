package main

import (
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Broadcaster struct {
	tg    *tgbotapi.BotAPI
	store *SubscriberStore
	debug bool
}

func NewBroadcaster(tg *tgbotapi.BotAPI, store *SubscriberStore, debug bool) *Broadcaster {
	return &Broadcaster{tg: tg, store: store, debug: debug}
}

func (b *Broadcaster) SendTextHTML(htmlText string) error {
	chatIDs := b.store.ChatIDs()
	for _, chatID := range chatIDs {
		msg := tgbotapi.NewMessage(chatID, clampRunes(htmlText, 4096))
		msg.ParseMode = tgbotapi.ModeHTML
		msg.DisableWebPagePreview = true

		if _, err := b.tg.Send(msg); err != nil {
			log.Printf("telegram send text error chat=%d: %v", chatID, err)
		}
	}
	return nil
}

func (b *Broadcaster) SendPhotoURLHTML(photoURL, captionHTML string) error {
	chatIDs := b.store.ChatIDs()
	for _, chatID := range chatIDs {
		p := tgbotapi.NewPhoto(chatID, tgbotapi.FileURL(photoURL))
		p.Caption = clampRunes(captionHTML, 1024)
		p.ParseMode = tgbotapi.ModeHTML

		if _, err := b.tg.Send(p); err != nil {
			log.Printf("telegram send photo error chat=%d: %v", chatID, err)
		}
	}
	return nil
}
