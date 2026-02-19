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

	// cache
	r.mu.RLock()
	if v, ok := r.cache[id]; ok && strings.TrimSpace(v) != "" {
		r.mu.RUnlock()
		return v
	}
	r.mu.RUnlock()

	name, ok := r.fetchName(id)
	if ok && strings.TrimSpace(name) != "" {
		r.mu.Lock()
		r.cache[id] = name
		r.mu.Unlock()
		return name
	}

	// НЕ кешируем фейлы, чтобы в следующий раз попытаться снова
	if id > 0 {
		return fmt.Sprintf("id%d", id)
	}
	return fmt.Sprintf("club%d", -id)
}

func (r *NameResolver) fetchName(id int) (string, bool) {
	if id > 0 {
		var out struct {
			Response []struct {
				FirstName string `json:"first_name"`
				LastName  string `json:"last_name"`
			} `json:"response"`
		}

		err := r.vk.RequestUnmarshal("users.get", &out, api.Params{
			"user_ids":  fmt.Sprintf("%d", id),
			"name_case": "nom",
		})
		if err != nil {
			if r.debug {
				log.Printf("[users.get] id=%d error: %v", id, err)
			}
			return "", false
		}
		if len(out.Response) == 0 {
			if r.debug {
				log.Printf("[users.get] id=%d empty response", id)
			}
			return "", false
		}

		fn := strings.TrimSpace(out.Response[0].FirstName)
		ln := strings.TrimSpace(out.Response[0].LastName)
		name := strings.TrimSpace(fn + " " + ln)
		if name == "" {
			return "", false
		}
		return name, true
	}

	// group/community
	gid := -id
	var gr struct {
		Response []struct {
			Name string `json:"name"`
		} `json:"response"`
	}

	// ВАЖНО: group_ids (надежнее)
	err := r.vk.RequestUnmarshal("groups.getById", &gr, api.Params{
		"group_ids": fmt.Sprintf("%d", gid),
	})
	if err != nil {
		if r.debug {
			log.Printf("[groups.getById] gid=%d error: %v", gid, err)
		}
		return "", false
	}
	if len(gr.Response) == 0 {
		if r.debug {
			log.Printf("[groups.getById] gid=%d empty response", gid)
		}
		return "", false
	}

	name := strings.TrimSpace(gr.Response[0].Name)
	if name == "" {
		return "", false
	}
	return name, true
}
