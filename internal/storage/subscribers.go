package storage

// Package storage keeps persistent data on disk.
// Right now we store only Telegram subscriber chat IDs in a JSON file.
//
// Why JSON?
// - easy to read and edit
// - no database needed for a small tool

import (
	"encoding/json"
	"os"
	"sync"
	"time"
)

// Subscriber describes one Telegram chat that wants to receive VK messages.
type Subscriber struct {
	ChatID    int64  `json:"chat_id"`
	Username  string `json:"username,omitempty"`
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
	AddedAt   string `json:"added_at,omitempty"`
}

// SubscriberStore loads/saves subscribers from/to a JSON file.
// It is safe for concurrent use (we use mutex).
type SubscriberStore struct {
	path string

	mu   sync.RWMutex
	subs map[int64]Subscriber
}

// NewSubscriberStore creates a store that will use the given JSON file path.
func NewSubscriberStore(path string) *SubscriberStore {
	if path == "" {
		path = "subscribers.json"
	}
	return &SubscriberStore{
		path: path,
		subs: map[int64]Subscriber{},
	}
}

// Load reads subscribers from file.
// It also supports older file formats and auto-migrates to the newest format.
func (s *SubscriberStore) Load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	b, err := os.ReadFile(s.path)
	if err != nil {
		// File does not exist yet — that's OK.
		return nil
	}

	// 1) New format: {"subscribers":[...]}
	var wrap struct {
		Subscribers []Subscriber `json:"subscribers"`
	}
	if err := json.Unmarshal(b, &wrap); err == nil && wrap.Subscribers != nil {
		for _, sub := range wrap.Subscribers {
			if sub.ChatID == 0 {
				continue
			}
			if sub.AddedAt == "" {
				sub.AddedAt = time.Now().Format(time.RFC3339)
			}
			s.subs[sub.ChatID] = sub
		}
		return nil
	}

	// 2) Array of Subscriber objects: [{"chat_id":...}, ...]
	var arrSubs []Subscriber
	if err := json.Unmarshal(b, &arrSubs); err == nil && arrSubs != nil {
		for _, sub := range arrSubs {
			if sub.ChatID == 0 {
				continue
			}
			if sub.AddedAt == "" {
				sub.AddedAt = time.Now().Format(time.RFC3339)
			}
			s.subs[sub.ChatID] = sub
		}
		// Migrate to the newest format.
		return s.saveLocked()
	}

	// 3) Very old format: [123, 456] (just chat IDs)
	var arrIDs []int64
	if err := json.Unmarshal(b, &arrIDs); err == nil && arrIDs != nil {
		for _, id := range arrIDs {
			if id == 0 {
				continue
			}
			s.subs[id] = Subscriber{
				ChatID:  id,
				AddedAt: time.Now().Format(time.RFC3339),
			}
		}
		// Migrate to the newest format.
		return s.saveLocked()
	}

	// If nothing matched — return a real JSON error for the user.
	return json.Unmarshal(b, &wrap)
}

// Save writes current subscribers to disk.
func (s *SubscriberStore) Save() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.saveLocked()
}

func (s *SubscriberStore) saveLocked() error {
	wrap := struct {
		Subscribers []Subscriber `json:"subscribers"`
	}{
		Subscribers: make([]Subscriber, 0, len(s.subs)),
	}
	for _, sub := range s.subs {
		wrap.Subscribers = append(wrap.Subscribers, sub)
	}

	b, err := json.MarshalIndent(wrap, "", "  ")
	if err != nil {
		return err
	}

	// 0600: only current user can read/write the file (protect tokens/chat IDs).
	return os.WriteFile(s.path, b, 0o600)
}

// Add adds a new subscriber (Telegram chat) and saves the file.
// Returns changed=false if the subscriber already exists.
func (s *SubscriberStore) Add(sub Subscriber) (changed bool, _ error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if sub.ChatID == 0 {
		return false, nil
	}
	if _, ok := s.subs[sub.ChatID]; ok {
		return false, nil
	}
	if sub.AddedAt == "" {
		sub.AddedAt = time.Now().Format(time.RFC3339)
	}
	s.subs[sub.ChatID] = sub
	return true, s.saveLocked()
}

// Remove removes subscriber by chat ID and saves the file.
// Returns changed=false if this chat wasn't subscribed.
func (s *SubscriberStore) Remove(chatID int64) (changed bool, _ error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.subs[chatID]; !ok {
		return false, nil
	}
	delete(s.subs, chatID)
	return true, s.saveLocked()
}

// ChatIDs returns all subscriber chat IDs.
// We return a copy, so callers cannot modify internal map.
func (s *SubscriberStore) ChatIDs() []int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]int64, 0, len(s.subs))
	for id := range s.subs {
		out = append(out, id)
	}
	return out
}

// Count returns number of subscribers.
func (s *SubscriberStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.subs)
}
