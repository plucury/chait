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

// OpenAIProvider implements the Provider interface for OpenAI API
type OpenAIProvider struct {
	BaseProvider // 嵌入基础提供者结构体
}

const (
	openaiAPIURL             = "https://api.openai.com/v1/chat/completions"
	openaiDefaultModel       = "gpt-4o"
	openaiDefaultTemperature = 1.0
)

// Available models for OpenAI API
var openaiAvailableModels = []string{
	"o1",          // OpenAI o1
	"o3-mini",     // OpenAI o3-mini
	"gpt-4.5",     // GPT-4.5
	"gpt-4o",      // GPT-4o
	"gpt-4o-mini", // GPT-4o mini
}

// Available temperature presets for OpenAI API
var openaiTemperaturePresets = []TemperaturePreset{
	{"Code Generation", 0.0, "Code generation or math problem solving"},
	{"Data Extraction", 0.3, "Data extraction and analysis"},
	{"General Conversation", 0.7, "General conversation"},
	{"Translation", 0.5, "Translation tasks"},
	{"Creative Writing", 1.0, "Creative writing or poetry"},
}

// NewOpenAIProvider creates a new instance of OpenAIProvider
func NewOpenAIProvider() Provider {
	provider := &OpenAIProvider{
		BaseProvider: BaseProvider{
			Name:               "openai",
			CurrentModel:       openaiDefaultModel,
			CurrentTemperature: openaiDefaultTemperature,
		},
	}
	return provider
}

// GetName returns the name of the provider
func (p *OpenAIProvider) GetName() string {
	return p.Name
}

// GetDefaultModel returns the default model for this provider
func (p *OpenAIProvider) GetDefaultModel() string {
	return openaiDefaultModel
}

// GetAvailableModels returns the list of available models for this provider
func (p *OpenAIProvider) GetAvailableModels() []string {
	return openaiAvailableModels
}

// GetDefaultTemperature returns the default temperature for this provider
func (p *OpenAIProvider) GetDefaultTemperature() float64 {
	return openaiDefaultTemperature
}

// GetTemperaturePresets returns the available temperature presets for this provider
func (p *OpenAIProvider) GetTemperaturePresets() []TemperaturePreset {
	return openaiTemperaturePresets
}

// SetCurrentTemperature sets the current temperature with OpenAI-specific validation
func (p *OpenAIProvider) SetCurrentTemperature(temp float64) error {
	// Validate temperature range specific to OpenAI (0-1)
	if temp < 0 || temp > 1.0 {
		return fmt.Errorf("OpenAI temperature must be between 0.0 and 1.0")
	}

	p.CurrentTemperature = temp
	return nil
}

// chatRequest represents the request to the OpenAI chat API
type openaiChatRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	Temperature float64       `json:"temperature,omitempty"`
	Stream      bool          `json:"stream,omitempty"`
}

// chatResponse represents the response from the OpenAI chat API
type openaiChatResponse struct {
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
	Error *openaiErrorResponse `json:"error,omitempty"`
}

// errorResponse represents an error from the OpenAI API
type openaiErrorResponse struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Param   string `json:"param"`
	Code    string `json:"code"`
}

// SendChatRequest sends a chat request to the OpenAI API
func (p *OpenAIProvider) SendChatRequest(messages []ChatMessage) (string, error) {
	// 检查 API Key 是否已设置
	if p.APIKey == "" {
		return "", fmt.Errorf("API key not set for OpenAI provider")
	}

	// 确保模型已设置，如果未设置则使用默认模型
	if p.CurrentModel == "" {
		p.CurrentModel = openaiDefaultModel
		fmt.Printf("WARNING: Model not set for OpenAI provider, using default model: %s\n", openaiDefaultModel)
	}

	// 输出调试信息
	util.DebugLog("Using OpenAI model: %s", p.CurrentModel)

	// 创建请求体
	requestBody := openaiChatRequest{
		Model:    p.CurrentModel,
		Messages: messages,
	}

	// Only set temperature for models that support it
	// o1 and o3-mini models ignore temperature
	if p.CurrentModel != "o1" && p.CurrentModel != "o3-mini" {
		requestBody.Temperature = p.CurrentTemperature
		util.DebugLog("Using temperature: %.1f", p.CurrentTemperature)
	} else {
		util.DebugLog("Temperature ignored for model %s", p.CurrentModel)
	}

	// 将请求体转换为 JSON
	requestJSON, err := json.Marshal(requestBody)
	util.DebugLog("Request JSON: %s", string(requestJSON))
	if err != nil {
		return "", fmt.Errorf("error marshaling request: %v", err)
	}

	// 创建 HTTP 请求
	req, err := http.NewRequest("POST", openaiAPIURL, bytes.NewBuffer(requestJSON))
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
		return "", fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应体
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response: %v", err)
	}

	// 解析响应
	var chatResp openaiChatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return "", fmt.Errorf("error unmarshaling response: %v", err)
	}

	// 检查是否有错误
	if chatResp.Error != nil {
		return "", fmt.Errorf("API error: %s", chatResp.Error.Message)
	}

	// 检查是否有响应
	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("no response from API")
	}

	// 返回响应内容
	return chatResp.Choices[0].Message.Content, nil
}

// SendStreamingChatRequest sends a streaming chat request to the OpenAI API
func (p *OpenAIProvider) SendStreamingChatRequest(messages []ChatMessage) (<-chan StreamResponse, error) {
	respChan := make(chan StreamResponse)

	// 检查 API Key 是否已设置
	if p.APIKey == "" {
		return nil, fmt.Errorf("API key not set for OpenAI provider")
	}

	// 确保模型已设置，如果未设置则使用默认模型
	if p.CurrentModel == "" {
		p.CurrentModel = openaiDefaultModel
		fmt.Printf("WARNING: Model not set for OpenAI provider, using default model: %s\n", openaiDefaultModel)
	}

	// 输出调试信息
	util.DebugLog("Using OpenAI model: %s (streaming)", p.CurrentModel)

	// 创建请求体
	requestBody := openaiChatRequest{
		Model:    p.CurrentModel,
		Messages: messages,
		Stream:   true,
	}

	// Only set temperature for models that support it
	if p.CurrentModel != "o1" && p.CurrentModel != "o3-mini" {
		requestBody.Temperature = p.CurrentTemperature
		util.DebugLog("Using temperature: %.1f", p.CurrentTemperature)
	} else {
		util.DebugLog("Temperature ignored for model %s", p.CurrentModel)
	}

	// 将请求体转换为 JSON
	requestJSON, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %v", err)
	}

	// 创建 HTTP 请求
	req, err := http.NewRequest("POST", openaiAPIURL, bytes.NewBuffer(requestJSON))
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
		return nil, fmt.Errorf("error sending request: %v", err)
	}

	// 检查状态码
	if resp.StatusCode != http.StatusOK {
		// 读取错误响应
		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		// 尝试解析错误响应
		var errorResp openaiChatResponse
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
				util.DebugLog("OpenAI stream line: %s", string(line))
			}

			// Parse the response
			var streamResp openaiChatResponse
			if err := json.Unmarshal(line, &streamResp); err != nil {
				if util.IsDebugMode() {
					util.DebugLog("Error parsing OpenAI stream: %v (line: %s)", err, string(line))
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
func (p *OpenAIProvider) SetCurrentModel(model string) error {
	// 验证模型是否有效
	valid := false
	for _, m := range openaiAvailableModels {
		if m == model {
			valid = true
			break
		}
	}

	if !valid {
		fmt.Printf("WARNING: Invalid model: %s. Available models: %v\n", model, openaiAvailableModels)
		return fmt.Errorf("invalid model: %s. Available models: %v", model, openaiAvailableModels)
	}

	// 设置模型并输出调试信息
	p.CurrentModel = model
	util.DebugLog("OpenAI model set to: %s", model)
	return nil
}

// LoadConfig loads the provider configuration from the given map
func (p *OpenAIProvider) LoadConfig(config map[string]interface{}) error {
	// 加载 API Key
	if apiKey, ok := config["api_key"].(string); ok {
		p.APIKey = apiKey
		util.DebugLog("Loaded API key for OpenAI provider")
	}

	// 加载当前模型
	if model, ok := config["model"].(string); ok {
		util.DebugLog("Found model in config: %s", model)
		if err := p.SetCurrentModel(model); err != nil {
			// 如果模型无效，使用默认模型
			fmt.Printf("WARNING: Invalid model in config, using default model: %s\n", openaiDefaultModel)
			p.CurrentModel = openaiDefaultModel
		}
	} else {
		// 如果没有设置模型，使用默认模型
		util.DebugLog("No model found in config, using default model: %s", openaiDefaultModel)
		p.CurrentModel = openaiDefaultModel
	}

	// 加载温度设置
	if temp, ok := config["temperature"].(float64); ok {
		if err := p.SetCurrentTemperature(temp); err != nil {
			// 如果温度无效，使用默认温度
			p.CurrentTemperature = openaiDefaultTemperature
		}
	} else {
		// 如果没有设置温度，使用默认温度
		p.CurrentTemperature = openaiDefaultTemperature
	}

	return nil
}

// SaveConfig saves the provider configuration to the given map
func (p *OpenAIProvider) SaveConfig(config map[string]interface{}) {
	// 保存 API Key
	config["api_key"] = p.APIKey

	// 确保模型已设置，如果未设置则使用默认模型
	if p.CurrentModel == "" {
		p.CurrentModel = openaiDefaultModel
		fmt.Printf("WARNING: Model not set when saving config, using default model: %s\n", openaiDefaultModel)
	}

	// 保存当前模型
	config["model"] = p.CurrentModel
	util.DebugLog("Saving OpenAI model to config: %s", p.CurrentModel)

	// 保存温度设置
	config["temperature"] = p.CurrentTemperature
}

// IsReady returns whether the provider is ready to use
// For OpenAI, the provider is ready if the API key is set
func (p *OpenAIProvider) IsReady() bool {
	return p.APIKey != ""
}

func init() {
	// Register the OpenAI provider
	Register("openai", NewOpenAIProvider)
}
