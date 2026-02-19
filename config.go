package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	VKToken string

	// Фильтр диалогов:
	VKPeerID  int          // один peer
	VKPeerIDs map[int]bool // список peer'ов

	TGToken         string
	SubscribersFile string

	VKForwardOutbox bool
	Debug           bool
}

func cleanEnvValue(s string) string {
	s = strings.TrimSpace(s)
	// на всякий случай режем инлайн-комменты
	if i := strings.Index(s, "#"); i >= 0 {
		s = strings.TrimSpace(s[:i])
	}
	return strings.TrimSpace(s)
}

func parseBool(name string, def bool) bool {
	raw := cleanEnvValue(os.Getenv(name))
	if raw == "" {
		return def
	}
	v, err := strconv.ParseBool(raw)
	if err != nil {
		return def
	}
	return v
}

func LoadConfig() (Config, error) {
	var cfg Config

	cfg.VKToken = cleanEnvValue(os.Getenv("VK_TOKEN"))
	if cfg.VKToken == "" {
		return cfg, fmt.Errorf("VK_TOKEN is required")
	}

	cfg.TGToken = cleanEnvValue(os.Getenv("TG_TOKEN"))
	if cfg.TGToken == "" {
		return cfg, fmt.Errorf("TG_TOKEN is required")
	}

	cfg.SubscribersFile = cleanEnvValue(os.Getenv("SUBSCRIBERS_FILE"))
	if cfg.SubscribersFile == "" {
		cfg.SubscribersFile = "subscribers.json"
	}

	cfg.VKForwardOutbox = parseBool("VK_FORWARD_OUTBOX", true)
	cfg.Debug = parseBool("DEBUG", false)

	// VK_PEER_ID (одно число)
	vkPeerRaw := cleanEnvValue(os.Getenv("VK_PEER_ID"))
	if vkPeerRaw != "" {
		id, err := strconv.Atoi(vkPeerRaw)
		if err != nil {
			return cfg, fmt.Errorf(`VK_PEER_ID must be int, got %q`, vkPeerRaw)
		}
		cfg.VKPeerID = id
	}

	// VK_PEER_IDS (список)
	cfg.VKPeerIDs = map[int]bool{}
	vkPeersRaw := cleanEnvValue(os.Getenv("VK_PEER_IDS")) // "2000000004,2000000005"
	if vkPeersRaw != "" {
		parts := strings.Split(vkPeersRaw, ",")
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			id, err := strconv.Atoi(p)
			if err != nil {
				return cfg, fmt.Errorf(`VK_PEER_IDS contains non-int: %q`, p)
			}
			cfg.VKPeerIDs[id] = true
		}
	}

	return cfg, nil
}
