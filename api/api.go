package api

import (
	"fmt"

	"github.com/plucury/chait/api/provider"
	"github.com/plucury/chait/util"
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
	util.DebugLog("Loading configuration for provider: %s", providerName)
	p, exists := provider.GetProvider(providerName)
	if !exists {
		return fmt.Errorf("provider %s not found", providerName)
	}

	// 加载配置
	if err := p.LoadConfig(config); err != nil {
		util.DebugLog("Error loading configuration for provider %s: %v", providerName, err)
		return err
	}
	util.DebugLog("Successfully loaded configuration for provider: %s", providerName)
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
	util.DebugLog("Setting active provider to: %s", providerName)
	p, exists := provider.GetProvider(providerName)
	if !exists {
		util.DebugLog("Provider not found: %s", providerName)
		return fmt.Errorf("provider %s not found", providerName)
	}

	activeProvider = p
	util.DebugLog("Active provider set to: %s", providerName)
	return nil
}

// SendChatRequest 发送聊天请求到当前活跃的 provider
// 这个函数保持向后兼容性
func SendChatRequest(apiKey string, messages []ChatMessage, model string, temperature float64) (string, error) {
	util.DebugLog("Sending chat request to provider: %s", activeProvider.GetName())
	
	// 如果提供了 API key，设置到 provider 中
	if apiKey != "" {
		util.DebugLog("Setting API key for provider: %s", activeProvider.GetName())
		activeProvider.SetAPIKey(apiKey)
	}

	// 如果提供了模型，设置到 provider 中
	if model != "" {
		util.DebugLog("Setting model for provider %s: %s", activeProvider.GetName(), model)
		if err := activeProvider.SetCurrentModel(model); err != nil {
			util.DebugLog("Error setting model for provider %s: %v", activeProvider.GetName(), err)
			return "", err
		}
	}

	// 如果提供了温度，设置到 provider 中
	if temperature != 0 {
		util.DebugLog("Setting temperature for provider %s: %.1f", activeProvider.GetName(), temperature)
		if err := activeProvider.SetCurrentTemperature(temperature); err != nil {
			util.DebugLog("Error setting temperature for provider %s: %v", activeProvider.GetName(), err)
			return "", err
		}
	}

	// 发送请求
	util.DebugLog("Sending request to %s with %d messages", activeProvider.GetName(), len(messages))
	return activeProvider.SendChatRequest(messages)
}

// GetAvailableProviders 返回所有可用的 provider 实例
func GetAvailableProviders() []provider.Provider {
	// 直接返回 provider 实例列表
	return provider.GetAvailableProviders()
}

// GetAvailableProviderNames 返回可用的 provider 名称列表
func GetAvailableProviderNames() []string {
	// 获取所有可用的 provider 名称
	return provider.GetAvailableProviderNames()
}

// GetProvider 根据名称返回 provider
func GetProvider(name string) (provider.Provider, bool) {
	return provider.GetProvider(name)
}

// GetReadyProviders 返回所有就绪的 provider 列表
func GetReadyProviders() []provider.Provider {
	util.DebugLog("Getting list of ready providers")
	var readyProviders []provider.Provider
	providers := GetAvailableProviders()

	for _, p := range providers {
		if p.IsReady() {
			util.DebugLog("Provider ready: %s", p.GetName())
			readyProviders = append(readyProviders, p)
		} else {
			util.DebugLog("Provider not ready: %s", p.GetName())
		}
	}

	util.DebugLog("Found %d ready providers", len(readyProviders))
	return readyProviders
}
