package mom

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Attachment represents a file attached to a Slack message.
type Attachment struct {
	Original string `json:"original"` // Original filename
	Local    string `json:"local"`    // Relative path in working dir
}

// LoggedMessage is a message stored in the channel log.
type LoggedMessage struct {
	Date        string       `json:"date"`
	Ts          string       `json:"ts"`
	User        string       `json:"user"`
	UserName    string       `json:"userName,omitempty"`
	DisplayName string       `json:"displayName,omitempty"`
	Text        string       `json:"text"`
	Attachments []Attachment `json:"attachments"`
	IsBot       bool         `json:"isBot"`
}

// ChannelStoreConfig configures a ChannelStore.
type ChannelStoreConfig struct {
	WorkingDir string
	BotToken   string // Needed for authenticated file downloads
}

// PendingDownload is a queued attachment download.
type PendingDownload struct {
	ChannelID string
	LocalPath string
	URL       string
}

// ChannelStore manages per-channel directories, message logging, and attachment downloads.
type ChannelStore struct {
	mu             sync.Mutex
	workingDir     string
	botToken       string
	pendingDLs     []PendingDownload
	isDownloading  bool
	recentlyLogged map[string]time.Time // "channelId:ts" → logged time
}

// NewChannelStore creates a new ChannelStore.
func NewChannelStore(config ChannelStoreConfig) *ChannelStore {
	os.MkdirAll(config.WorkingDir, 0755)
	return &ChannelStore{
		workingDir:     config.WorkingDir,
		botToken:       config.BotToken,
		recentlyLogged: make(map[string]time.Time),
	}
}

// GetChannelDir returns (creating if needed) the directory for a channel.
func (s *ChannelStore) GetChannelDir(channelID string) string {
	dir := filepath.Join(s.workingDir, channelID)
	os.MkdirAll(dir, 0755)
	return dir
}

// GenerateLocalFilename creates a unique filename for an attachment.
func (s *ChannelStore) GenerateLocalFilename(originalName string, timestamp string) string {
	ts := parseSlackTimestamp(timestamp)
	sanitized := sanitizeFilename(originalName)
	return fmt.Sprintf("%d_%s", ts, sanitized)
}

// SlackFile represents a Slack attachment file.
type SlackFile struct {
	Name               string
	URLPrivateDownload string
	URLPrivate         string
}

// ProcessAttachments processes Slack attachments and queues downloads.
func (s *ChannelStore) ProcessAttachments(channelID string, files []SlackFile, timestamp string) []Attachment {
	var attachments []Attachment
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, f := range files {
		url := f.URLPrivateDownload
		if url == "" {
			url = f.URLPrivate
		}
		if url == "" || f.Name == "" {
			continue
		}

		filename := s.GenerateLocalFilename(f.Name, timestamp)
		localPath := fmt.Sprintf("%s/attachments/%s", channelID, filename)

		attachments = append(attachments, Attachment{
			Original: f.Name,
			Local:    localPath,
		})

		s.pendingDLs = append(s.pendingDLs, PendingDownload{
			ChannelID: channelID,
			LocalPath: localPath,
			URL:       url,
		})
	}

	go s.processDownloadQueue()
	return attachments
}

// LogMessage appends a message to the channel's log.jsonl. Returns false if duplicate.
func (s *ChannelStore) LogMessage(channelID string, msg LoggedMessage) (bool, error) {
	s.mu.Lock()
	dedupeKey := channelID + ":" + msg.Ts
	if _, exists := s.recentlyLogged[dedupeKey]; exists {
		s.mu.Unlock()
		return false, nil
	}
	s.recentlyLogged[dedupeKey] = time.Now()
	s.mu.Unlock()

	// Clean up old entries (> 60s)
	go func() {
		time.Sleep(60 * time.Second)
		s.mu.Lock()
		delete(s.recentlyLogged, dedupeKey)
		s.mu.Unlock()
	}()

	if msg.Date == "" {
		msg.Date = time.Now().UTC().Format(time.RFC3339)
	}

	logPath := filepath.Join(s.GetChannelDir(channelID), "log.jsonl")
	data, err := json.Marshal(msg)
	if err != nil {
		return false, err
	}

	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return false, err
	}
	defer f.Close()

	_, err = f.Write(append(data, '\n'))
	return err == nil, err
}

// LogBotResponse logs a bot response to the channel log.
func (s *ChannelStore) LogBotResponse(channelID string, text string, ts string) error {
	_, err := s.LogMessage(channelID, LoggedMessage{
		Date:        time.Now().UTC().Format(time.RFC3339),
		Ts:          ts,
		User:        "bot",
		Text:        text,
		Attachments: nil,
		IsBot:       true,
	})
	return err
}

// GetLastTimestamp returns the timestamp of the last logged message, or empty string.
func (s *ChannelStore) GetLastTimestamp(channelID string) string {
	logPath := filepath.Join(s.workingDir, channelID, "log.jsonl")
	data, err := os.ReadFile(logPath)
	if err != nil {
		return ""
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) == 0 || lines[0] == "" {
		return ""
	}

	lastLine := lines[len(lines)-1]
	var msg LoggedMessage
	if err := json.Unmarshal([]byte(lastLine), &msg); err != nil {
		return ""
	}
	return msg.Ts
}

func (s *ChannelStore) processDownloadQueue() {
	s.mu.Lock()
	if s.isDownloading || len(s.pendingDLs) == 0 {
		s.mu.Unlock()
		return
	}
	s.isDownloading = true
	s.mu.Unlock()

	for {
		s.mu.Lock()
		if len(s.pendingDLs) == 0 {
			s.isDownloading = false
			s.mu.Unlock()
			return
		}
		item := s.pendingDLs[0]
		s.pendingDLs = s.pendingDLs[1:]
		s.mu.Unlock()

		if err := s.downloadAttachment(item.LocalPath, item.URL); err != nil {
			// Log warning but continue
			fmt.Fprintf(os.Stderr, "[WARN] Failed to download attachment %s: %v\n", item.LocalPath, err)
		}
	}
}

func (s *ChannelStore) downloadAttachment(localPath string, url string) error {
	filePath := filepath.Join(s.workingDir, localPath)
	dir := filepath.Dir(filePath)
	os.MkdirAll(dir, 0755)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+s.botToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	return os.WriteFile(filePath, data, 0644)
}

// --- Helpers ---

func parseSlackTimestamp(ts string) int64 {
	if strings.Contains(ts, ".") {
		f, _ := strconv.ParseFloat(ts, 64)
		return int64(f * 1000)
	}
	i, _ := strconv.ParseInt(ts, 10, 64)
	return i
}

func sanitizeFilename(name string) string {
	var b strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '.' || r == '_' || r == '-' {
			b.WriteRune(r)
		} else {
			b.WriteByte('_')
		}
	}
	return b.String()
}
