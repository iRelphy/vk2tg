package main

import (
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/SevereCloud/vksdk/v3/api"
)

type NameResolver struct {
	vk    *api.VK
	debug bool

	mu    sync.RWMutex
	cache map[int]string // id -> name
}

func NewNameResolver(vk *api.VK, debug bool) *NameResolver {
	return &NameResolver{
		vk:    vk,
		debug: debug,
		cache: map[int]string{},
	}
}

func (r *NameResolver) Name(id int) string {
	if id == 0 {
		return "id0"
	}

	r.mu.RLock()
	if v, ok := r.cache[id]; ok && strings.TrimSpace(v) != "" {
		r.mu.RUnlock()
		return v
	}
	r.mu.RUnlock()

	name := strings.TrimSpace(r.fetchName(id))
	if name == "" {
		if id < 0 {
			name = fmt.Sprintf("club%d", -id)
		} else {
			name = fmt.Sprintf("id%d", id)
		}
	}

	r.mu.Lock()
	r.cache[id] = name
	r.mu.Unlock()

	return name
}

func (r *NameResolver) fetchName(id int) string {
	if id > 0 {
		// users.get -> response это МАССИВ
		var out []struct {
			FirstName string `json:"first_name"`
			LastName  string `json:"last_name"`
		}

		err := r.vk.RequestUnmarshal(
			"users.get",
			&out,
			api.Params{
				"user_ids":  fmt.Sprintf("%d", id),
				"name_case": "nom",
			},
		)
		if err != nil {
			if r.debug {
				log.Printf("[users.get] id=%d error: %v", id, err)
			}
			return ""
		}
		if len(out) == 0 {
			if r.debug {
				log.Printf("[users.get] id=%d empty", id)
			}
			return ""
		}
		return strings.TrimSpace(out[0].FirstName + " " + out[0].LastName)
	}

	// groups.getById -> response тоже МАССИВ
	gid := -id
	var out []struct {
		Name string `json:"name"`
	}
	err := r.vk.RequestUnmarshal(
		"groups.getById",
		&out,
		api.Params{
			"group_ids": fmt.Sprintf("%d", gid),
		},
	)
	if err != nil {
		if r.debug {
			log.Printf("[groups.getById] id=%d error: %v", gid, err)
		}
		return ""
	}
	if len(out) == 0 {
		if r.debug {
			log.Printf("[groups.getById] id=%d empty", gid)
		}
		return ""
	}
	return strings.TrimSpace(out[0].Name)
}
