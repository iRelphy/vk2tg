package main

import (
	"context"
	"fmt"
	"log"
	"strings"
)

type Bridge struct {
	cfg   Config
	vk    *VKClient
	names *NameResolver
	peers *PeerResolver
	bc    *Broadcaster
	photo *PhotoHandler
	debug bool
}

func NewBridge(cfg Config, vk *VKClient, names *NameResolver, peers *PeerResolver, bc *Broadcaster, photo *PhotoHandler) *Bridge {
	return &Bridge{
		cfg:   cfg,
		vk:    vk,
		names: names,
		peers: peers,
		bc:    bc,
		photo: photo,
		debug: cfg.Debug,
	}
}

func asInt(ev []interface{}, idx int) int {
	if idx < 0 || idx >= len(ev) {
		return 0
	}
	switch v := ev[idx].(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	default:
		return 0
	}
}

func asString(ev []interface{}, idx int) string {
	if idx < 0 || idx >= len(ev) {
		return ""
	}
	switch v := ev[idx].(type) {
	case string:
		return v
	default:
		return ""
	}
}

func (b *Bridge) peerAllowed(peerID int) bool {
	// 1) VK_PEER_IDS whitelist
	if len(b.cfg.VKPeerIDs) > 0 {
		return b.cfg.VKPeerIDs[peerID]
	}
	// 2) VK_PEER_ID single
	if b.cfg.VKPeerID != 0 {
		return peerID == b.cfg.VKPeerID
	}
	// 3) otherwise allow all
	return true
}

// LongPoll event 4 handler
func (b *Bridge) HandleNewMessageEvent(ev []interface{}) error {
	// event 4: [4, msg_id, flags, peer_id, ts, text, ...]
	msgID := asInt(ev, 1)
	flags := asInt(ev, 2)
	peerID := asInt(ev, 3)
	ts := int64(asInt(ev, 4))
	textEv := asString(ev, 5)

	if peerID == 0 || msgID == 0 {
		return nil
	}

	if !b.peerAllowed(peerID) {
		return nil
	}

	if !b.cfg.VKForwardOutbox && (flags&2) != 0 {
		return nil
	}

	// правильный запрос: messages.getById
	msg, err := b.vk.GetMessageByID(msgID)
	if err != nil {
		if b.debug {
			log.Printf("vk get message error peer=%d msg_id=%d: %v (fallback to LP text)", peerID, msgID, err)
		}
		chatTitle := b.peers.Title(peerID)
		sender := "id0"
		payload := buildMessageHTML(chatTitle, sender, ts, "text", textEv)
		return b.bc.SendTextHTML(payload)
	}

	// иногда полезно проверить, что сообщение реально из этого peer
	if msg.PeerID != 0 && msg.PeerID != peerID {
		if b.debug {
			log.Printf("skip msg_id=%d: msg.peer_id=%d != lp.peer_id=%d", msgID, msg.PeerID, peerID)
		}
		return nil
	}

	chatTitle := b.peers.Title(peerID)
	sender := b.names.Name(msg.FromID)

	// Debug log
	if b.debug {
		short := strings.TrimSpace(msg.Text)
		if len([]rune(short)) > 80 {
			short = string([]rune(short)[:80]) + "…"
		}
		log.Printf("[MSG] peer=%d msg_id=%d from_id=%d ts=%s text=%q",
			peerID, msgID, msg.FromID, formatTime(msg.Date), short)
	}

	// 1) Попробуем отправить фото (variant A). Если оно отправилось и текст ушёл в caption — текст отдельно не дублируем.
	textSentInCaption := false
	if b.photo != nil {
		res, perr := b.photo.HandleVariantA(context.Background(), chatTitle, sender, msg.Date, msg.Text, msg)
		if perr != nil {
			log.Printf("photo handler error: %v", perr)
		}
		if res.Sent && res.UsedTextInCap {
			textSentInCaption = true
		}
	}

	// 2) Текст отдельно (если он есть и не ушёл в caption)
	if strings.TrimSpace(msg.Text) != "" && !textSentInCaption {
		payload := buildMessageHTML(chatTitle, sender, msg.Date, "text", msg.Text)
		if err := b.bc.SendTextHTML(payload); err != nil {
			return fmt.Errorf("broadcast text: %w", err)
		}
	}

	// 3) Если текста нет и фото нет — можно ничего не отправлять
	return nil
}
