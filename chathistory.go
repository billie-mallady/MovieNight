package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/zorchenhimer/MovieNight/common"
	"github.com/zorchenhimer/MovieNight/files"
)

type ChatHistory struct {
	Messages []HistoryMessage `json:"messages"`
	mutex    sync.RWMutex
	filename string
}

type HistoryMessage struct {
	Timestamp time.Time       `json:"timestamp"`
	Type      common.DataType `json:"type"`
	Data      json.RawMessage `json:"data"`
	HTML      string          `json:"html"`
}

func NewChatHistory(filename string) *ChatHistory {
	if filename == "" {
		filename = "chat_history.json"
	}

	ch := &ChatHistory{
		Messages: make([]HistoryMessage, 0),
		filename: files.JoinRunPath(filename),
	}

	// Load existing history
	ch.Load()
	return ch
}

func (ch *ChatHistory) AddMessage(chatData common.ChatData) {
	if !settings.ChatHistory {
		return
	}

	ch.mutex.Lock()
	defer ch.mutex.Unlock()

	// Convert ChatData to JSON
	jsonData, err := chatData.ToJSON()
	if err != nil {
		common.LogErrorf("Error converting chat data to JSON: %v", err)
		return
	}

	// Create history message
	histMsg := HistoryMessage{
		Timestamp: time.Now(),
		Type:      chatData.Type,
		Data:      jsonData.Data,
		HTML:      chatData.Data.HTML(),
	}

	ch.Messages = append(ch.Messages, histMsg)

	// Keep only the last MaxMessageCount messages to prevent unbounded growth
	maxMessages := settings.MaxMessageCount * 2 // Keep more in storage than we show
	if len(ch.Messages) > maxMessages {
		ch.Messages = ch.Messages[len(ch.Messages)-maxMessages:]
	}

	// Save to file asynchronously to avoid blocking
	go ch.Save()
}

func (ch *ChatHistory) GetRecentMessages(count int) []common.ChatData {
	ch.mutex.RLock()
	defer ch.mutex.RUnlock()

	if count <= 0 || count > len(ch.Messages) {
		count = len(ch.Messages)
	}

	messages := make([]common.ChatData, 0, count)
	startIdx := len(ch.Messages) - count

	for i := startIdx; i < len(ch.Messages); i++ {
		msg := ch.Messages[i]

		// Skip events like joins/leaves that don't need to be shown in history
		if msg.Type == common.DTEvent {
			continue
		}

		// Convert back to ChatData
		chatDataJSON := common.ChatDataJSON{
			Type: msg.Type,
			Data: msg.Data,
		}

		chatData, err := chatDataJSON.ToData()
		if err != nil {
			common.LogErrorf("Error converting history message to ChatData: %v", err)
			continue
		}

		messages = append(messages, chatData)
	}

	return messages
}

func (ch *ChatHistory) Load() error {
	ch.mutex.Lock()
	defer ch.mutex.Unlock()

	if _, err := os.Stat(ch.filename); os.IsNotExist(err) {
		// File doesn't exist, start with empty history
		common.LogInfof("Chat history file %s does not exist, starting with empty history", ch.filename)
		return nil
	}

	data, err := os.ReadFile(ch.filename)
	if err != nil {
		return fmt.Errorf("error reading chat history file: %w", err)
	}

	if len(data) == 0 {
		// Empty file, start with empty history
		return nil
	}

	err = json.Unmarshal(data, ch)
	if err != nil {
		return fmt.Errorf("error unmarshaling chat history: %w", err)
	}

	common.LogInfof("Loaded %d messages from chat history", len(ch.Messages))
	return nil
}

func (ch *ChatHistory) Save() error {
	ch.mutex.RLock()
	defer ch.mutex.RUnlock()

	data, err := json.MarshalIndent(ch, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling chat history: %w", err)
	}

	// Write to temporary file first, then rename for atomic operation
	tempFile := ch.filename + ".tmp"
	err = os.WriteFile(tempFile, data, 0644)
	if err != nil {
		return fmt.Errorf("error writing chat history temp file: %w", err)
	}

	err = os.Rename(tempFile, ch.filename)
	if err != nil {
		return fmt.Errorf("error renaming chat history temp file: %w", err)
	}

	return nil
}

func (ch *ChatHistory) Clear() error {
	ch.mutex.Lock()
	defer ch.mutex.Unlock()

	ch.Messages = make([]HistoryMessage, 0)
	return ch.Save()
}
