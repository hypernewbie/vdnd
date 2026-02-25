package main

import (
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
)

const (
	discordMessageSoftLimit = 1900
	discordFlushInterval    = 3 * time.Second
)

// DiscordStreamer buffers streamed content and flushes updates to Discord on a ticker.
type DiscordStreamer struct {
	Session     *discordgo.Session
	Interaction *discordgo.Interaction
	ChannelID   string

	buffer      strings.Builder
	fullMessage strings.Builder
	lastEdit    string

	currentMsgID string

	chunkChan  chan string
	statusChan chan string
	doneChan   chan struct{}

	doneOnce sync.Once
	wg       sync.WaitGroup
}

func NewDiscordStreamer(session *discordgo.Session, interaction *discordgo.Interaction, channelID string) *DiscordStreamer {
	d := &DiscordStreamer{
		Session:     session,
		Interaction: interaction,
		ChannelID:   channelID,
		chunkChan:   make(chan string, 64),
		statusChan:  make(chan string, 32),
		doneChan:    make(chan struct{}),
	}
	d.wg.Add(1)
	go d.run()
	return d
}

func (d *DiscordStreamer) Status(msg string) {
	d.send(d.statusChan, msg)
}

func (d *DiscordStreamer) Chunk(delta string) {
	d.send(d.chunkChan, delta)
}

func (d *DiscordStreamer) Close() {
	d.doneOnce.Do(func() {
		close(d.doneChan)
	})
	d.wg.Wait()
}

func (d *DiscordStreamer) send(ch chan string, text string) {
	if text == "" {
		return
	}
	select {
	case <-d.doneChan:
		return
	case ch <- text:
	}
}

func (d *DiscordStreamer) run() {
	defer d.wg.Done()
	ticker := time.NewTicker(discordFlushInterval)
	defer ticker.Stop()

	for {
		select {
		case delta := <-d.chunkChan:
			d.appendText(delta)
		case status := <-d.statusChan:
			d.appendText(status)
		case <-ticker.C:
			d.flush()
		case <-d.doneChan:
			d.flush()
			return
		}
	}
}

func (d *DiscordStreamer) appendText(text string) {
	d.buffer.WriteString(text)
	d.fullMessage.WriteString(text)
}

func (d *DiscordStreamer) flush() {
	content := d.buffer.String()
	if content == "" || content == d.lastEdit {
		return
	}

	for len(content) > discordMessageSoftLimit {
		split := findShardSplit(content, discordMessageSoftLimit)
		chunkToKeep := content[:split]
		if chunkToKeep != d.lastEdit {
			if err := d.editCurrent(chunkToKeep); err != nil {
				slog.Error("failed to edit Discord message", "error", err)
				return
			}
			d.lastEdit = chunkToKeep
		}

		overflow := strings.TrimLeft(content[split:], "\n")
		nextChunk := overflow
		if len(nextChunk) > discordMessageSoftLimit {
			nextSplit := findShardSplit(nextChunk, discordMessageSoftLimit)
			nextChunk = nextChunk[:nextSplit]
			overflow = strings.TrimLeft(overflow[nextSplit:], "\n")
		} else {
			overflow = ""
		}
		if nextChunk == "" {
			nextChunk = overflow
			overflow = ""
		}

		msg, err := d.Session.ChannelMessageSend(d.ChannelID, nextChunk)
		if err != nil {
			slog.Error("failed to create Discord shard message", "error", err)
			return
		}
		d.currentMsgID = msg.ID
		d.lastEdit = nextChunk
		content = nextChunk + overflow
	}

	if content != d.lastEdit {
		if err := d.editCurrent(content); err != nil {
			slog.Error("failed to flush Discord content", "error", err)
			return
		}
		d.lastEdit = content
	}

	d.buffer.Reset()
	d.buffer.WriteString(content)
}

func (d *DiscordStreamer) editCurrent(content string) error {
	if d.Session == nil {
		return fmt.Errorf("discord session is nil")
	}

	if d.currentMsgID == "" {
		_, err := d.Session.InteractionResponseEdit(d.Interaction, &discordgo.WebhookEdit{Content: &content})
		return err
	}
	_, err := d.Session.ChannelMessageEdit(d.ChannelID, d.currentMsgID, content)
	return err
}

func findShardSplit(content string, limit int) int {
	if len(content) <= limit {
		return len(content)
	}
	for i := limit; i > 0 && i > limit-200; i-- {
		switch content[i-1] {
		case '\n', '.', '!', '?', ' ':
			return i
		}
	}
	return limit
}

type discordReporter struct {
	streamer *DiscordStreamer
}

func (r *discordReporter) Status(msg string) {
	r.streamer.Status(msg)
}

func (r *discordReporter) Chunk(delta string) {
	r.streamer.Chunk(delta)
}
