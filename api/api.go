package api

import (
	"fmt"

	"github.com/plucury/chait/api/provider"
)

// Re-export ChatMessage from provider package
type ChatMessage = provider.ChatMessage

// Re-export TemperaturePreset from provider package
type TemperaturePreset = provider.TemperaturePreset

// DefaultProvider is the default provider name
const DefaultProvider = "deepseek"

// DefaultModel is the default model for the default provider
var DefaultModel string

// AvailableModels is the list of available models for the default provider
var AvailableModels []string

// DefaultTemperature is the default temperature for the default provider
var DefaultTemperature float64

// TemperaturePresets is the list of available temperature presets for the default provider
var TemperaturePresets []TemperaturePreset

// 当前活跃的 provider
var activeProvider provider.Provider

// Initialize default values from the default provider
func init() {
	// 初始化默认 provider
	var exists bool
	activeProvider, exists = provider.GetProvider(DefaultProvider)
	if !exists {
		panic("Default provider not found")
	}

	// 初始化默认值
	DefaultModel = activeProvider.GetDefaultModel()
	AvailableModels = activeProvider.GetAvailableModels()
	DefaultTemperature = activeProvider.GetDefaultTemperature()
	TemperaturePresets = activeProvider.GetTemperaturePresets()
}

// LoadProviderConfig 从配置中加载 provider 配置
func LoadProviderConfig(providerName string, config map[string]interface{}) error {
	p, exists := provider.GetProvider(providerName)
	if !exists {
		return fmt.Errorf("provider %s not found", providerName)
	}

	// 加载配置
	if err := p.LoadConfig(config); err != nil {
		return err
	}
	return nil
}

// SaveProviderConfig 保存 provider 配置到配置中
func SaveProviderConfig(providerName string, config map[string]interface{}) error {
	p, exists := provider.GetProvider(providerName)
	if !exists {
		return fmt.Errorf("provider %s not found", providerName)
	}

	// 保存配置
	p.SaveConfig(config)
	return nil
}

// GetActiveProvider 返回当前活跃的 provider
func GetActiveProvider() provider.Provider {
	return activeProvider
}

// SetActiveProvider 设置当前活跃的 provider
func SetActiveProvider(providerName string) error {
	p, exists := provider.GetProvider(providerName)
	if !exists {
		return fmt.Errorf("provider %s not found", providerName)
	}

	activeProvider = p
	return nil
}

// SendChatRequest 发送聊天请求到当前活跃的 provider
// 这个函数保持向后兼容性
func SendChatRequest(apiKey string, messages []ChatMessage, model string, temperature float64) (string, error) {
	// 如果提供了 API key，设置到 provider 中
	if apiKey != "" {
		activeProvider.SetAPIKey(apiKey)
	}

	// 如果提供了模型，设置到 provider 中
	if model != "" {
		if err := activeProvider.SetCurrentModel(model); err != nil {
			return "", err
		}
	}

	// 如果提供了温度，设置到 provider 中
	if temperature != 0 {
		if err := activeProvider.SetCurrentTemperature(temperature); err != nil {
			return "", err
		}
	}

	// 发送请求
	return activeProvider.SendChatRequest(messages)
}

// GetAvailableProviders 返回可用的 provider 名称列表
func GetAvailableProviders() []string {
	return provider.GetAvailableProviders()
}

// GetProvider 根据名称返回 provider
func GetProvider(name string) (provider.Provider, bool) {
	return provider.GetProvider(name)
}

// GetReadyProviders 返回所有就绪的 provider 列表
func GetReadyProviders() []provider.Provider {
	var readyProviders []provider.Provider
	providerNames := provider.GetAvailableProviders()

	for _, name := range providerNames {
		p, exists := provider.GetProvider(name)
		if exists && p.IsReady() {
			readyProviders = append(readyProviders, p)
		}
	}

	return readyProviders
}
