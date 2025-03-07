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

// DeepseekProvider implements the Provider interface for Deepseek API
type DeepseekProvider struct {
	BaseProvider // 嵌入基础提供者结构体
}

const (
	deepseekAPIURL     = "https://api.deepseek.com/v1/chat/completions"
	deepseekDefaultModel = "deepseek-chat"
	deepseekDefaultTemperature = 1.0
)

// Available models for Deepseek API
var deepseekAvailableModels = []string{
	"deepseek-chat",
	"deepseek-reasoner",
}

// Available temperature presets for Deepseek API
var deepseekTemperaturePresets = []TemperaturePreset{
	{"Code Generation", 0.0, "Code generation or math problem solving"},
	{"Data Extraction", 1.0, "Data extraction and analysis"},
	{"General Conversation", 1.3, "General conversation"},
	{"Translation", 1.3, "Translation tasks"},
	{"Creative Writing", 1.5, "Creative writing or poetry"},
}

// NewDeepseekProvider creates a new instance of DeepseekProvider
func NewDeepseekProvider() Provider {
	provider := &DeepseekProvider{
		BaseProvider: BaseProvider{
			Name:               "deepseek",
			CurrentModel:       deepseekDefaultModel,
			CurrentTemperature: deepseekDefaultTemperature,
		},
	}
	return provider
}

// GetName returns the name of the provider
func (p *DeepseekProvider) GetName() string {
	return p.Name
}

// GetDefaultModel returns the default model for this provider
func (p *DeepseekProvider) GetDefaultModel() string {
	return deepseekDefaultModel
}

// GetAvailableModels returns the list of available models for this provider
func (p *DeepseekProvider) GetAvailableModels() []string {
	return deepseekAvailableModels
}

// GetDefaultTemperature returns the default temperature for this provider
func (p *DeepseekProvider) GetDefaultTemperature() float64 {
	return deepseekDefaultTemperature
}

// GetTemperaturePresets returns the available temperature presets for this provider
func (p *DeepseekProvider) GetTemperaturePresets() []TemperaturePreset {
	return deepseekTemperaturePresets
}

// SetCurrentTemperature sets the current temperature with Deepseek-specific validation
func (p *DeepseekProvider) SetCurrentTemperature(temp float64) error {
	// Validate temperature range specific to Deepseek (0-2)
	if temp < 0 || temp > 2.0 {
		return fmt.Errorf("Deepseek temperature must be between 0.0 and 2.0")
	}

	p.CurrentTemperature = temp
	return nil
}

// chatRequest represents the request to the Deepseek chat API
type chatRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	Temperature float64       `json:"temperature,omitempty"`
	Stream      bool          `json:"stream,omitempty"`
}

// chatResponse represents the response from the Deepseek chat API
type chatResponse struct {
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
	Error *errorResponse `json:"error,omitempty"`
}

// errorResponse represents an error from the Deepseek API
type errorResponse struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Param   string `json:"param"`
	Code    string `json:"code"`
}

// SendChatRequest sends a chat request to the Deepseek API
func (p *DeepseekProvider) SendChatRequest(messages []ChatMessage) (string, error) {
	// 检查 API Key 是否已设置
	if p.APIKey == "" {
		return "", fmt.Errorf("API key not set for Deepseek provider")
	}

	// 创建请求体
	requestBody := chatRequest{
		Model:       p.CurrentModel,
		Messages:    messages,
		Temperature: p.CurrentTemperature,
	}
	
	util.DebugLog("Using Deepseek model: %s", p.CurrentModel)
	util.DebugLog("Using temperature: %.1f", p.CurrentTemperature)

	// 将请求体序列化为 JSON
	jsonData, err := json.Marshal(requestBody)
	util.DebugLog("Request JSON: %s", string(jsonData))
	if err != nil {
		return "", fmt.Errorf("error marshaling request: %v", err)
	}

	// 创建 HTTP 请求
	req, err := http.NewRequest("POST", deepseekAPIURL, bytes.NewBuffer(jsonData))
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
			case errorResp.Error.Message == "Insufficient Balance" || errorResp.Error.Code == "invalid_request_error":
				return "", fmt.Errorf("Deepseek API account has insufficient balance. Please check your account or contact Deepseek support.")
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
	var chatResp chatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return "", fmt.Errorf("error parsing response: %v", err)
	}

	// 检查是否有选择
	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("no response choices returned")
	}

	// 返回第一个选择的内容
	return chatResp.Choices[0].Message.Content, nil
}

// SendStreamingChatRequest sends a streaming chat request to the Deepseek API
func (p *DeepseekProvider) SendStreamingChatRequest(messages []ChatMessage) (<-chan StreamResponse, error) {
	respChan := make(chan StreamResponse)

	// 检查 API Key 是否已设置
	if p.APIKey == "" {
		return nil, fmt.Errorf("API key not set for Deepseek provider")
	}

	// 创建请求体
	requestBody := chatRequest{
		Model:       p.CurrentModel,
		Messages:    messages,
		Temperature: p.CurrentTemperature,
		Stream:      true,
	}

	util.DebugLog("Using Deepseek model: %s (streaming)", p.CurrentModel)
	util.DebugLog("Using temperature: %.1f", p.CurrentTemperature)

	// 将请求体序列化为 JSON
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %v", err)
	}

	// 创建 HTTP 请求
	req, err := http.NewRequest("POST", deepseekAPIURL, bytes.NewBuffer(jsonData))
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
		var errorResp chatResponse
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
				util.DebugLog("Deepseek stream line: %s", string(line))
			}

			// Parse the response
			var streamResp chatResponse
			if err := json.Unmarshal(line, &streamResp); err != nil {
				if util.IsDebugMode() {
					util.DebugLog("Error parsing Deepseek stream: %v (line: %s)", err, string(line))
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
func (p *DeepseekProvider) SetCurrentModel(model string) error {
	// 验证模型是否有效
	valid := false
	for _, m := range deepseekAvailableModels {
		if m == model {
			valid = true
			break
		}
	}

	if !valid {
		return fmt.Errorf("invalid model: %s. Available models: %v", model, deepseekAvailableModels)
	}

	p.CurrentModel = model
	util.DebugLog("Deepseek model set to: %s", model)
	return nil
}

// LoadConfig loads the provider configuration from the given map
func (p *DeepseekProvider) LoadConfig(config map[string]interface{}) error {
	// 加载 API Key
	if apiKey, ok := config["api_key"].(string); ok {
		p.APIKey = apiKey
		util.DebugLog("Loaded API key for Deepseek provider")
	}

	// 加载当前模型
	if model, ok := config["model"].(string); ok {
		util.DebugLog("Found model in config: %s", model)
		if err := p.SetCurrentModel(model); err != nil {
			// 如果模型无效，使用默认模型
			p.CurrentModel = deepseekDefaultModel
		}
	} else {
		// 如果没有设置模型，使用默认模型
		util.DebugLog("No model found in config, using default model: %s", deepseekDefaultModel)
		p.CurrentModel = deepseekDefaultModel
	}

	// 加载温度设置
	if temp, ok := config["temperature"].(float64); ok {
		if err := p.SetCurrentTemperature(temp); err != nil {
			// 如果温度无效，使用默认温度
			p.CurrentTemperature = deepseekDefaultTemperature
		}
	} else {
		// 如果没有设置温度，使用默认温度
		p.CurrentTemperature = deepseekDefaultTemperature
	}

	return nil
}

// SaveConfig saves the provider configuration to the given map
func (p *DeepseekProvider) SaveConfig(config map[string]interface{}) {
	// 保存 API Key
	config["api_key"] = p.APIKey
	
	// 保存当前模型
	config["model"] = p.CurrentModel
	util.DebugLog("Saving Deepseek model to config: %s", p.CurrentModel)
	
	// 保存温度设置
	config["temperature"] = p.CurrentTemperature
}

// IsReady returns whether the provider is ready to use
// For Deepseek, the provider is ready if the API key is set
func (p *DeepseekProvider) IsReady() bool {
	return p.APIKey != ""
}



func init() {
	// Register the Deepseek provider
	Register("deepseek", NewDeepseekProvider)
}
