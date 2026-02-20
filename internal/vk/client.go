package vk

// Package vk is responsible for all interaction with VK:
// - creating VK API client
// - fetching messages by id / conversation_message_id
// - extracting attachments (photo URL)
//
// This package does NOT know anything about Telegram.

import (
	"fmt"

	"github.com/SevereCloud/vksdk/v3/api"
)

// Client is a small wrapper around vksdk API client.
type Client struct {
	API *api.VK
}

func NewClient(token string) *Client {
	vk := api.NewVK(token)
	return &Client{API: vk}
}

// VK API responses for messages.getById and messages.getByConversationMessageId
// are shaped like: { "count": N, "items": [...] }.
//
// vksdk's RequestUnmarshal already extracts "response", so we unmarshal directly
// into this struct.
type getMessagesResponse struct {
	Count int       `json:"count"`
	Items []Message `json:"items"`
}

// Message is the part of VK message we need for forwarding.
type Message struct {
	ID                    int          `json:"id"`
	ConversationMessageID int          `json:"conversation_message_id"`
	Date                  int64        `json:"date"`
	PeerID                int          `json:"peer_id"`
	FromID                int          `json:"from_id"`
	Text                  string       `json:"text"`
	Attachments           []Attachment `json:"attachments"`
}

type Attachment struct {
	Type  string `json:"type"`
	Photo *Photo `json:"photo,omitempty"`
}

type Photo struct {
	ID      int         `json:"id"`
	OwnerID int         `json:"owner_id"`
	Sizes   []PhotoSize `json:"sizes"`
}

type PhotoSize struct {
	URL    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
	Type   string `json:"type"`
}

// GetMessageByID fetches a message by global message_id.
// This often works for private dialogs, but for group chats LongPoll may give
// conversation_message_id instead (not global id).
func (c *Client) GetMessageByID(msgID int) (*Message, error) {
	var out getMessagesResponse

	err := c.API.RequestUnmarshal(
		"messages.getById",
		&out,
		api.Params{
			"message_ids": msgID,
			"extended":    0,
		},
	)
	if err != nil {
		return nil, err
	}
	if len(out.Items) == 0 {
		return nil, fmt.Errorf("no message items for message_id=%d", msgID)
	}

	msg := out.Items[0]
	return &msg, nil
}

// GetMessageByConversationMessageID fetches a message by conversation_message_id inside a specific peer (chat).
// This is the fallback for the common LongPoll case in group chats.
func (c *Client) GetMessageByConversationMessageID(peerID, convMsgID int) (*Message, error) {
	var out getMessagesResponse

	err := c.API.RequestUnmarshal(
		"messages.getByConversationMessageId",
		&out,
		api.Params{
			"peer_id":                  peerID,
			"conversation_message_ids": convMsgID,
			"extended":                 0,
		},
	)
	if err != nil {
		return nil, err
	}
	if len(out.Items) == 0 {
		return nil, fmt.Errorf("no message items for peer_id=%d conversation_message_id=%d", peerID, convMsgID)
	}

	msg := out.Items[0]
	return &msg, nil
}

func bestPhotoURL(p *Photo) string {
	if p == nil {
		return ""
	}
	best := ""
	bestArea := 0
	for _, s := range p.Sizes {
		if s.URL == "" {
			continue
		}
		area := s.Width * s.Height
		if area >= bestArea {
			bestArea = area
			best = s.URL
		}
	}
	return best
}

// ExtractPhotoURLs возвращает список URL всех фото-вложений (в порядке attachments).
// Для каждого photo выбираем самый большой размер.
func (c *Client) ExtractPhotoURLs(msg *Message) []string {
	if msg == nil {
		return nil
	}

	seen := make(map[string]bool, 8)
	out := make([]string, 0, 4)

	for _, a := range msg.Attachments {
		if a.Type != "photo" || a.Photo == nil {
			continue
		}
		u := bestPhotoURL(a.Photo)
		if u == "" || seen[u] {
			continue
		}
		seen[u] = true
		out = append(out, u)
	}

	return out
}

// ExtractFirstPhotoURL оставляем для совместимости (если где-то используется).
func (c *Client) ExtractFirstPhotoURL(msg *Message) string {
	urls := c.ExtractPhotoURLs(msg)
	if len(urls) == 0 {
		return ""
	}
	return urls[0]
}
