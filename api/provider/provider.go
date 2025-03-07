package provider

import (
	"fmt"
	
	"github.com/plucury/chait/util"
)

// ChatMessage represents a message in the chat
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// TemperaturePreset represents a predefined temperature setting for specific use cases
type TemperaturePreset struct {
	Name        string
	Value       float64
	Description string
}

// StreamResponse represents a streaming response chunk from the API
type StreamResponse struct {
	Content string
	Done    bool
	Error   error
}

// Provider defines the interface for AI chat providers
type Provider interface {
	// GetName returns the name of the provider
	GetName() string

	// GetDefaultModel returns the default model for this provider
	GetDefaultModel() string

	// GetAvailableModels returns the list of available models for this provider
	GetAvailableModels() []string

	// GetDefaultTemperature returns the default temperature for this provider
	GetDefaultTemperature() float64

	// GetTemperaturePresets returns the available temperature presets for this provider
	GetTemperaturePresets() []TemperaturePreset

	// GetCurrentModel returns the currently selected model
	GetCurrentModel() string

	// SetCurrentModel sets the current model
	SetCurrentModel(model string) error

	// GetCurrentTemperature returns the currently set temperature
	GetCurrentTemperature() float64

	// SetCurrentTemperature sets the current temperature
	SetCurrentTemperature(temp float64) error

	// GetAPIKey returns the API key (masked for security)
	GetAPIKey() string

	// SetAPIKey sets the API key
	SetAPIKey(apiKey string) error

	// IsReady returns whether the provider is ready to use
	IsReady() bool

	// SendChatRequest sends a chat request to the provider's API
	SendChatRequest(messages []ChatMessage) (string, error)

	// SendStreamingChatRequest sends a chat request and returns a channel for streaming responses
	SendStreamingChatRequest(messages []ChatMessage) (<-chan StreamResponse, error)

	// LoadConfig loads the provider configuration from the given map
	LoadConfig(config map[string]interface{}) error

	// SaveConfig saves the provider configuration to the given map
	SaveConfig(config map[string]interface{})
}

// BaseProvider implements common functionality for all providers
type BaseProvider struct {
	Name               string
	APIKey             string
	CurrentModel       string
	CurrentTemperature float64
}

// GetAPIKey returns a masked version of the API key for security
func (p *BaseProvider) GetAPIKey() string {
	if p.APIKey == "" {
		return ""
	}

	// Mask the API key for security (show only first 4 and last 4 characters)
	if len(p.APIKey) <= 8 {
		return "****"
	}

	return p.APIKey[:4] + "****" + p.APIKey[len(p.APIKey)-4:]
}

// SetAPIKey sets the API key
func (p *BaseProvider) SetAPIKey(apiKey string) error {
	p.APIKey = apiKey
	return nil
}

// GetCurrentModel returns the currently selected model
func (p *BaseProvider) GetCurrentModel() string {
	return p.CurrentModel
}

// SetCurrentModel sets the current model
func (p *BaseProvider) SetCurrentModel(model string) error {
	// This should be overridden by providers to validate the model
	p.CurrentModel = model
	return nil
}

// GetCurrentTemperature returns the currently set temperature
func (p *BaseProvider) GetCurrentTemperature() float64 {
	return p.CurrentTemperature
}

// SetCurrentTemperature sets the current temperature
func (p *BaseProvider) SetCurrentTemperature(temp float64) error {
	// Validate temperature range (common for most providers)
	if temp < 0 || temp > 2.0 {
		return fmt.Errorf("temperature must be between 0.0 and 2.0")
	}

	p.CurrentTemperature = temp
	return nil
}

// Default temperature presets for all providers
var DefaultTemperaturePresets = []TemperaturePreset{
	{"Precise", 0.0, "Highly deterministic responses for factual queries"},
	{"Balanced", 0.7, "Good balance between creativity and coherence"},
	{"Creative", 1.0, "More varied and creative responses"},
	{"Very Creative", 1.5, "Highly varied and potentially more unexpected responses"},
}

// GetTemperaturePresets returns the default temperature presets
// This should be overridden by providers that have specific presets
func (p *BaseProvider) GetTemperaturePresets() []TemperaturePreset {
	return DefaultTemperaturePresets
}

// IsReady returns whether the provider is ready to use
// By default, a provider is ready if it has an API key set
func (p *BaseProvider) IsReady() bool {
	return p.APIKey != ""
}

// Factory is a function that creates a provider instance
type Factory func() Provider

// registry of available providers
var providers = make(map[string]Factory)

// 缓存已创建的 provider 实例
var providerInstances = make(map[string]Provider)

// Register adds a provider factory to the registry
func Register(name string, factory Factory) {
	providers[name] = factory
}

// GetProvider returns a provider by name
func GetProvider(name string) (Provider, bool) {
	// 首先检查是否已经有缓存的实例
	if instance, ok := providerInstances[name]; ok {
		return instance, true
	}

	// 如果没有缓存的实例，则创建一个新的实例
	factory, exists := providers[name]
	if !exists {
		return nil, false
	}
	// debug
	util.DebugLog("Creating new provider instance for %s", name)

	// 创建新实例并缓存
	instance := factory()
	providerInstances[name] = instance

	return instance, true
}

// GetAvailableProviders returns the list of available provider instances
func GetAvailableProviders() []Provider {
	var providerList []Provider
	for name := range providers {
		// Get or create the provider instance
		instance, exists := GetProvider(name)
		if exists {
			providerList = append(providerList, instance)
		}
	}
	return providerList
}

// GetAvailableProviderNames returns the list of available provider names
func GetAvailableProviderNames() []string {
	var names []string
	for name := range providers {
		names = append(names, name)
	}
	return names
}
