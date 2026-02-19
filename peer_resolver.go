package main

import (
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/SevereCloud/vksdk/v3/api"
)

type vkConversationsByID struct {
	Count int `json:"count"`
	Items []struct {
		ChatSettings struct {
			Title string `json:"title"`
		} `json:"chat_settings"`
	} `json:"items"`
}

type PeerResolver struct {
	vk    *api.VK
	debug bool

	mu    sync.RWMutex
	cache map[int]string // peer_id -> title
}

func NewPeerResolver(vk *api.VK, debug bool) *PeerResolver {
	return &PeerResolver{
		vk:    vk,
		debug: debug,
		cache: map[int]string{},
	}
}

func (r *PeerResolver) Title(peerID int) string {
	if peerID == 0 {
		return "Диалог"
	}

	r.mu.RLock()
	if v, ok := r.cache[peerID]; ok && strings.TrimSpace(v) != "" {
		r.mu.RUnlock()
		return v
	}
	r.mu.RUnlock()

	title := strings.TrimSpace(r.fetchTitle(peerID))

	if title == "" {
		// fallback
		if peerID >= 2000000000 {
			title = fmt.Sprintf("Чат %d", peerID-2000000000)
		} else {
			title = "Личные сообщения"
		}
	}

	r.mu.Lock()
	r.cache[peerID] = title
	r.mu.Unlock()

	return title
}

func (r *PeerResolver) fetchTitle(peerID int) string {
	var out vkConversationsByID

	err := r.vk.RequestUnmarshal(
		"messages.getConversationsById",
		&out,
		api.Params{
			"peer_ids": fmt.Sprintf("%d", peerID),
			"extended": 0,
		},
	)
	if err != nil {
		if r.debug {
			log.Printf("[messages.getConversationsById] peer=%d error: %v", peerID, err)
		}
		return ""
	}
	if len(out.Items) == 0 {
		if r.debug {
			log.Printf("[messages.getConversationsById] peer=%d empty items", peerID)
		}
		return ""
	}

	return strings.TrimSpace(out.Items[0].ChatSettings.Title)
}
