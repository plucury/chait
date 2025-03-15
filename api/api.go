package api

import (
	"fmt"

	"github.com/plucury/chait/api/provider"
	"github.com/plucury/chait/util"
	"github.com/spf13/viper"
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

var activeProvider provider.Provider

// Initialize default values from the default provider
func init() {
	// Try to load active provider from configuration
	providerName := DefaultProvider
	if viper.IsSet("active_provider") {
		configProvider := viper.GetString("active_provider")
		if _, exists := provider.GetProvider(configProvider); exists {
			providerName = configProvider
			util.DebugLog("Loaded active provider from config: %s", providerName)
		} else {
			util.DebugLog("Provider from config not found: %s, using default: %s", configProvider, DefaultProvider)
		}
	}

	// Initialize active provider
	var exists bool
	activeProvider, exists = provider.GetProvider(providerName)
	if !exists {
		panic("Default provider not found")
	}

	// Initialize default values
	DefaultModel = activeProvider.GetDefaultModel()
	AvailableModels = activeProvider.GetAvailableModels()
	DefaultTemperature = activeProvider.GetDefaultTemperature()
	TemperaturePresets = activeProvider.GetTemperaturePresets()
}

func LoadProviderConfig(providerName string, config map[string]interface{}) error {
	util.DebugLog("Loading configuration for provider: %s", providerName)
	p, exists := provider.GetProvider(providerName)
	if !exists {
		return fmt.Errorf("provider %s not found", providerName)
	}

	if err := p.LoadConfig(config); err != nil {
		util.DebugLog("Error loading configuration for provider %s: %v", providerName, err)
		return err
	}
	util.DebugLog("Successfully loaded configuration for provider: %s", providerName)
	return nil
}

func SaveProviderConfig(providerName string, config map[string]interface{}) error {
	p, exists := provider.GetProvider(providerName)
	if !exists {
		return fmt.Errorf("provider %s not found", providerName)
	}

	p.SaveConfig(config)
	return nil
}

func GetActiveProvider() provider.Provider {
	return activeProvider
}

func GetActiveProviderName() string {
	return activeProvider.GetName()
}

func GetCurrentModel() string {
	return activeProvider.GetCurrentModel()
}

func GetCurrentAvailableModels() []string {
	return activeProvider.GetAvailableModels()
}

func GetCurrentTemperature() float64 {
	return activeProvider.GetCurrentTemperature()
}

func GetCurrentTemperaturePresets() []TemperaturePreset {
	return activeProvider.GetTemperaturePresets()
}

// SetAPIKey sets the API key for the active provider and saves it to the configuration
func SetAPIKey(apiKey string) error {
	err := activeProvider.SetAPIKey(apiKey)
	if err != nil {
		return err
	}

	viper.Set(fmt.Sprintf("providers.%s.api_key", activeProvider.GetName()), apiKey)

	// Write to the configuration file
	if err := viper.WriteConfig(); err != nil {
		util.DebugLog("Error persisting API key to config: %v", err)
		// Don't return error as the provider was successfully set in memory
		// Just log the error for debugging purposes
	}
	return nil
}

func SetActiveProvider(providerName string) error {
	util.DebugLog("Setting active provider to: %s", providerName)
	p, exists := provider.GetProvider(providerName)
	if !exists {
		util.DebugLog("Provider not found: %s", providerName)
		return fmt.Errorf("provider %s not found", providerName)
	}

	activeProvider = p
	util.DebugLog("Active provider set to: %s", providerName)

	// Persist the active provider to configuration
	viper.Set("provider", providerName)

	// Write to the configuration file
	if err := viper.WriteConfig(); err != nil {
		util.DebugLog("Error persisting active provider to config: %v", err)
		// Don't return error as the provider was successfully set in memory
		// Just log the error for debugging purposes
	}

	return nil
}

func SetProviderModel(provider provider.Provider, model string) error {
	err := provider.SetCurrentModel(model)
	if err != nil {
		return fmt.Errorf("failed to set model for provider %s: %v", provider.GetName(), err)
	}
	viper.Set(fmt.Sprintf("providers.%s.model", provider.GetName()), model)
	// Write to the configuration file
	if err := viper.WriteConfig(); err != nil {
		util.DebugLog("Error persisting active provider to config: %v", err)
		// Don't return error as the provider was successfully set in memory
		// Just log the error for debugging purposes
	}
	return nil
}

func SetProviderTemperature(provider provider.Provider, temperature float64) error {
	err := provider.SetCurrentTemperature(temperature)
	if err != nil {
		return fmt.Errorf("failed to set temperature for provider %s: %v", provider.GetName(), err)
	}
	viper.Set(fmt.Sprintf("providers.%s.temperature", provider.GetName()), temperature)
	// Write to the configuration file
	if err := viper.WriteConfig(); err != nil {
		util.DebugLog("Error persisting active provider to config: %v", err)
		// Don't return error as the provider was successfully set in memory
		// Just log the error for debugging purposes
	}
	return nil
}

// SendStreamingChatRequest 发送流式聊天请求到当前活跃的 provider
// 返回一个通道，用于接收流式响应
func SendStreamingChatRequest(messages []ChatMessage) (<-chan provider.StreamResponse, error) {
	util.DebugLog("Sending streaming chat request to provider: %s", activeProvider.GetName())

	// 发送流式请求
	util.DebugLog("Sending streaming request to %s with %d messages", activeProvider.GetName(), len(messages))
	return activeProvider.SendStreamingChatRequest(messages)
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
