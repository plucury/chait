package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const (
	DeepseekAPIURL     = "https://api.deepseek.com/v1/chat/completions"
	DefaultModel       = "deepseek-chat"
	DefaultTemperature = 1.0
)

// Available models for Deepseek API
var AvailableModels = []string{
	"deepseek-chat",
	"deepseek-reasoner",
}

// Temperature presets for different use cases
type TemperaturePreset struct {
	Name        string
	Value       float64
	Description string
}

// Available temperature presets for Deepseek API
var TemperaturePresets = []TemperaturePreset{
	{"Code Generation", 0.0, "Code generation or math problem solving"},
	{"Data Extraction", 1.0, "Data extraction and analysis"},
	{"General Conversation", 1.3, "General conversation"},
	{"Translation", 1.3, "Translation tasks"},
	{"Creative Writing", 1.5, "Creative writing or poetry"},
}

// ChatMessage represents a message in the chat
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatRequest represents the request to the chat API
type ChatRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	Temperature float64       `json:"temperature,omitempty"`
}

// ChatResponse represents the response from the chat API
type ChatResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index        int         `json:"index"`
		Message      ChatMessage `json:"message"`
		FinishReason string      `json:"finish_reason"`
	} `json:"choices"`
	Error *ErrorResponse `json:"error,omitempty"`
}

// ErrorResponse represents an error from the API
type ErrorResponse struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Param   string `json:"param"`
	Code    string `json:"code"`
}

// SendChatRequest sends a chat request to the Deepseek API
func SendChatRequest(apiKey string, messages []ChatMessage, model string, temperature float64) (string, error) {
	// If model is empty, use the default model
	if model == "" {
		model = DefaultModel
	}

	// Create the request body
	requestBody := ChatRequest{
		Model:       model,
		Messages:    messages,
		Temperature: temperature,
	}

	// Marshal the request body to JSON
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("error marshaling request: %v", err)
	}

	// Create a new HTTP request
	req, err := http.NewRequest("POST", DeepseekAPIURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("error creating request: %v", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response: %v", err)
	}

	// Check for error status code
	if resp.StatusCode != http.StatusOK {
		// Try to parse the error response
		var errorResp struct {
			Error struct {
				Message string `json:"message"`
				Type    string `json:"type"`
				Code    string `json:"code"`
			} `json:"error"`
		}

		if err := json.Unmarshal(body, &errorResp); err == nil && errorResp.Error.Message != "" {
			// Handle specific error types
			switch {
			case errorResp.Error.Message == "Insufficient Balance" || errorResp.Error.Code == "invalid_request_error":
				return "", fmt.Errorf("Deepseek API account has insufficient balance. Please check your account or contact Deepseek support.")
			case resp.StatusCode == 401:
				return "", fmt.Errorf("Authentication failed. Please check your API key.")
			default:
				return "", fmt.Errorf("API error: %s (Code: %s)", errorResp.Error.Message, errorResp.Error.Code)
			}
		} else {
			// Fallback to generic error message
			return "", fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
		}
	}

	// Parse the response
	var chatResponse ChatResponse
	if err := json.Unmarshal(body, &chatResponse); err != nil {
		return "", fmt.Errorf("error parsing response: %v", err)
	}

	// Check if we have any choices
	if len(chatResponse.Choices) == 0 {
		return "", fmt.Errorf("no response choices returned")
	}

	// Return the content of the first choice
	return chatResponse.Choices[0].Message.Content, nil
}
