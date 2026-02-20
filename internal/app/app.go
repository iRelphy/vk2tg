package app

// Package app wires all parts together and runs the whole program.
//
// Why have this package?
// - cmd/vk2tg/main.go stays short and easy to read
// - all "create client / create store / start loops" logic is in one place

import (
	"context"
	"fmt"
	"log"

	longpoll "github.com/SevereCloud/vksdk/v3/longpoll-user"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/iRelphy/vk2tg/internal/bridge"
	"github.com/iRelphy/vk2tg/internal/config"
	"github.com/iRelphy/vk2tg/internal/storage"
	"github.com/iRelphy/vk2tg/internal/tg"
	"github.com/iRelphy/vk2tg/internal/vk"
)

// App is a configured instance of vk2tg.
type App struct {
	cfg config.Config

	bot   *tgbotapi.BotAPI
	store *storage.SubscriberStore

	broadcaster *tg.Broadcaster
	commands    *tg.Commands

	vkClient *vk.Client
	names    *vk.NameResolver
	peers    *vk.PeerResolver

	bridge *bridge.Bridge
	lp     *longpoll.LongPoll
}

// New creates and configures the app, but does not start background loops yet.
func New(cfg config.Config) (*App, error) {
	// --- Telegram ---
	bot, err := tgbotapi.NewBotAPI(cfg.TGToken)
	if err != nil {
		return nil, fmt.Errorf("telegram init: %w", err)
	}
	// We use polling (GetUpdates), so webhook must be disabled.
	_, _ = bot.Request(tgbotapi.DeleteWebhookConfig{DropPendingUpdates: false})
	log.Printf("Telegram bot: @%s", bot.Self.UserName)

	store := storage.NewSubscriberStore(cfg.SubscribersFile)
	if err := store.Load(); err != nil {
		return nil, fmt.Errorf("subscribers load: %w", err)
	}
	log.Printf("Subscribers loaded: %d", store.Count())

	bc := tg.NewBroadcaster(bot, store, cfg.Debug)
	cmd := tg.NewCommands(bot, store, cfg.Debug)

	// --- VK ---
	vkClient := vk.NewClient(cfg.VKToken)
	names := vk.NewNameResolver(vkClient.API, cfg.Debug)
	peers := vk.NewPeerResolver(vkClient.API, cfg.Debug)

	photo := bridge.NewPhotoHandler(vkClient, bc, cfg.Debug)
	br := bridge.New(cfg, vkClient, names, peers, bc, photo)

	// --- VK LongPoll ---
	mode := longpoll.ReceiveAttachments + longpoll.ExtendedEvents
	lp, err := longpoll.NewLongPoll(vkClient.API, mode)
	if err != nil {
		return nil, fmt.Errorf("vk longpoll init: %w", err)
	}
	lp.EventNew(4, br.HandleNewMessageEvent)

	return &App{
		cfg:         cfg,
		bot:         bot,
		store:       store,
		broadcaster: bc,
		commands:    cmd,
		vkClient:    vkClient,
		names:       names,
		peers:       peers,
		bridge:      br,
		lp:          lp,
	}, nil
}

// Run starts background loops and blocks until LongPoll stops.
// To stop the app, cancel the context (ctx.Done()).
func (a *App) Run(ctx context.Context) error {
	// Telegram commands loop (runs in background).
	go a.commands.Run(ctx)

	// Stop VK LongPoll when context is cancelled.
	go func() {
		<-ctx.Done()
		a.lp.Shutdown()
	}()

	log.Printf("VK→TG started.")
	return a.lp.Run()
}
