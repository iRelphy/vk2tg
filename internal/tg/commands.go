package tg

import (
	"context"
	"log"
	"strings"

	"github.com/iRelphy/vk2tg/internal/storage"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Commands listens for Telegram updates and reacts to bot commands.
// We currently support:
// - /start : subscribe this chat
// - /stop  : unsubscribe this chat
type Commands struct {
	bot   *tgbotapi.BotAPI
	store *storage.SubscriberStore
	debug bool
}

func NewCommands(bot *tgbotapi.BotAPI, store *storage.SubscriberStore, debug bool) *Commands {
	return &Commands{bot: bot, store: store, debug: debug}
}

// Run blocks forever until ctx is cancelled.
func (c *Commands) Run(ctx context.Context) {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 30

	updates := c.bot.GetUpdatesChan(u)
	log.Printf("[TG] command loop started")

	for {
		select {
		case <-ctx.Done():
			return

		case upd := <-updates:
			if upd.Message == nil {
				continue
			}

			text := strings.TrimSpace(upd.Message.Text)
			if text == "" {
				continue
			}

			switch {
			case strings.HasPrefix(text, "/start"):
				sub := storage.Subscriber{
					ChatID:    upd.Message.Chat.ID,
					Username:  upd.Message.From.UserName,
					FirstName: upd.Message.From.FirstName,
					LastName:  upd.Message.From.LastName,
				}
				changed, _ := c.store.Add(sub)

				reply := "✅ Подписал! Теперь буду присылать сообщения из ВК сюда.\n\n" +
					"ℹ️ Если хочешь остановить — напиши /stop"
				if !changed {
					reply = "✅ Ты уже подписан(а).\n\nℹ️ Остановить — /stop"
				}

				msg := tgbotapi.NewMessage(upd.Message.Chat.ID, reply)
				_, _ = c.bot.Send(msg)

			case strings.HasPrefix(text, "/stop"):
				changed, _ := c.store.Remove(upd.Message.Chat.ID)

				reply := "🛑 Ок, отключил."
				if !changed {
					reply = "ℹ️ Ты и так не был(а) подписан(а)."
				}

				msg := tgbotapi.NewMessage(upd.Message.Chat.ID, reply)
				_, _ = c.bot.Send(msg)
			}
		}
	}
}
