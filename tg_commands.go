package main

import (
	"context"
	"log"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type TGCommands struct {
	tg    *tgbotapi.BotAPI
	store *SubscriberStore
	debug bool
}

func NewTGCommands(tg *tgbotapi.BotAPI, store *SubscriberStore, debug bool) *TGCommands {
	return &TGCommands{tg: tg, store: store, debug: debug}
}

func (c *TGCommands) Run(ctx context.Context) {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 30

	updates := c.tg.GetUpdatesChan(u)
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
				sub := Subscriber{
					ChatID:    upd.Message.Chat.ID,
					Username:  upd.Message.From.UserName,
					FirstName: upd.Message.From.FirstName,
					LastName:  upd.Message.From.LastName,
				}
				changed, _ := c.store.Add(sub)
				reply := "✅ Подписал! Теперь буду присылать сообщения из ВК сюда.\n\n" +
					"ℹ️ Если ты хочешь остановить — напиши /stop"
				if !changed {
					reply = "✅ Ты уже подписан(а).\n\nℹ️ Остановить — /stop"
				}
				msg := tgbotapi.NewMessage(upd.Message.Chat.ID, reply)
				_, _ = c.tg.Send(msg)

			case strings.HasPrefix(text, "/stop"):
				changed, _ := c.store.Remove(upd.Message.Chat.ID)
				reply := "🛑 Ок, отключил."
				if !changed {
					reply = "ℹ️ Ты и так не был(а) подписан(а)."
				}
				msg := tgbotapi.NewMessage(upd.Message.Chat.ID, reply)
				_, _ = c.tg.Send(msg)
			}
		}
	}
}
