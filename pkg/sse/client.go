package sse

import (
	"sync"
)

type SseClient struct {
	Id   string
	Name string
}

type Clients struct {
	data map[string]SseClient
	mu   sync.Mutex
}

func (c *Clients) Add(user SseClient) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data[user.Id] = user
}

func (c *Clients) Remove(user SseClient) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.data, user.Id)
}
func (c *Clients) Count() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.data)
}

func (c *Clients) Clients() *Clients {
	return c
}
