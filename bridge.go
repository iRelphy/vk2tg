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

// --- helpers to read ExtendedEvents tail safely ---
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

// LongPoll event 4 handler
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
	if !b.peerAllowed(peerID) {
		return nil
	}
	// outbox filtering
	if !b.cfg.VKForwardOutbox && (flags&2) != 0 {
		return nil
	}

	// Try to get full message:
	// 1) messages.getById (works if msgID is global)
	msg, err := b.vk.GetMessageByID(msgID)
	if err != nil {
		// 2) messages.getByConversationMessageId (works if msgID is conversation_message_id)
		msg2, err2 := b.vk.GetMessageByConversationMessageID(peerID, msgID)
		if err2 == nil {
			msg = msg2
			err = nil
		}
	}

	// Parse ExtendedEvents tail for fallback title/from
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

	// If still no msg -> fallback to LP text
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

	// sanity check
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

	// Debug log
	if b.debug {
		short := strings.TrimSpace(msg.Text)
		if len([]rune(short)) > 80 {
			short = string([]rune(short)[:80]) + "…"
		}
		log.Printf("[MSG] peer=%d msg_id=%d from_id=%d ts=%s text=%q", peerID, msgID, senderID, formatTime(msg.Date), short)
	}

	// 1) Try photo
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

	// 2) Text separately (if not in caption)
	if strings.TrimSpace(msg.Text) != "" && !textSentInCaption {
		payload := buildMessageHTML(chatTitle, sender, msg.Date, "text", msg.Text)
		if err := b.bc.SendTextHTML(payload); err != nil {
			return fmt.Errorf("broadcast text: %w", err)
		}
	}

	return nil
}
