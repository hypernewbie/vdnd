package main

import (
	"sync"

	"github.com/bwmarrin/discordgo"
)

// MessageCache stores recent messages for each channel
type MessageCache struct {
	mu       sync.RWMutex
	messages map[string][]*discordgo.Message
	maxLimit int
}

// NewMessageCache creates a new instance of MessageCache
func NewMessageCache(limit int) *MessageCache {
	if limit == 0 {
		limit = 50
	}
	return &MessageCache{
		messages: make(map[string][]*discordgo.Message),
		maxLimit: limit,
	}
}

// Add appends a message to the cache for its channel
func (c *MessageCache) Add(m *discordgo.Message) {
	c.mu.Lock()
	defer c.mu.Unlock()
	msgs := c.messages[m.ChannelID]
	msgs = append(msgs, m)
	if len(msgs) > c.maxLimit {
		msgs = msgs[1:]
	}
	c.messages[m.ChannelID] = msgs
}

// Clear removes all cached messages for a specific channel
func (c *MessageCache) Clear(channelID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.messages, channelID)
}

// Get returns a copy of the cached messages for a channel
func (c *MessageCache) Get(channelID string) []*discordgo.Message {
	c.mu.RLock()
	defer c.mu.RUnlock()
	msgs := c.messages[channelID]
	// Return a copy to avoid race conditions
	res := make([]*discordgo.Message, len(msgs))
	copy(res, msgs)
	return res
}
