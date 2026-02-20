package bridge

// Package bridge connects VK and Telegram.
// It receives VK LongPoll events and forwards them to Telegram subscribers.
//
// Main idea:
// 1) VK LongPoll tells us: "new message happened"
// 2) We load full message details via VK API (for sender name, attachments, etc.)
// 3) We format a neat Telegram message (HTML)
// 4) We send it to all subscribed Telegram chats

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/iRelphy/vk2tg/internal/config"
	"github.com/iRelphy/vk2tg/internal/tg"
	"github.com/iRelphy/vk2tg/internal/util"
	"github.com/iRelphy/vk2tg/internal/vk"
)

// Bridge glues together all components: config, VK, resolvers and Telegram broadcaster.
type Bridge struct {
	cfg   config.Config
	vk    *vk.Client
	names *vk.NameResolver
	peers *vk.PeerResolver
	bc    *tg.Broadcaster
	photo *PhotoHandler
	debug bool
}

// New creates a ready-to-use bridge.
func New(cfg config.Config, vkClient *vk.Client, names *vk.NameResolver, peers *vk.PeerResolver, bc *tg.Broadcaster, photo *PhotoHandler) *Bridge {
	return &Bridge{
		cfg:   cfg,
		vk:    vkClient,
		names: names,
		peers: peers,
		bc:    bc,
		photo: photo,
		debug: cfg.Debug,
	}
}

// asInt and asString convert LongPoll event fields to Go types safely.
// LongPoll events are []interface{} and VK may return numbers as int/int64/float64.
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

// --- helpers to read ExtendedEvents tail safely ---
//
// With ExtendedEvents enabled, LongPoll event 4 may contain an extra map at index 6
// with useful fields like "from" and sometimes "title".
func extraMap(ev []interface{}) map[string]interface{} {
	if len(ev) <= 6 {
		return nil
	}
	switch m := ev[6].(type) {
	case map[string]interface{}:
		return m
	case map[string]string:
		out := make(map[string]interface{}, len(m))
		for k, v := range m {
			out[k] = v
		}
		return out
	default:
		return nil
	}
}

func getIntAny(v interface{}) int {
	switch t := v.(type) {
	case int:
		return t
	case int64:
		return int(t)
	case float64:
		return int(t)
	case string:
		t = strings.TrimSpace(t)
		if t == "" {
			return 0
		}
		var n int
		_, _ = fmt.Sscanf(t, "%d", &n)
		return n
	default:
		return 0
	}
}

func getStrAny(v interface{}) string {
	if s, ok := v.(string); ok {
		return strings.TrimSpace(s)
	}
	return ""
}

// HandleNewMessageEvent is registered as LongPoll handler for VK event type 4 ("new message").
// Signature must match vksdk longpoll callback.
func (b *Bridge) HandleNewMessageEvent(ev []interface{}) error {
	// event 4: [4, msg_id OR conversation_msg_id, flags, peer_id, ts, text, ...]
	msgID := asInt(ev, 1)
	flags := asInt(ev, 2)
	peerID := asInt(ev, 3)
	ts := int64(asInt(ev, 4))
	textEv := asString(ev, 5)

	if peerID == 0 || msgID == 0 {
		return nil
	}
	if !b.cfg.PeerAllowed(peerID) {
		return nil
	}

	// VK flag bit 2 means "outbox" (sent by us).
	// If VK_FORWARD_OUTBOX=false, skip such messages.
	if !b.cfg.VKForwardOutbox && (flags&2) != 0 {
		return nil
	}

	// Parse ExtendedEvents tail for fallback title/from.
	ext := extraMap(ev)
	fromIDEv := 0
	titleEv := ""
	if ext != nil {
		if v, ok := ext["from"]; ok {
			fromIDEv = getIntAny(v)
		}
		if fromIDEv == 0 {
			if v, ok := ext["from_id"]; ok {
				fromIDEv = getIntAny(v)
			}
		}
		if v, ok := ext["title"]; ok {
			titleEv = getStrAny(v)
		}
	}

	chatTitle := strings.TrimSpace(titleEv)
	if chatTitle == "" {
		chatTitle = b.peers.Title(peerID)
	}

	// Try to get full VK message:
	// 1) messages.getById (global message_id)
	msg, err := b.vk.GetMessageByID(msgID)
	if err != nil {
		// 2) messages.getByConversationMessageId (conversation_message_id inside a chat)
		msg2, err2 := b.vk.GetMessageByConversationMessageID(peerID, msgID)
		if err2 == nil {
			msg = msg2
			err = nil
		}
	}

	// If still no message -> fallback to raw LongPoll text.
	if err != nil || msg == nil {
		if b.debug {
			log.Printf("vk get message error peer=%d msg_id=%d: %v (fallback to LP text)", peerID, msgID, err)
		}

		senderID := fromIDEv
		sender := "id0"
		if senderID != 0 {
			sender = b.names.Name(senderID)
		}

		payload := buildMessageHTML(chatTitle, sender, ts, "text", textEv)
		return b.bc.SendTextHTML(payload)
	}

	// sanity check: if VK returned a different peer, skip.
	if msg.PeerID != 0 && msg.PeerID != peerID {
		if b.debug {
			log.Printf("skip msg_id=%d: msg.peer_id=%d != lp.peer_id=%d", msgID, msg.PeerID, peerID)
		}
		return nil
	}

	senderID := msg.FromID
	if senderID == 0 {
		senderID = fromIDEv
	}
	sender := "id0"
	if senderID != 0 {
		sender = b.names.Name(senderID)
	}

	if b.debug {
		short := strings.TrimSpace(msg.Text)
		r := []rune(short)
		if len(r) > 80 {
			short = string(r[:80]) + "…"
		}
		log.Printf("[MSG] peer=%d msg_id=%d from_id=%d ts=%s text=%q", peerID, msgID, senderID, util.FormatTime(msg.Date), short)
	}

	// 1) Try photo first (if exists).
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

	// 2) Send text separately if it was not included in the photo caption.
	if strings.TrimSpace(msg.Text) != "" && !textSentInCaption {
		payload := buildMessageHTML(chatTitle, sender, msg.Date, "text", msg.Text)
		if err := b.bc.SendTextHTML(payload); err != nil {
			return fmt.Errorf("broadcast text: %w", err)
		}
	}

	return nil
}
