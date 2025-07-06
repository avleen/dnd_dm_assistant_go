package claude

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

const (
	anthropicAPIURL = "https://api.anthropic.com/v1/messages"
	defaultModel    = "claude-3-5-sonnet-20241022"
	maxTokens       = 4096
	timeout         = 60 * time.Second
)

// Service handles communication with the Anthropic Claude API
type Service struct {
	apiKey string
	client *http.Client
	debug  bool
}

// Message represents a single message in the conversation (with timestamp for internal use)
type Message struct {
	Role      string      `json:"role"`      // "user", "assistant", or "system"
	Content   interface{} `json:"content"`   // string or []ContentBlock
	Timestamp time.Time   `json:"timestamp"` // When this message was created
}

// APIMessage represents a message for the Claude API (without timestamp)
type APIMessage struct {
	Role    string      `json:"role"`    // "user", "assistant", or "system"
	Content interface{} `json:"content"` // string or []ContentBlock
}

// ContentBlock represents a content block (text, image, etc.)
type ContentBlock struct {
	Type string `json:"type"` // "text"
	Text string `json:"text"`
}

// APIRequest represents a request to the Claude API
type APIRequest struct {
	Model     string       `json:"model"`
	Messages  []APIMessage `json:"messages"`
	MaxTokens int          `json:"max_tokens"`
	System    string       `json:"system,omitempty"`
}

// Request represents a request to the Claude API (deprecated, kept for compatibility)
type Request struct {
	Model     string    `json:"model"`
	Messages  []Message `json:"messages"`
	MaxTokens int       `json:"max_tokens"`
	System    string    `json:"system,omitempty"`
}

// Response represents a response from the Claude API
type Response struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Role    string `json:"role"`
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Model        string `json:"model"`
	StopReason   string `json:"stop_reason"`
	StopSequence string `json:"stop_sequence"`
	Usage        struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

// ErrorResponse represents an error response from the Claude API
type ErrorResponse struct {
	Type  string `json:"type"`
	Error struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

// NewService creates a new Claude service
func NewService(apiKey string, debug bool) *Service {
	return &Service{
		apiKey: apiKey,
		client: &http.Client{
			Timeout: timeout,
		},
		debug: debug,
	}
}

// SendMessage sends a message to Claude and returns the response
func (s *Service) SendMessage(messages []Message, systemPrompt string) (*Response, error) {
	if s.debug {
		log.Printf("[CLAUDE] Sending %d messages to Claude API", len(messages))
	}

	// Create API-compatible messages (without timestamp field)
	apiMessages := make([]APIMessage, len(messages))
	for i, msg := range messages {
		apiMessages[i] = APIMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	// Prepare the request
	request := APIRequest{
		Model:     defaultModel,
		Messages:  apiMessages,
		MaxTokens: maxTokens,
		System:    systemPrompt,
	}

	// Convert to JSON
	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	if s.debug {
		log.Printf("[CLAUDE] Request payload size: %d bytes", len(jsonData))
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", anthropicAPIURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", s.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	// Send request
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if s.debug {
		log.Printf("[CLAUDE] Response status: %d, body size: %d bytes", resp.StatusCode, len(body))
	}

	// Handle non-200 responses
	if resp.StatusCode != http.StatusOK {
		var errorResp ErrorResponse
		if err := json.Unmarshal(body, &errorResp); err != nil {
			return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
		}
		return nil, fmt.Errorf("API error: %s - %s", errorResp.Error.Type, errorResp.Error.Message)
	}

	// Parse successful response
	var response Response
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if s.debug {
		log.Printf("[CLAUDE] Response: model=%s, input_tokens=%d, output_tokens=%d",
			response.Model, response.Usage.InputTokens, response.Usage.OutputTokens)
	}

	return &response, nil
}

// CreateUserMessage creates a user message
func CreateUserMessage(content string) Message {
	return Message{
		Role:      "user",
		Content:   content,
		Timestamp: time.Now(),
	}
}

// CreateAssistantMessage creates an assistant message
func CreateAssistantMessage(content string) Message {
	return Message{
		Role:      "assistant",
		Content:   content,
		Timestamp: time.Now(),
	}
}

// GetResponseText extracts the text content from a Claude response
func GetResponseText(response *Response) string {
	if len(response.Content) > 0 && response.Content[0].Type == "text" {
		return response.Content[0].Text
	}
	return ""
}
