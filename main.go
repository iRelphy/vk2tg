package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	longpoll "github.com/SevereCloud/vksdk/v3/longpoll-user"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func main() {
	LoadDotEnv()

	cfg, err := LoadConfig()
	if err != nil {
		log.Fatal(err)
	}

	// TG init
	tg, err := tgbotapi.NewBotAPI(cfg.TGToken)
	if err != nil {
		log.Fatalf("telegram init: %v", err)
	}
	// на всякий случай отключаем webhook, чтобы getUpdates не конфликтовал
	_, _ = tg.Request(tgbotapi.DeleteWebhookConfig{DropPendingUpdates: false})

	log.Printf("Telegram bot: @%s", tg.Self.UserName)

	// Subscribers store
	store := NewSubscriberStore(cfg.SubscribersFile)
	if err := store.Load(); err != nil {
		log.Fatalf("subscribers load: %v", err)
	}
	log.Printf("Subscribers loaded: %d", store.Count())

	bc := NewBroadcaster(tg, store, cfg.Debug)

	// VK init
	vkClient := NewVKClient(cfg.VKToken)
	vk := vkClient.VK

	names := NewNameResolver(vk, cfg.Debug)
	peers := NewPeerResolver(vk)

	photo := NewPhotoHandler(vkClient, bc, cfg.Debug)

	bridge := NewBridge(cfg, vkClient, names, peers, bc, photo)

	// LongPoll init
	mode := longpoll.ReceiveAttachments + longpoll.ExtendedEvents
	lp, err := longpoll.NewLongPoll(vk, mode)
	if err != nil {
		log.Fatalf("vk longpoll init: %v", err)
	}

	// Событие 4: новое сообщение
	lp.EventNew(4, bridge.HandleNewMessageEvent)

	// Context + signals
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		cmd := NewTGCommands(tg, store, cfg.Debug)
		cmd.Run(ctx)
	}()

	log.Printf("VK→TG started.")

	// graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		log.Printf("Stopping...")
		cancel()
		lp.Shutdown()
	}()

	// Run longpoll (blocking)
	if err := lp.Run(); err != nil {
		log.Fatalf("vk longpoll run: %v", err)
	}
}
