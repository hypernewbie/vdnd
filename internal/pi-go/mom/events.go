package mom

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// EventType identifies the type of mom event.
type EventType string

const (
	EventImmediate EventType = "immediate"
	EventOneShot   EventType = "one-shot"
	EventPeriodic  EventType = "periodic"
)

// MomEvent represents an event that triggers agent activity.
type MomEvent struct {
	Type      EventType `json:"type"`
	ChannelID string    `json:"channelId"`
	Text      string    `json:"text"`
	At        string    `json:"at,omitempty"`       // For one-shot: ISO timestamp
	Schedule  string    `json:"schedule,omitempty"` // For periodic: cron expression
	Timezone  string    `json:"timezone,omitempty"` // For periodic: timezone
}

// EventHandler is called when an event fires.
type EventHandler func(event MomEvent)

// EventsWatcher monitors a directory for event files and executes them.
type EventsWatcher struct {
	mu        sync.Mutex
	eventsDir string
	handler   EventHandler
	timers    map[string]*time.Timer
	stopCh    chan struct{}
	started   bool
}

// NewEventsWatcher creates an events watcher for the given directory.
func NewEventsWatcher(eventsDir string, handler EventHandler) *EventsWatcher {
	os.MkdirAll(eventsDir, 0755)
	return &EventsWatcher{
		eventsDir: eventsDir,
		handler:   handler,
		timers:    make(map[string]*time.Timer),
	}
}

// Start begins watching for events.
func (w *EventsWatcher) Start() {
	w.mu.Lock()
	if w.started {
		w.mu.Unlock()
		return
	}
	w.started = true
	w.stopCh = make(chan struct{})
	w.mu.Unlock()

	// Scan existing events
	w.scanExisting()

	// Poll for new events periodically
	go w.pollLoop()
}

// Stop stops watching and cancels all scheduled events.
func (w *EventsWatcher) Stop() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.started {
		return
	}
	w.started = false
	close(w.stopCh)

	for name, timer := range w.timers {
		timer.Stop()
		delete(w.timers, name)
	}
}

func (w *EventsWatcher) pollLoop() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-w.stopCh:
			return
		case <-ticker.C:
			w.scanExisting()
		}
	}
}

func (w *EventsWatcher) scanExisting() {
	entries, err := os.ReadDir(w.eventsDir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		w.handleFile(entry.Name())
	}
}

func (w *EventsWatcher) handleFile(filename string) {
	path := filepath.Join(w.eventsDir, filename)
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}

	var event MomEvent
	if err := json.Unmarshal(data, &event); err != nil {
		fmt.Fprintf(os.Stderr, "[WARN] Invalid event file %s: %v\n", filename, err)
		return
	}

	switch event.Type {
	case EventImmediate:
		w.handleImmediate(filename, event)
	case EventOneShot:
		w.handleOneShot(filename, event)
	case EventPeriodic:
		// Periodic events use cron; simplified here as a repeating timer
		w.handlePeriodic(filename, event)
	}
}

func (w *EventsWatcher) handleImmediate(filename string, event MomEvent) {
	// Execute immediately and delete file
	w.handler(event)
	w.deleteFile(filename)
}

func (w *EventsWatcher) handleOneShot(filename string, event MomEvent) {
	w.mu.Lock()
	if _, exists := w.timers[filename]; exists {
		w.mu.Unlock()
		return // Already scheduled
	}
	w.mu.Unlock()

	// Parse target time
	target, err := time.Parse(time.RFC3339, event.At)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[WARN] Invalid time in event %s: %v\n", filename, err)
		return
	}

	delay := time.Until(target)
	if delay <= 0 {
		// Already past, execute now
		w.handler(event)
		w.deleteFile(filename)
		return
	}

	timer := time.AfterFunc(delay, func() {
		w.handler(event)
		w.deleteFile(filename)
		w.mu.Lock()
		delete(w.timers, filename)
		w.mu.Unlock()
	})

	w.mu.Lock()
	w.timers[filename] = timer
	w.mu.Unlock()
}

func (w *EventsWatcher) handlePeriodic(filename string, event MomEvent) {
	w.mu.Lock()
	if _, exists := w.timers[filename]; exists {
		w.mu.Unlock()
		return // Already scheduled
	}
	w.mu.Unlock()

	// Simplified: use a fixed 1-hour interval for periodic events
	// A full implementation would parse cron expressions
	timer := time.AfterFunc(1*time.Hour, func() {
		w.handler(event)
		// Reschedule
		w.mu.Lock()
		delete(w.timers, filename)
		w.mu.Unlock()
		w.handlePeriodic(filename, event)
	})

	w.mu.Lock()
	w.timers[filename] = timer
	w.mu.Unlock()
}

func (w *EventsWatcher) deleteFile(filename string) {
	path := filepath.Join(w.eventsDir, filename)
	os.Remove(path)
}
