package main

import (
	"fmt"
	"strings"
	"sync"

	"github.com/SevereCloud/vksdk/v3/api"
)

type PeerResolver struct {
	vk    *api.VK
	mu    sync.RWMutex
	cache map[int]string // peer_id -> title
}

func NewPeerResolver(vk *api.VK) *PeerResolver {
	return &PeerResolver{
		vk:    vk,
		cache: map[int]string{},
	}
}

func (r *PeerResolver) Title(peerID int) string {
	if peerID == 0 {
		return "Диалог"
	}

	r.mu.RLock()
	if v, ok := r.cache[peerID]; ok && v != "" {
		r.mu.RUnlock()
		return v
	}
	r.mu.RUnlock()

	title := r.fetchTitle(peerID)
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
	var out struct {
		Response struct {
			Count int `json:"count"`
			Items []struct {
				ChatSettings struct {
					Title string `json:"title"`
				} `json:"chat_settings"`
			} `json:"items"`
		} `json:"response"`
	}

	_ = r.vk.RequestUnmarshal("messages.getConversationsById", &out, api.Params{
		"peer_ids": peerID,
		"extended": 0,
	})

	if len(out.Response.Items) == 0 {
		return ""
	}
	return strings.TrimSpace(out.Response.Items[0].ChatSettings.Title)
}
