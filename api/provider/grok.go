package provider

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/plucury/chait/util"
)

// GrokProvider implements the Provider interface for Grok API
type GrokProvider struct {
	BaseProvider // 嵌入基础提供者结构体
}

const (
	grokAPIURL             = "https://api.x.ai/v1/chat/completions"
	grokDefaultModel       = "grok-2-1212"
	grokDefaultTemperature = 1.0 // Default temperature as per Grok API documentation
)

// Available models for Grok API
var grokAvailableModels = []string{
	"grok-2-1212",
}

// Available temperature presets for Grok API
var grokTemperaturePresets = []TemperaturePreset{
	{"Focused", 0.2, "More focused and deterministic responses for specific tasks"},
	{"Balanced Low", 0.5, "Good balance with slight focus on determinism"},
	{"Balanced", 1.0, "Default balance between randomness and determinism"},
	{"Creative", 1.5, "More random and creative responses"},
	{"Highly Creative", 2.0, "Maximum randomness for highly varied outputs"},
}

// NewGrokProvider creates a new instance of GrokProvider
func NewGrokProvider() Provider {
	provider := &GrokProvider{
		BaseProvider: BaseProvider{
			Name:               "grok",
			CurrentModel:       grokDefaultModel,
			CurrentTemperature: grokDefaultTemperature,
		},
	}
	return provider
}

// GetName returns the name of the provider
func (p *GrokProvider) GetName() string {
	return p.Name
}

// GetDefaultModel returns the default model for this provider
func (p *GrokProvider) GetDefaultModel() string {
	return grokDefaultModel
}

// GetAvailableModels returns the list of available models for this provider
func (p *GrokProvider) GetAvailableModels() []string {
	return grokAvailableModels
}

// GetDefaultTemperature returns the default temperature for this provider
func (p *GrokProvider) GetDefaultTemperature() float64 {
	return grokDefaultTemperature
}

// GetTemperaturePresets returns the available temperature presets for this provider
func (p *GrokProvider) GetTemperaturePresets() []TemperaturePreset {
	return grokTemperaturePresets
}

// SetCurrentTemperature sets the current temperature with Grok-specific validation
func (p *GrokProvider) SetCurrentTemperature(temp float64) error {
	// Validate temperature range specific to Grok (0-2)
	if temp < 0 || temp > 2.0 {
		return fmt.Errorf("Grok temperature must be between 0.0 and 2.0. Higher values like 0.8 will make the output more random, while lower values like 0.2 will make it more focused and deterministic")
	}

	p.CurrentTemperature = temp
	return nil
}

// chatRequest represents the request to the Grok chat API
type grokChatRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	Temperature float64       `json:"temperature,omitempty"`
	Stream      bool          `json:"stream,omitempty"`
}

// chatResponse represents the response from the Grok chat API
type grokChatResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index        int         `json:"index"`
		Message      ChatMessage `json:"message"`
		Delta        ChatMessage `json:"delta,omitempty"`
		FinishReason string      `json:"finish_reason"`
	} `json:"choices"`
	Error *grokErrorResponse `json:"error,omitempty"`
}

// errorResponse represents an error from the Grok API
type grokErrorResponse struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Param   string `json:"param"`
	Code    string `json:"code"`
}

// SendChatRequest sends a chat request to the Grok API
func (p *GrokProvider) SendChatRequest(messages []ChatMessage) (string, error) {
	// 检查 API Key 是否已设置
	if p.APIKey == "" {
		return "", fmt.Errorf("API key not set for Grok provider")
	}

	// 创建请求体
	requestBody := grokChatRequest{
		Model:       p.CurrentModel,
		Messages:    messages,
		Temperature: p.CurrentTemperature,
	}

	util.DebugLog("Using Grok model: %s", p.CurrentModel)
	util.DebugLog("Using temperature: %.1f", p.CurrentTemperature)

	// 将请求体序列化为 JSON
	jsonData, err := json.Marshal(requestBody)
	util.DebugLog("Request JSON: %s", string(jsonData))
	if err != nil {
		return "", fmt.Errorf("error marshaling request: %v", err)
	}

	// 创建 HTTP 请求
	req, err := http.NewRequest("POST", grokAPIURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("error creating request: %v", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.APIKey)

	// 发送请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error connecting to Grok API: %v. Please check your internet connection and that the API is available.", err)
	}
	defer resp.Body.Close()

	// 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response: %v", err)
	}

	// 检查错误状态码
	if resp.StatusCode != http.StatusOK {
		// 尝试解析错误响应
		var errorResp struct {
			Error struct {
				Message string `json:"message"`
				Type    string `json:"type"`
				Code    string `json:"code"`
			} `json:"error"`
		}

		if err := json.Unmarshal(body, &errorResp); err == nil && errorResp.Error.Message != "" {
			// 处理特定类型的错误
			switch {
			case resp.StatusCode == 401:
				return "", fmt.Errorf("Authentication failed. Please check your API key.")
			default:
				return "", fmt.Errorf("API error: %s (Code: %s)", errorResp.Error.Message, errorResp.Error.Code)
			}
		} else {
			// 回退到通用错误消息
			return "", fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
		}
	}

	// 解析响应
	var chatResp grokChatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return "", fmt.Errorf("error parsing response: %v", err)
	}

	// 检查是否有选择
	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("no response choices in API response")
	}

	// 检查是否有错误
	if chatResp.Error != nil {
		return "", fmt.Errorf("API error: %s", chatResp.Error.Message)
	}

	// 返回响应内容
	return chatResp.Choices[0].Message.Content, nil
}

// SendStreamingChatRequest sends a streaming chat request to the Grok API
func (p *GrokProvider) SendStreamingChatRequest(messages []ChatMessage) (<-chan StreamResponse, error) {
	respChan := make(chan StreamResponse)

	// 检查 API Key 是否已设置
	if p.APIKey == "" {
		return nil, fmt.Errorf("API key not set for Grok provider")
	}

	// 创建请求体
	requestBody := grokChatRequest{
		Model:       p.CurrentModel,
		Messages:    messages,
		Temperature: p.CurrentTemperature,
		Stream:      true,
	}

	util.DebugLog("Using Grok model: %s (streaming)", p.CurrentModel)
	util.DebugLog("Using temperature: %.1f", p.CurrentTemperature)

	// 将请求体序列化为 JSON
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %v", err)
	}

	// 创建 HTTP 请求
	req, err := http.NewRequest("POST", grokAPIURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.APIKey)

	// 发送请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error connecting to Grok API: %v. Please check your internet connection and that the API is available.", err)
	}

	// 检查状态码
	if resp.StatusCode != http.StatusOK {
		// 读取错误响应
		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		// 尝试解析错误响应
		var errorResp grokChatResponse
		if err := json.Unmarshal(respBody, &errorResp); err == nil && errorResp.Error != nil {
			return nil, fmt.Errorf("API error: %s", errorResp.Error.Message)
		}

		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	// 启动 goroutine 处理流式响应
	go func() {
		defer resp.Body.Close()
		defer close(respChan)

		reader := bufio.NewReader(resp.Body)

		for {
			line, err := reader.ReadBytes('\n')
			if err != nil {
				if err != io.EOF {
					respChan <- StreamResponse{Error: fmt.Errorf("error reading stream: %v", err)}
				}
				break
			}

			// Skip empty lines
			line = bytes.TrimSpace(line)
			if len(line) == 0 {
				continue
			}

			// Remove "data: " prefix
			if bytes.HasPrefix(line, []byte("data: ")) {
				line = bytes.TrimPrefix(line, []byte("data: "))
			}

			// Check for stream end
			if string(line) == "[DONE]" {
				respChan <- StreamResponse{Done: true}
				break
			}

			// Skip empty JSON objects or invalid lines
			if string(line) == "{}" || len(line) == 0 {
				continue
			}

			// Debug log the line for troubleshooting only when debug mode is enabled
			if util.IsDebugMode() {
				util.DebugLog("Grok stream line: %s", string(line))
			}

			// Parse the response
			var streamResp grokChatResponse
			if err := json.Unmarshal(line, &streamResp); err != nil {
				if util.IsDebugMode() {
					util.DebugLog("Error parsing Grok stream: %v (line: %s)", err, string(line))
				}
				continue // Skip this line instead of breaking
			}

			// Check for API errors
			if streamResp.Error != nil {
				respChan <- StreamResponse{Error: fmt.Errorf("API error: %s", streamResp.Error.Message)}
				break
			}

			// Extract content from choices
			if len(streamResp.Choices) > 0 {
				content := streamResp.Choices[0].Delta.Content
				if content != "" {
					respChan <- StreamResponse{Content: content}
				}
			}
		}
	}()

	return respChan, nil
}

// SetCurrentModel sets the current model after validating it
func (p *GrokProvider) SetCurrentModel(model string) error {
	// 验证模型是否有效
	valid := false
	for _, m := range grokAvailableModels {
		if m == model {
			valid = true
			break
		}
	}

	if !valid {
		fmt.Printf("WARNING: Invalid model: %s. Available models: %v\n", model, grokAvailableModels)
		return fmt.Errorf("invalid model: %s. Available models: %v", model, grokAvailableModels)
	}

	// 设置模型并输出调试信息
	p.CurrentModel = model
	util.DebugLog("Grok model set to: %s", model)
	return nil
}

// LoadConfig loads the provider configuration from the given map
func (p *GrokProvider) LoadConfig(config map[string]interface{}) error {
	// 加载 API Key
	if apiKey, ok := config["api_key"].(string); ok {
		p.APIKey = apiKey
		util.DebugLog("Loaded API key for Grok provider")
	}

	// 加载当前模型
	if model, ok := config["model"].(string); ok {
		util.DebugLog("Found model in config: %s", model)
		if err := p.SetCurrentModel(model); err != nil {
			// 如果模型无效，使用默认模型
			fmt.Printf("WARNING: Invalid model in config, using default model: %s\n", grokDefaultModel)
			p.CurrentModel = grokDefaultModel
		}
	} else {
		// 如果没有设置模型，使用默认模型
		util.DebugLog("No model found in config, using default model: %s", grokDefaultModel)
		p.CurrentModel = grokDefaultModel
	}

	// 加载温度设置
	if temp, ok := config["temperature"].(float64); ok {
		if err := p.SetCurrentTemperature(temp); err != nil {
			// 如果温度无效，使用默认温度
			p.CurrentTemperature = grokDefaultTemperature
		}
	} else {
		// 如果没有设置温度，使用默认温度
		p.CurrentTemperature = grokDefaultTemperature
	}

	return nil
}

// SaveConfig saves the provider configuration to the given map
func (p *GrokProvider) SaveConfig(config map[string]interface{}) {
	config["api_key"] = p.APIKey
	config["model"] = p.CurrentModel
	config["temperature"] = p.CurrentTemperature
}

// IsReady returns whether the provider is ready to use
// For Grok, the provider is ready if the API key is set
func (p *GrokProvider) IsReady() bool {
	return p.APIKey != ""
}

// Register the provider
func init() {
	Register("grok", NewGrokProvider)
}
