package main

import (
	"encoding/json"
	"os"
	"sync"
	"time"
)

type Subscriber struct {
	ChatID    int64  `json:"chat_id"`
	Username  string `json:"username,omitempty"`
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
	AddedAt   string `json:"added_at,omitempty"`
}

type SubscriberStore struct {
	path string
	mu   sync.RWMutex
	subs map[int64]Subscriber
}

func NewSubscriberStore(path string) *SubscriberStore {
	if path == "" {
		path = "subscribers.json"
	}
	return &SubscriberStore{
		path: path,
		subs: map[int64]Subscriber{},
	}
}

func (s *SubscriberStore) Load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	b, err := os.ReadFile(s.path)
	if err != nil {
		// файла нет — ок
		return nil
	}

	// 1) Новый формат: {"subscribers":[...]}
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

	// 2) Массив подписчиков: [{"chat_id":...}, ...]
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
		// миграция в новый формат
		return s.saveLocked()
	}

	// 3) Старый формат: [123,456]
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
		// миграция в новый формат
		return s.saveLocked()
	}

	// если вообще не распарсили — значит файл битый
	return json.Unmarshal(b, &wrap) // вернёт нормальную ошибку
}

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
	return os.WriteFile(s.path, b, 0o600)
}

func (s *SubscriberStore) Add(sub Subscriber) (bool, error) {
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

func (s *SubscriberStore) Remove(chatID int64) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.subs[chatID]; !ok {
		return false, nil
	}
	delete(s.subs, chatID)
	return true, s.saveLocked()
}

func (s *SubscriberStore) ChatIDs() []int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]int64, 0, len(s.subs))
	for id := range s.subs {
		out = append(out, id)
	}
	return out
}

func (s *SubscriberStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.subs)
}
