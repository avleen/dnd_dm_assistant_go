package claude

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"
)

// ConversationManager manages the persistent conversation with Claude
type ConversationManager struct {
	service          *Service
	filePath         string
	maxMessages      int
	debug            bool
	systemPrompt     string
	messages         []Message
	transcriptionBuf []string
	mutex            sync.RWMutex
}

// ConversationData represents the data structure saved to disk
type ConversationData struct {
	SystemPrompt string    `json:"system_prompt"`
	Messages     []Message `json:"messages"`
	LastSaved    time.Time `json:"last_saved"`
	Version      string    `json:"version"`
}

const (
	conversationVersion = "1.0"
	defaultSystemPrompt = `You are an expert Dungeon Master assistant for a D&D 5e game. You are listening to live voice transcriptions from the players and DM during their session.

Your role is to:
1. Help the DM by answering rules questions quickly and accurately
2. Suggest interesting plot developments or complications when asked
3. Provide NPC dialogue, descriptions, or roleplay assistance
4. Help track combat, initiative, or game state when requested
5. Offer creative solutions to problems that arise during play

Guidelines:
- Keep responses concise but helpful (1-3 paragraphs max unless asked for more detail)
- Always specify which D&D 5e rules you're referencing
- If you're uncertain about a rule, say so and suggest where to look it up
- Don't make decisions for the DM - offer options and suggestions
- Pay attention to the ongoing conversation context
- The DM or others may ask you questions directly by addressing you as CLAUDE, so be ready to respond

The conversation below represents the ongoing D&D session. Recent transcriptions will show as "[TRANSCRIPTION] SSRC <number>: <text>" where each SSRC represents a different speaker.`
)

// NewConversationManager creates a new conversation manager
func NewConversationManager(service *Service, filePath string, maxMessages int, debug bool) *ConversationManager {
	cm := &ConversationManager{
		service:          service,
		filePath:         filePath,
		maxMessages:      maxMessages,
		debug:            debug,
		systemPrompt:     defaultSystemPrompt,
		messages:         make([]Message, 0),
		transcriptionBuf: make([]string, 0),
	}

	// Try to load existing conversation
	if err := cm.loadFromDisk(); err != nil {
		if debug {
			log.Printf("[CLAUDE] No existing conversation file or failed to load: %v", err)
			log.Printf("[CLAUDE] Starting fresh conversation")
		}
	}

	return cm
}

// AddTranscription adds a transcription to the buffer
func (cm *ConversationManager) AddTranscription(ssrc uint32, text string) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	transcription := fmt.Sprintf("[TRANSCRIPTION] SSRC %d: %s", ssrc, text)
	cm.transcriptionBuf = append(cm.transcriptionBuf, transcription)

	if cm.debug {
		log.Printf("[CLAUDE] Added transcription to buffer (total: %d)", len(cm.transcriptionBuf))
	}
}

// FlushTranscriptions flushes buffered transcriptions to the conversation
func (cm *ConversationManager) FlushTranscriptions() {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	if len(cm.transcriptionBuf) == 0 {
		return
	}

	// Combine all buffered transcriptions into a single user message
	content := strings.Join(cm.transcriptionBuf, "\n")
	message := CreateUserMessage(content)

	cm.messages = append(cm.messages, message)
	cm.transcriptionBuf = cm.transcriptionBuf[:0] // Clear buffer

	if cm.debug {
		log.Printf("[CLAUDE] Flushed transcriptions to conversation (total messages: %d)", len(cm.messages))
	}

	// Trim messages if we exceed the limit
	cm.trimMessages()

	// Save to disk
	if err := cm.saveToDisk(); err != nil {
		log.Printf("[CLAUDE] ⚠️ Failed to save conversation: %v", err)
	}
}

// AskQuestion sends a direct question to Claude and returns the response
func (cm *ConversationManager) AskQuestion(question string) (string, error) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	// First flush any pending transcriptions
	if len(cm.transcriptionBuf) > 0 {
		content := strings.Join(cm.transcriptionBuf, "\n")
		transcriptionMsg := CreateUserMessage(content)
		cm.messages = append(cm.messages, transcriptionMsg)
		cm.transcriptionBuf = cm.transcriptionBuf[:0]
	}

	// Add the question as a user message
	questionMsg := CreateUserMessage(question)
	cm.messages = append(cm.messages, questionMsg)

	if cm.debug {
		log.Printf("[CLAUDE] Asking question: %s", question)
	}

	// Prepare messages for API (exclude system messages from the message array)
	apiMessages := make([]Message, 0, len(cm.messages))
	for _, msg := range cm.messages {
		if msg.Role != "system" {
			apiMessages = append(apiMessages, msg)
		}
	}

	// Send to Claude
	response, err := cm.service.SendMessage(apiMessages, cm.systemPrompt)
	if err != nil {
		return "", fmt.Errorf("failed to get response from Claude: %w", err)
	}

	// Extract response text
	responseText := GetResponseText(response)
	if responseText == "" {
		return "", fmt.Errorf("received empty response from Claude")
	}

	// Add Claude's response to the conversation
	assistantMsg := CreateAssistantMessage(responseText)
	cm.messages = append(cm.messages, assistantMsg)

	// Trim messages if needed
	cm.trimMessages()

	// Save to disk
	if err := cm.saveToDisk(); err != nil {
		log.Printf("[CLAUDE] ⚠️ Failed to save conversation: %v", err)
	}

	if cm.debug {
		log.Printf("[CLAUDE] Got response (%d chars)", len(responseText))
	}

	return responseText, nil
}

// GetConversationSummary returns a summary of the current conversation
func (cm *ConversationManager) GetConversationSummary() string {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	summary := fmt.Sprintf("Conversation: %d messages", len(cm.messages))
	if len(cm.transcriptionBuf) > 0 {
		summary += fmt.Sprintf(", %d pending transcriptions", len(cm.transcriptionBuf))
	}

	return summary
}

// ClearConversation clears the conversation history
func (cm *ConversationManager) ClearConversation() error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	cm.messages = cm.messages[:0]
	cm.transcriptionBuf = cm.transcriptionBuf[:0]

	if err := cm.saveToDisk(); err != nil {
		return fmt.Errorf("failed to save cleared conversation: %w", err)
	}

	if cm.debug {
		log.Printf("[CLAUDE] Conversation cleared")
	}

	return nil
}

// HasPendingTranscriptions returns true if there are transcriptions waiting to be flushed
func (cm *ConversationManager) HasPendingTranscriptions() bool {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()
	return len(cm.transcriptionBuf) > 0
}

// trimMessages removes old messages if we exceed the maximum
func (cm *ConversationManager) trimMessages() {
	if len(cm.messages) <= cm.maxMessages {
		return
	}

	// Keep the most recent messages
	keepCount := cm.maxMessages * 3 / 4 // Keep 75% when trimming
	startIndex := len(cm.messages) - keepCount

	cm.messages = cm.messages[startIndex:]

	if cm.debug {
		log.Printf("[CLAUDE] Trimmed conversation to %d messages", len(cm.messages))
	}
}

// saveToDisk saves the conversation to disk
func (cm *ConversationManager) saveToDisk() error {
	data := ConversationData{
		SystemPrompt: cm.systemPrompt,
		Messages:     cm.messages,
		LastSaved:    time.Now(),
		Version:      conversationVersion,
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal conversation data: %w", err)
	}

	if err := os.WriteFile(cm.filePath, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write conversation file: %w", err)
	}

	if cm.debug {
		log.Printf("[CLAUDE] Saved conversation to %s (%d messages)", cm.filePath, len(cm.messages))
	}

	return nil
}

// loadFromDisk loads the conversation from disk
func (cm *ConversationManager) loadFromDisk() error {
	data, err := os.ReadFile(cm.filePath)
	if err != nil {
		return fmt.Errorf("failed to read conversation file: %w", err)
	}

	var conversationData ConversationData
	if err := json.Unmarshal(data, &conversationData); err != nil {
		return fmt.Errorf("failed to unmarshal conversation data: %w", err)
	}

	// Validate version compatibility
	if conversationData.Version != conversationVersion {
		if cm.debug {
			log.Printf("[CLAUDE] ⚠️ Conversation file version mismatch (file: %s, current: %s)",
				conversationData.Version, conversationVersion)
		}
	}

	cm.systemPrompt = conversationData.SystemPrompt
	if cm.systemPrompt == "" {
		cm.systemPrompt = defaultSystemPrompt
	}

	cm.messages = conversationData.Messages
	if cm.messages == nil {
		cm.messages = make([]Message, 0)
	}

	if cm.debug {
		log.Printf("[CLAUDE] Loaded conversation from %s (%d messages, last saved: %s)",
			cm.filePath, len(cm.messages), conversationData.LastSaved.Format(time.RFC3339))
	}

	return nil
}
