package tg

// Package tg contains Telegram-related code:
// - sending messages/photos to subscribers
// - handling Telegram commands (/start, /stop)
//
// We keep Telegram-specific details here, so VK/bridge code stays clean.

import (
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/iRelphy/vk2tg/internal/storage"
	"github.com/iRelphy/vk2tg/internal/util"
)

// Broadcaster sends prepared messages to all subscribed Telegram chats.
type Broadcaster struct {
	bot   *tgbotapi.BotAPI
	store *storage.SubscriberStore
	debug bool
}

func NewBroadcaster(bot *tgbotapi.BotAPI, store *storage.SubscriberStore, debug bool) *Broadcaster {
	return &Broadcaster{bot: bot, store: store, debug: debug}
}

// SendTextHTML sends one HTML message to every subscriber chat.
func (b *Broadcaster) SendTextHTML(htmlText string) error {
	chatIDs := b.store.ChatIDs()

	for _, chatID := range chatIDs {
		msg := tgbotapi.NewMessage(chatID, util.ClampRunes(htmlText, 4096))
		msg.ParseMode = tgbotapi.ModeHTML
		msg.DisableWebPagePreview = true

		if _, err := b.bot.Send(msg); err != nil {
			log.Printf("telegram send text error chat=%d: %v", chatID, err)
		}
	}

	return nil
}

// SendPhotoURLHTML sends a photo by URL with HTML caption to every subscriber chat.
func (b *Broadcaster) SendPhotoURLHTML(photoURL, captionHTML string) error {
	chatIDs := b.store.ChatIDs()

	for _, chatID := range chatIDs {
		p := tgbotapi.NewPhoto(chatID, tgbotapi.FileURL(photoURL))
		p.Caption = util.ClampRunes(captionHTML, 1024)
		p.ParseMode = tgbotapi.ModeHTML

		if _, err := b.bot.Send(p); err != nil {
			log.Printf("telegram send photo error chat=%d: %v", chatID, err)
		}
	}

	return nil
}

// SendMediaGroupPhotosURLHTML отправляет альбом (media group) из фото URL всем подписчикам.
// Telegram ограничения:
// - 2..10 медиа в одной группе
// - caption только у первого элемента, лимит 1024 символа
func (b *Broadcaster) SendMediaGroupPhotosURLHTML(photoURLs []string, captionHTML string) error {
	if len(photoURLs) == 0 {
		return nil
	}

	// TG принимает максимум 10 элементов в группе
	const batchSize = 10
	captionHTML = util.ClampRunes(captionHTML, 1024)

	chatIDs := b.store.ChatIDs()
	for _, chatID := range chatIDs {
		for batchStart := 0; batchStart < len(photoURLs); batchStart += batchSize {
			end := batchStart + batchSize
			if end > len(photoURLs) {
				end = len(photoURLs)
			}
			batch := photoURLs[batchStart:end]

			media := make([]interface{}, 0, len(batch))
			for i, url := range batch {
				im := tgbotapi.NewInputMediaPhoto(tgbotapi.FileURL(url))

				// caption только на первом элементе первой пачки
				if batchStart == 0 && i == 0 && captionHTML != "" {
					im.Caption = captionHTML
					im.ParseMode = tgbotapi.ModeHTML
				}

				media = append(media, im)
			}

			cfg := tgbotapi.NewMediaGroup(chatID, media)
			if _, err := b.bot.SendMediaGroup(cfg); err != nil {
				log.Printf("telegram send media group error chat=%d: %v", chatID, err)
			}
		}
	}

	return nil
}
