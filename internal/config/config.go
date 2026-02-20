package config

// Package config reads and validates application settings.
// We keep configuration in environment variables (and optionally in main.env file).
//
// Why env?
// - easy to run locally and on servers
// - secrets (tokens) are not stored in code
//
// See main.env.example for a ready template.

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config is all settings needed to run the bridge.
type Config struct {
	// VKToken is a VK API token (must have access to messages).
	VKToken string

	// Dialog filters:
	// - VKPeerID: single peer_id to forward
	// - VKPeerIDs: several peer_ids to forward
	// If both are empty -> forward from all dialogs we can see.
	VKPeerID  int
	VKPeerIDs map[int]bool

	// TGToken is Telegram bot token from @BotFather.
	TGToken string

	// SubscribersFile is where we store Telegram chat IDs.
	SubscribersFile string

	// VKForwardOutbox controls whether we forward our own outgoing messages.
	VKForwardOutbox bool

	// Debug prints extra logs.
	Debug bool
}

// PeerAllowed checks if a peer_id passes current filter rules.
func (c Config) PeerAllowed(peerID int) bool {
	if peerID == 0 {
		return false
	}
	if len(c.VKPeerIDs) > 0 {
		return c.VKPeerIDs[peerID]
	}
	if c.VKPeerID != 0 {
		return peerID == c.VKPeerID
	}
	return true
}

func cleanEnvValue(s string) string {
	s = strings.TrimSpace(s)
	// Allow inline comments in env file: "VALUE # comment"
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

// Load reads env vars, fills defaults and validates required fields.
func Load() (Config, error) {
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

	// VK_PEER_ID (one number)
	if raw := cleanEnvValue(os.Getenv("VK_PEER_ID")); raw != "" {
		id, err := strconv.Atoi(raw)
		if err != nil {
			return cfg, fmt.Errorf("VK_PEER_ID must be int, got %q", raw)
		}
		cfg.VKPeerID = id
	}

	// VK_PEER_IDS (comma-separated)
	cfg.VKPeerIDs = map[int]bool{}
	if raw := cleanEnvValue(os.Getenv("VK_PEER_IDS")); raw != "" {
		parts := strings.Split(raw, ",")
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			id, err := strconv.Atoi(p)
			if err != nil {
				return cfg, fmt.Errorf("VK_PEER_IDS contains non-int: %q", p)
			}
			cfg.VKPeerIDs[id] = true
		}
	}

	return cfg, nil
}
