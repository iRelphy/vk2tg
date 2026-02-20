package vk

import (
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/SevereCloud/vksdk/v3/api"
)

// PeerResolver converts peer_id -> human-readable chat title.
// We cache titles in memory to avoid extra VK API calls.
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

// Title returns chat title for a VK peer_id.
// If VK does not provide a title (for example, no access), we return a fallback.
func (r *PeerResolver) Title(peerID int) string {
	if peerID == 0 {
		return "Диалог"
	}

	// 1) Fast path: cache.
	r.mu.RLock()
	if v, ok := r.cache[peerID]; ok && strings.TrimSpace(v) != "" {
		r.mu.RUnlock()
		return v
	}
	r.mu.RUnlock()

	// 2) Ask VK.
	title := strings.TrimSpace(r.fetchTitle(peerID))

	// 3) Fallback for cases when VK can't / doesn't return title.
	if title == "" {
		if peerID >= 2000000000 {
			title = fmt.Sprintf("Чат %d", peerID-2000000000)
		} else {
			title = "Личные сообщения"
		}
	}

	// Save to cache.
	r.mu.Lock()
	r.cache[peerID] = title
	r.mu.Unlock()

	return title
}

type conversationsByID struct {
	Count int `json:"count"`
	Items []struct {
		ChatSettings struct {
			Title string `json:"title"`
		} `json:"chat_settings"`
	} `json:"items"`
}

func (r *PeerResolver) fetchTitle(peerID int) string {
	var out conversationsByID

	err := r.vk.RequestUnmarshal(
		"messages.getConversationsById",
		&out,
		api.Params{
			"peer_ids": fmt.Sprintf("%d", peerID), // string is the most compatible
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
