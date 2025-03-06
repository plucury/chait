package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/chzyer/readline"

	"github.com/plucury/chait/api"
	"github.com/plucury/chait/api/provider"
	"github.com/plucury/chait/util"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

// Version represents the current version of the application
const Version = "0.0.1"

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "chait",
	Short: "A AI chat command-line tool",
	Long:  `A AI chat command-line tool built with Cobra. support providers: openai, deepseek`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Skip loading provider configurations for version command
		if showVersion {
			return
		}

		// Load all provider configurations
		loadProviderConfigurations()
		DebugLog("Loaded provider configurations")
	},
	Run: func(cmd *cobra.Command, args []string) {
		// Check if we need to display the version information
		if showVersion {
			fmt.Printf("chait version %s\n", Version)
			return
		}

		// Check if we need to interactively select a provider
		if selectProvider {
			if err := configureProvider(); err != nil {
				fmt.Printf("Error configuring provider: %v\n", err)
				return
			}
			fmt.Println("Provider configured successfully. Run 'chait' to start interactive mode.")
			return
		}

		// Get the currently used provider from configuration
		providerName := viper.GetString("provider")

		// If no provider is configured, prompt the user to select one
		if providerName == "" {
			fmt.Println("No provider selected. Let's choose one.")
			// Prompt the user to select and configure a provider
			if err := configureProvider(); err != nil {
				fmt.Printf("Error configuring provider: %v\n", err)
				return
			}

			// Get the currently used provider from configuration again
			providerName = viper.GetString("provider")
			if providerName == "" {
				// If still empty, use the default value
				providerName = api.DefaultProvider
			}
		}

		// Load provider configuration
		providerConfig := viper.GetStringMap(fmt.Sprintf("providers.%s", providerName))

		// Convert viper configuration to map[string]interface{}
		config := make(map[string]interface{})
		for k, v := range providerConfig {
			config[k] = v
		}

		// Load provider configuration
		DebugLog("Loading provider configuration for %s", providerName)
		if err := api.LoadProviderConfig(providerName, config); err != nil {
			fmt.Printf("Error loading provider config: %v\n", err)
			return
		}
		DebugLog("Successfully loaded provider configuration for %s", providerName)

		// Get all ready providers
		readyProviders := api.GetReadyProviders()

		// Check if there are any available providers
		if len(readyProviders) == 0 {
			fmt.Println("No ready providers found. Let's configure one.")
			// Prompt the user to select and configure a provider
			if err := configureProvider(); err != nil {
				fmt.Printf("Error configuring provider: %v\n", err)
				return
			}

			// Get ready providers again
			readyProviders = api.GetReadyProviders()
			if len(readyProviders) == 0 {
				fmt.Println("Still no ready providers. Exiting.")

				// Debug information
				DebugLog("Checking provider status...")
				providers := api.GetAvailableProviders()
				for _, p := range providers {
					DebugLog("Provider %s exists, IsReady: %v, API Key set: %v",
						p.GetName(), p.IsReady(), p.GetAPIKey() != "")
				}
				return
			}
		}

		// Get the active provider
		provider := api.GetActiveProvider()

		// Check if the current active provider is ready
		if !provider.IsReady() {
			// If the current active provider is not ready, but there are other ready providers, switch to the first ready provider
			if err := api.SetActiveProvider(readyProviders[0].GetName()); err != nil {
				fmt.Printf("Error setting active provider: %v\n", err)
				return
			}
			provider = readyProviders[0]
			fmt.Printf("Switched to ready provider: %s\n", provider.GetName())
		}

		// API key is set, enter interactive mode
		startInteractiveMode()
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

// Whether to display the version information
var showVersion bool

// Whether to interactively select a provider
var selectProvider bool

// configureProvider prompts the user to select and configure a provider
func configureProvider() error {
	// Create an input reader
	reader := bufio.NewReader(os.Stdin)

	// Get all available providers
	providers := api.GetAvailableProviders()
	if len(providers) == 0 {
		return fmt.Errorf("no available providers found")
	}

	// Get provider names for selection
	var providerNames []string
	for _, p := range providers {
		providerNames = append(providerNames, p.GetName())
	}

	// Display the list of available providers
	fmt.Println("Available providers:")
	for i, p := range providers {
		readyStatus := "not ready"
		if p.IsReady() {
			readyStatus = "ready"
		}
		fmt.Printf("  %d. %s (%s)\n", i+1, p.GetName(), readyStatus)
		fmt.Printf("     Available models: %s\n", strings.Join(p.GetAvailableModels(), ", "))
	}

	// Prompt the user to select a provider
	fmt.Print("\nSelect a provider (enter number): ")
	choiceStr, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("error reading input: %v", err)
	}

	// Process the input
	choiceStr = strings.TrimSpace(choiceStr)
	choice, err := strconv.Atoi(choiceStr)
	if err != nil || choice < 1 || choice > len(providers) {
		return fmt.Errorf("invalid choice: %v", err)
	}

	// Get the selected provider
	selectedProvider := providers[choice-1]
	providerName := selectedProvider.GetName()

	// Set as the active provider
	if err := api.SetActiveProvider(providerName); err != nil {
		return fmt.Errorf("error setting active provider: %v", err)
	}

	// Save the selected provider to the configuration file
	viper.Set("provider", providerName)

	// Write to the configuration file
	if err := viper.WriteConfig(); err != nil {
		fmt.Printf("Error saving provider setting: %v\n", err)
	}

	// Load provider configuration
	providerConfig := viper.GetStringMap(fmt.Sprintf("providers.%s", providerName))
	// print config detail for debugging
	config := make(map[string]interface{})
	for k, v := range providerConfig {
		config[k] = v
	}

	// Load provider configuration
	if err := api.LoadProviderConfig(providerName, config); err != nil {
		return fmt.Errorf("error loading provider config: %v", err)
	}

	// Check if the API key is already set
	if selectedProvider.GetAPIKey() == "" {
		// Prompt the user to enter an API key
		fmt.Printf("Enter API key for %s: ", providerName)
		apiKeyStr, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("error reading API key: %v", err)
		}

		// Process the input
		apiKey := strings.TrimSpace(apiKeyStr)

		// Set the API key
		selectedProvider.SetAPIKey(apiKey)

		// Save provider configuration
		config := make(map[string]interface{})
		selectedProvider.SaveConfig(config)

		// Save to viper
		for k, v := range config {
			viper.Set(fmt.Sprintf("providers.%s.%s", providerName, k), v)
		}

		// Write to the configuration file
		if err := viper.WriteConfig(); err != nil {
			return fmt.Errorf("error saving API key: %v", err)
		}

		// Reload configuration to ensure the API key takes effect
		if err := api.LoadProviderConfig(providerName, config); err != nil {
			return fmt.Errorf("error reloading provider config: %v", err)
		}

		fmt.Printf("%s API key set successfully!\n", providerName)
	}

	// Final check if the provider is ready
	if !selectedProvider.IsReady() {
		fmt.Printf("WARNING: Provider %s is still not ready after configuration.\n", providerName)
		fmt.Println("Please check your API key and try again.")
	} else {
		// start interactive mode
		startInteractiveMode()
	}

	return nil
}

// loadProviderConfigurations loads all provider configurations from the config file
func loadProviderConfigurations() {
	// Get all available providers
	providers := api.GetAvailableProviders()

	// Load configuration for each provider
	for _, p := range providers {
		providerName := p.GetName()
		providerConfig := viper.GetStringMap(fmt.Sprintf("providers.%s", providerName))

		// Convert viper configuration to map[string]interface{}
		config := make(map[string]interface{})
		for k, v := range providerConfig {
			config[k] = v
		}

		// Load provider configuration
		if err := api.LoadProviderConfig(providerName, config); err != nil {
			fmt.Printf("Warning: Error loading configuration for provider %s: %v\n", providerName, err)
		}
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Add version flag
	rootCmd.Flags().BoolVarP(&showVersion, "version", "v", false, "Display the current version of chait")
	// Add provider selection flag
	rootCmd.Flags().BoolVarP(&selectProvider, "provider", "p", false, "Interactively select a provider")

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.config/chait/config.json)")
}

// Interactive mode function
// displayHelpCommands prints all available interactive mode commands
func displayHelpCommands() {
	fmt.Println("Available commands:")
	fmt.Println("  :help, :h        - Show this help message")
	fmt.Println("  :clear, :c       - Start a new conversation")
	fmt.Println("  :model           - Switch between available models")
	fmt.Println("  :temperature, :temp - Set the temperature parameter")
	fmt.Println("  :provider        - Configure or switch provider")
	fmt.Println("  :debug           - Toggle debug mode")
	fmt.Println("  :quit, :q        - Exit the interactive mode")
}

func startInteractiveMode() {
	// Get the active provider
	var provider provider.Provider
	var currentModel string
	var currentTemperature float64

	// Initialize provider and settings
	provider = api.GetActiveProvider()
	currentModel = provider.GetCurrentModel()
	currentTemperature = provider.GetCurrentTemperature()
	fmt.Println("Welcome to chait interactive mode!")
	fmt.Printf("Provider: %s (Model: %s, Temperature: %.1f)\n", provider.GetName(), currentModel, currentTemperature)
	fmt.Println("Type ':help' or ':h' to see all available commands.")
	fmt.Println("-----------------------------------")

	// Use the readline library to handle terminal input, providing better line editing capabilities
	rl, err := readline.New("> ")
	if err != nil {
		fmt.Printf("Error initializing readline: %v\n", err)
		return
	}
	defer rl.Close()

	// Save conversation history
	var messages []api.ChatMessage

	// Add system message
	messages = append(messages, api.ChatMessage{
		Role:    "system",
		Content: "You are a helpful assistant.",
	})

	for {
		// Use readline to read user input, providing better line editing capabilities
		input, err := rl.Readline()
		if err != nil { // io.EOF, readline.ErrInterrupt
			if err.Error() == "Interrupt" {
				fmt.Println("\nUse :quit or :q to exit")
				continue
			}
			break
		}

		// Check if it's a command (starts with a colon)
		if len(input) > 0 && input[0] == ':' {
			// Remove the colon
			cmd := input[1:]

			// Handle exit command
			if cmd == "quit" || cmd == "q" {
				fmt.Println("Goodbye!")
				break
			}

			// Handle help command
			if cmd == "help" || cmd == "h" {
				displayHelpCommands()
				continue
			}

			// Handle clear conversation history command
			if cmd == "clear" || cmd == "c" {
				messages = messages[:1] // Only keep the system message
				fmt.Println("Conversation history cleared.")
				continue
			}

			// Handle provider configuration command
			if cmd == "provider" {
				fmt.Println("Configuring provider...")
				if err := configureProvider(); err != nil {
					fmt.Printf("Error configuring provider: %v\n", err)
				} else {
					// Get the newly configured active provider
					provider = api.GetActiveProvider()
					currentModel = provider.GetCurrentModel()
					currentTemperature = provider.GetCurrentTemperature()
					DebugLog("Switched to provider: %s, model: %s, temperature: %.1f", provider.GetName(), currentModel, currentTemperature)

					// Clear the conversation history when switching providers
					messages = messages[:0] // Clear all messages

					// Add system message back
					messages = append(messages, api.ChatMessage{
						Role:    "system",
						Content: "You are a helpful assistant.",
					})

					fmt.Printf("Provider switched to %s (Model: %s, Temperature: %.1f)\n",
						provider.GetName(), currentModel, currentTemperature)
					fmt.Println("Conversation history cleared.")
				}
				continue
			}

			// Handle debug mode toggle command
			if cmd == "debug" {
				// Get current debug mode status
				currentDebugMode := viper.GetBool("debug")

				// Toggle debug mode
				newDebugMode := !currentDebugMode
				viper.Set("debug", newDebugMode)

				// Save to config file
				if err := viper.WriteConfig(); err != nil {
					fmt.Printf("Error saving debug mode setting: %v\n", err)
				} else {
					if newDebugMode {
						fmt.Println("Debug mode enabled. Debug logs will be displayed.")
						DebugLog("Debug mode enabled")
					} else {
						fmt.Println("Debug mode disabled. Debug logs will not be displayed.")
					}
				}
				continue
			}

			// Handle providers command
			if cmd == "providers" {
				readyProviders := api.GetReadyProviders()
				if len(readyProviders) == 0 {
					fmt.Println("No ready providers found. Please set API keys for providers.")
				} else {
					fmt.Println("Ready providers:")
					for i, p := range readyProviders {
						fmt.Printf("  %d. %s (Model: %s, Temperature: %.1f)\n", i+1, p.GetName(), p.GetCurrentModel(), p.GetCurrentTemperature())
					}
				}
				continue
			}

			// Handle temperature setting command
			if cmd == "temperature" || cmd == "temp" {
				DebugLog("Temperature setting command triggered for provider %s", provider.GetName())
				// Get the current provider
				provider := api.GetActiveProvider()
				currentModel := provider.GetCurrentModel()
				currentTemperature = provider.GetCurrentTemperature()

				// Check if the current model supports temperature settings
				if provider.GetName() == "openai" && (currentModel == "o1" || currentModel == "o3-mini") {
					fmt.Printf("Note: The current model '%s' does not support temperature settings. Temperature will be ignored.\n\n", currentModel)
				}

				// Get provider-specific temperature presets
				providerPresets := provider.GetTemperaturePresets()

				fmt.Printf("Current temperature for %s: %.1f\n\n", provider.GetName(), currentTemperature)
				fmt.Println("Available temperature presets:")

				// Display provider-specific presets if available
				if len(providerPresets) > 0 {
					for i, preset := range providerPresets {
						fmt.Printf("  %d. %s (%.1f) - %s%s\n", i+1, preset.Name, preset.Value, preset.Description, func() string {
							if preset.Value == currentTemperature {
								return " (current)"
							}
							return ""
						}())
					}
				} else {
					// Fall back to generic presets if provider doesn't have specific ones
					for i, preset := range api.TemperaturePresets {
						fmt.Printf("  %d. %s (%.1f) - %s%s\n", i+1, preset.Name, preset.Value, preset.Description, func() string {
							if preset.Value == currentTemperature {
								return " (current)"
							}
							return ""
						}())
					}
				}

				// Determine the max temperature based on the provider
				maxTemp := 2.0
				if provider.GetName() == "openai" {
					maxTemp = 1.0
				}
				fmt.Printf("  C. Custom - Enter a custom temperature value (0.0-%.1f)\n", maxTemp)

				// Use the readline instance to read temperature selection input
				rl.SetPrompt("\nEnter preset number or 'C' for custom (or press Enter to cancel): ")
				tempInput, err := rl.Readline()
				rl.SetPrompt("> ") // Restore the original prompt
				if err != nil {
					fmt.Printf("Error reading input: %v\n", err)
					continue
				}
				if tempInput == "" {
					fmt.Println("Temperature change canceled.")
					continue
				}

				// Handle custom temperature
				if tempInput == "C" || tempInput == "c" {
					// Determine the max temperature based on the provider
					maxTemp := 2.0
					if provider.GetName() == "openai" {
						maxTemp = 1.0
					}

					// Use the readline instance to read custom temperature input
					rl.SetPrompt(fmt.Sprintf("Enter custom temperature (0.0-%.1f): ", maxTemp))
					customTemp, err := rl.Readline()
					rl.SetPrompt("> ") // Restore the original prompt
					if err != nil {
						fmt.Printf("Error reading input: %v\n", err)
						continue
					}
					tempValue, err := strconv.ParseFloat(customTemp, 64)
					if err != nil || tempValue < 0 || tempValue > maxTemp {
						fmt.Printf("Invalid temperature value. Please enter a number between 0.0 and %.1f.\n", maxTemp)
						continue
					}

					// Set the new temperature
					currentTemperature = tempValue
				} else {
					// Handle preset temperature
					tempNum, err := strconv.Atoi(tempInput)

					// Check if we're using provider-specific presets or generic ones
					if len(providerPresets) > 0 {
						if err != nil || tempNum < 1 || tempNum > len(providerPresets) {
							fmt.Println("Invalid preset number. Please try again.")
							continue
						}
						// Set the new temperature from provider-specific presets
						currentTemperature = providerPresets[tempNum-1].Value
					} else {
						if err != nil || tempNum < 1 || tempNum > len(api.TemperaturePresets) {
							fmt.Println("Invalid preset number. Please try again.")
							continue
						}
						// Set the new temperature from generic presets
						currentTemperature = api.TemperaturePresets[tempNum-1].Value
					}
				}

				// Set the provider's temperature
				DebugLog("Setting temperature to %.1f for provider %s", currentTemperature, provider.GetName())
				if err := provider.SetCurrentTemperature(currentTemperature); err != nil {
					fmt.Printf("Error setting temperature: %v\n", err)
					continue
				}

				// 保存 provider 配置
				providerName := provider.GetName()
				config := make(map[string]interface{})
				provider.SaveConfig(config)

				// 保存到 viper
				for k, v := range config {
					viper.Set(fmt.Sprintf("providers.%s.%s", providerName, k), v)
				}

				// 写入配置文件
				if err := viper.WriteConfig(); err != nil {
					fmt.Printf("Error saving temperature setting: %v\n", err)
				} else {
					fmt.Printf("Temperature for %s set to %.1f and saved to config.\n", providerName, currentTemperature)
					DebugLog("Successfully saved temperature %.1f to config for provider %s", currentTemperature, providerName)
				}
				continue
			}

			// Handle model switching command
			if cmd == "model" {
				DebugLog("Model selection command triggered for provider %s", provider.GetName())
				// Get the current provider
				provider := api.GetActiveProvider()
				// Use the currentModel variable already declared externally
				currentModel = provider.GetCurrentModel()

				fmt.Printf("Current model: %s\n\n", currentModel)
				fmt.Println("Available models for provider: " + provider.GetName())

				// Get the available models for the current provider
				DebugLog("Retrieving available models for provider %s", provider.GetName())
				availableModels := provider.GetAvailableModels()
				if len(availableModels) == 0 {
					fmt.Println("No available models found for this provider.")
					continue
				}

				// Display available models
				for i, model := range availableModels {
					fmt.Printf("  %d. %s%s\n", i+1, model, func() string {
						if model == currentModel {
							return " (current)"
						}
						return ""
					}())
				}

				// Use the readline instance to read model selection input
				rl.SetPrompt("\nEnter model number to switch (or press Enter to cancel): ")
				modelInput, err := rl.Readline()
				rl.SetPrompt("> ") // Restore the original prompt
				if err != nil {
					fmt.Printf("Error reading input: %v\n", err)
					continue
				}
				if modelInput == "" {
					fmt.Println("Model switch canceled.")
					continue
				}

				// Convert user input to integer
				modelNum, err := strconv.Atoi(modelInput)
				if err != nil {
					DebugLog("Invalid model selection input: %s", modelInput)
					fmt.Println("Invalid model number. Please try again.")
					continue
				}

				// Save the old model
				oldModel := currentModel

				// Reuse the previously retrieved list of available models

				// Set the new model
				if modelNum < 1 || modelNum > len(availableModels) {
					DebugLog("Model number out of range: %d (valid range: 1-%d)", modelNum, len(availableModels))
					fmt.Println("Invalid model number. Please try again.")
					continue
				}

				newModel := availableModels[modelNum-1]
				DebugLog("Setting model to %s for provider %s", newModel, provider.GetName())
				if err := provider.SetCurrentModel(newModel); err != nil {
					fmt.Printf("Error setting model: %v\n", err)
					continue
				}
				currentModel = newModel

				// 保存 provider 配置
				providerName := provider.GetName()
				config := make(map[string]interface{})
				provider.SaveConfig(config)

				// 保存到 viper
				for k, v := range config {
					viper.Set(fmt.Sprintf("providers.%s.%s", providerName, k), v)
				}

				// 写入配置文件
				if err := viper.WriteConfig(); err != nil {
					fmt.Printf("Error saving model setting: %v\n", err)
				} else {
					DebugLog("Successfully saved model %s to config for provider %s", newModel, providerName)
				}

				// If the model has changed, clear the conversation history
				if oldModel != currentModel {
					messages = messages[:1] // Only keep the system message
					fmt.Printf("Switched to model: %s. Conversation history cleared.\n", currentModel)
				} else {
					fmt.Printf("Already using model: %s\n", currentModel)
				}
				continue
			}

			// If it's an unknown command
			fmt.Printf("Unknown command: %s\n", cmd)
			fmt.Println("Type :help for available commands.")
			continue
		}

		// Process other commands entered by the user
		if input != "" {
			// Add user message to history
			messages = append(messages, api.ChatMessage{
				Role:    "user",
				Content: input,
			})

			// Send request to AI provider
			fmt.Println("Thinking...")
			DebugLog("Sending chat request to provider %s with %d messages", provider.GetName(), len(messages))
			response, err := api.SendChatRequest("", messages, "", 0)
			if err != nil {
				// Handle specific errors
				errMsg := err.Error()
				fmt.Printf("\nError: %v\n\n", errMsg)

				// Check if it's an insufficient balance error
				if strings.Contains(errMsg, "insufficient balance") || strings.Contains(errMsg, "Insufficient Balance") {
					fmt.Println("Your Deepseek API account has insufficient balance.")
					fmt.Println("Please check your account at https://platform.deepseek.com/")
					fmt.Println("You can continue using the CLI with a different API key by running:")
					fmt.Println("  chait config providers.deepseek.api_key YOUR_NEW_API_KEY")
				}

				// Remove the last user message from history because the request failed
				messages = messages[:len(messages)-1]
				continue
			}

			// Print the AI's response
			fmt.Println("\n" + response + "\n")

			// Add AI response to history
			messages = append(messages, api.ChatMessage{
				Role:    "assistant",
				Content: response,
			})

			// If the history is too long, it can be trimmed
			if len(messages) > 20 {
				// Keep the system message and the most recent conversations
				messages = append(messages[:1], messages[len(messages)-19:]...)
			}
		}
	}
}

// IsDebugMode is a wrapper for util.IsDebugMode
func IsDebugMode() bool {
	return util.IsDebugMode()
}

// DebugLog is a wrapper for util.DebugLog
func DebugLog(format string, args ...interface{}) {
	util.DebugLog(format, args...)
}

func initConfig() {
	var configDir string

	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)

		// Ensure the directory for the config file exists
		configDir = filepath.Dir(cfgFile)
		if err := os.MkdirAll(configDir, 0755); err != nil {
			fmt.Printf("Error creating config directory %s: %v\n", configDir, err)
			os.Exit(1)
		}
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Printf("Error finding home directory: %v\n", err)
			os.Exit(1)
		}

		// Set up config in ~/.config/chait directory with name "config.json"
		configDir = filepath.Join(home, ".config", "chait")
		fmt.Printf("Config directory: %s\n", configDir)

		// Create config directory if it doesn't exist
		if err := os.MkdirAll(configDir, 0755); err != nil {
			fmt.Printf("Error creating config directory %s: %v\n", configDir, err)
			os.Exit(1)
		}

		viper.AddConfigPath(configDir)
		viper.SetConfigType("json")
		viper.SetConfigName("config")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found, creating a default one
			fmt.Println("Config file not found, creating default config")

			// Get all available providers
			providers := api.GetAvailableProviders()

			// Create default configuration
			defaultConfig := map[string]interface{}{
				"version":   Version,
				"provider":  "", // Current provider being used, empty string indicates user needs to choose
				"providers": map[string]interface{}{},
				"debug":     false, // Debug mode, when true prints debug logs
			}

			// Create default configuration for each provider
			providersConfig := defaultConfig["providers"].(map[string]interface{})
			for _, p := range providers {
				config := make(map[string]interface{})
				p.SaveConfig(config)
				providersConfig[p.GetName()] = config
			}

			for k, v := range defaultConfig {
				viper.Set(k, v)
			}

			// Determine the config file path
			configFile := viper.ConfigFileUsed()
			if configFile == "" {
				// If viper doesn't have a config file set, create one
				configFile = filepath.Join(configDir, "config.json")
				viper.SetConfigFile(configFile)
			}

			fmt.Printf("Writing default config to: %s\n", configFile)
			if err := viper.WriteConfig(); err != nil {
				fmt.Printf("Error writing default config: %v\n", err)
			} else {
				fmt.Println("Default config created successfully")
			}
		} else {
			fmt.Printf("Error reading config file: %v\n", err)
		}
	}
}
