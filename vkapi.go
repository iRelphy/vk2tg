package main

import (
	"fmt"

	"github.com/SevereCloud/vksdk/v3/api"
)

type VKClient struct {
	VK *api.VK
}

func NewVKClient(token string) *VKClient {
	vk := api.NewVK(token)
	return &VKClient{VK: vk}
}

type VKEnvelope[T any] struct {
	Response T `json:"response"`
}

// messages.getByConversationMessageId
type VKGetByConvMsgIDResponse struct {
	Count int         `json:"count"`
	Items []VKMessage `json:"items"`
}

type VKMessage struct {
	ID                    int            `json:"id"`
	ConversationMessageID int            `json:"conversation_message_id"`
	Date                  int64          `json:"date"`
	PeerID                int            `json:"peer_id"`
	FromID                int            `json:"from_id"`
	Text                  string         `json:"text"`
	Attachments           []VKAttachment `json:"attachments"`
}

type VKAttachment struct {
	Type  string   `json:"type"`
	Photo *VKPhoto `json:"photo,omitempty"`
}

type VKPhoto struct {
	ID      int           `json:"id"`
	OwnerID int           `json:"owner_id"`
	Sizes   []VKPhotoSize `json:"sizes"`
}

type VKPhotoSize struct {
	URL    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
	Type   string `json:"type"`
}

// messages.getById
type VKGetByIDResponse struct {
	Count int         `json:"count"`
	Items []VKMessage `json:"items"`
}

func (c *VKClient) GetMessageByID(msgID int) (*VKMessage, error) {
	var out VKEnvelope[VKGetByIDResponse]
	err := c.VK.RequestUnmarshal(
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
	if len(out.Response.Items) == 0 {
		return nil, fmt.Errorf("no message items for message_id=%d", msgID)
	}
	msg := out.Response.Items[0]
	return &msg, nil
}

func bestPhotoURL(p *VKPhoto) string {
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

func (c *VKClient) ExtractFirstPhotoURL(msg *VKMessage) string {
	if msg == nil {
		return ""
	}
	for _, a := range msg.Attachments {
		if a.Type == "photo" && a.Photo != nil {
			if u := bestPhotoURL(a.Photo); u != "" {
				return u
			}
		}
	}
	return ""
}
