package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/plucury/chait/api"
	"github.com/plucury/chait/util"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

// Version represents the current version of the application
var version string = "0.2.3"

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "chait",
	Short: "A AI chat command-line tool and more",
	Long:  `A AI chat command-line tool built with Cobra. support providers: openai, deepseek, grok`,
	// Allow arbitrary arguments to be passed
	Args: cobra.ArbitraryArgs,
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
			if version == "" {
				version = "undefined" // 默认版本号
			} else {
				fmt.Printf("chait version %s\n", version)
			}
			return
		}

		// Check if we need to interactively select a provider
		if selectProvider {
			if err := configureProvider(); err != nil {
				fmt.Printf("Error configuring provider: %v\n", err)
				return
			}
			return
		}

		// Get the currently used provider from configuration
		providerName := viper.GetString("provider")

		// Check if we need to interactively set temperature
		if setTemperatureInteractive {
			// Get the active provider
			provider := api.GetActiveProvider()
			if provider == nil {
				fmt.Println("No active provider found. Please configure a provider first.")
				return
			}

			currentModel := provider.GetCurrentModel()
			currentTemperature := provider.GetCurrentTemperature()

			// Check if the current model supports temperature settings
			if provider.GetName() == "openai" && (currentModel == "o1" || currentModel == "o3-mini") {
				fmt.Printf("Note: The current model '%s' does not support temperature settings. Temperature will be ignored.\n\n", currentModel)
			}

			// Get provider-specific temperature presets
			providerPresets := provider.GetTemperaturePresets()

			fmt.Printf("Current temperature for %s: %.1f\n\n", provider.GetName(), currentTemperature)
			fmt.Println("Available temperature presets:")

			// Display provider-specific presets if available
			var presets []api.TemperaturePreset
			if len(providerPresets) > 0 {
				presets = providerPresets
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
				presets = api.TemperaturePresets
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

			// Create an input reader
			reader := bufio.NewReader(os.Stdin)

			// Prompt the user to select a temperature preset
			fmt.Print("\nEnter preset number or 'C' for custom (or press Enter to cancel): ")
			tempInput, err := reader.ReadString('\n')
			if err != nil {
				fmt.Printf("Error reading input: %v\n", err)
				return
			}

			// Process the input
			tempInput = strings.TrimSpace(tempInput)
			if tempInput == "" {
				fmt.Println("Temperature change canceled.")
				return
			}

			// Handle custom temperature
			var newTemperature float64
			if tempInput == "C" || tempInput == "c" {
				// Prompt for custom temperature
				fmt.Printf("Enter custom temperature (0.0-%.1f): ", maxTemp)
				customTemp, err := reader.ReadString('\n')
				if err != nil {
					fmt.Printf("Error reading input: %v\n", err)
					return
				}
				customTemp = strings.TrimSpace(customTemp)
				tempValue, err := strconv.ParseFloat(customTemp, 64)
				if err != nil || tempValue < 0 || tempValue > maxTemp {
					fmt.Printf("Invalid temperature value. Please enter a number between 0.0 and %.1f.\n", maxTemp)
					return
				}
				newTemperature = tempValue
			} else {
				// Handle preset temperature
				tempNum, err := strconv.Atoi(tempInput)
				if err != nil || tempNum < 1 || tempNum > len(presets) {
					fmt.Println("Invalid preset number. Please try again.")
					return
				}
				newTemperature = presets[tempNum-1].Value
			}

			// Check if temperature is already set to the selected value
			if newTemperature == currentTemperature {
				fmt.Printf("Temperature is already set to %.1f\n", newTemperature)
				return
			}

			// Set the provider's temperature
			if err := provider.SetCurrentTemperature(newTemperature); err != nil {
				fmt.Printf("Error setting temperature: %v\n", err)
				return
			}

			// Save provider configuration
			config := make(map[string]interface{})
			provider.SaveConfig(config)

			// Save to viper
			for k, v := range config {
				viper.Set(fmt.Sprintf("providers.%s.%s", provider.GetName(), k), v)
			}

			// Write to the configuration file
			if err := viper.WriteConfig(); err != nil {
				fmt.Printf("Error saving temperature setting: %v\n", err)
			}

			fmt.Printf("Temperature for %s set to %.1f and saved to config.\n", provider.GetName(), newTemperature)
			return
		}

		// Check if we need to interactively select a model
		if selectModelInteractive {
			// Get the active provider
			provider := api.GetActiveProvider()
			if provider == nil {
				fmt.Println("No active provider found. Please configure a provider first.")
				return
			}

			// Get the current model
			currentModel := provider.GetCurrentModel()

			fmt.Printf("Current model: %s\n\n", currentModel)
			fmt.Println("Available models for provider: " + provider.GetName())

			// Get the available models for the current provider
			availableModels := provider.GetAvailableModels()
			if len(availableModels) == 0 {
				fmt.Println("No available models found for this provider.")
				return
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

			// Create an input reader
			reader := bufio.NewReader(os.Stdin)

			// Prompt the user to select a model
			fmt.Print("\nEnter model number to switch (or press Enter to cancel): ")
			modelInput, err := reader.ReadString('\n')
			if err != nil {
				fmt.Printf("Error reading input: %v\n", err)
				return
			}

			// Process the input
			modelInput = strings.TrimSpace(modelInput)
			if modelInput == "" {
				fmt.Println("Model switch canceled.")
				return
			}

			// Convert user input to integer
			modelNum, err := strconv.Atoi(modelInput)
			if err != nil {
				fmt.Println("Invalid model number. Please try again.")
				return
			}

			// Set the new model
			if modelNum < 1 || modelNum > len(availableModels) {
				fmt.Println("Invalid model number. Please try again.")
				return
			}

			newModel := availableModels[modelNum-1]
			if newModel == currentModel {
				fmt.Printf("Already using model: %s\n", newModel)
				return
			}

			// Set the new model
			if err := provider.SetCurrentModel(newModel); err != nil {
				fmt.Printf("Error setting model: %v\n", err)
				return
			}

			// Save provider configuration
			config := make(map[string]interface{})
			provider.SaveConfig(config)

			// Save to viper
			for k, v := range config {
				viper.Set(fmt.Sprintf("providers.%s.%s", provider.GetName(), k), v)
			}

			// Write to the configuration file
			if err := viper.WriteConfig(); err != nil {
				fmt.Printf("Error saving model setting: %v\n", err)
			}

			fmt.Printf("Switched to model: %s\n", newModel)
			return
		}
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

		// Check if there's piped input
		stat, _ := os.Stdin.Stat()
		hasPipedInput := (stat.Mode() & os.ModeCharDevice) == 0

		// We'll handle the -i flag without argument case in a simpler way

		// If there's piped input, read it
		if hasPipedInput {
			DebugLog("Detected piped input")
			reader := bufio.NewReader(os.Stdin)
			pipedInput, err := io.ReadAll(reader)
			if err != nil {
				fmt.Printf("Error reading piped input: %v\n", err)
				return
			}

			// Use the piped input as the input message
			inputMessage = strings.TrimSpace(string(pipedInput))
		}

		// No special case handling here - we'll handle it in a cleaner way

		// Get input from arguments if provided
		if len(args) > 0 {
			// 如果已经有管道输入，则将命令行参数添加到管道输入后面，而不是覆盖它
			if inputMessage != "" {
				inputMessage = inputMessage + "\n\n" + strings.Join(args, " ")
			} else {
				inputMessage = strings.Join(args, " ")
			}
		}

		// If we have any input (from arguments or piped input)
		if inputMessage != "" {
			// Create a single message
			messages := []api.ChatMessage{
				{Role: "user", Content: inputMessage},
			}

			if interactiveMode {
				StartInteractiveMode(inputMessage)
				return // Return after starting interactive mode to prevent double initialization
			} else {
				DebugLog("Sending chat request to provider %s with message: %s", provider.GetName(), inputMessage)

				// Use streaming API for better user experience
				streamChan, err := api.SendStreamingChatRequest(messages)
				if err != nil {
					fmt.Printf("\nError: %v\n\n", err.Error())
					return
				}

				// Process streaming response
				var fullResponse strings.Builder
				for streamResp := range streamChan {
					if streamResp.Error != nil {
						fmt.Printf("\nError: %v\n\n", streamResp.Error)
						return
					}
					fmt.Print(streamResp.Content)
					fullResponse.WriteString(streamResp.Content)
				}
				// 确保在响应后有足够的换行
				fmt.Println()
			}
		}

		// No input messages, check if we should enter interactive mode
		if interactiveMode {
			// Start interactive mode without printing welcome again
			StartInteractiveMode("")
		}
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	// No special handling needed for boolean flags

	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

// Whether to display the version information
var showVersion bool

// Whether to interactively select a provider
var selectProvider bool

// Whether to run in interactive mode
var interactiveMode bool

// Input message to send to the AI
var inputMessage string

// Whether to interactively select a model
var selectModelInteractive bool

// Whether to interactively set temperature
var setTemperatureInteractive bool

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
		fmt.Printf("Provider %s configured successfully!\n", providerName)
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

	// Set the active provider based on the config file
	configuredProvider := viper.GetString("provider")
	if configuredProvider != "" {
		DebugLog("Setting active provider from config: %s", configuredProvider)
		if err := api.SetActiveProvider(configuredProvider); err != nil {
			fmt.Printf("Warning: Error setting active provider to %s: %v\n", configuredProvider, err)
		} else {
			DebugLog("Successfully set active provider to: %s", configuredProvider)
		}
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// No wrapper needed with our new approach

	// Add version flag
	rootCmd.Flags().BoolVarP(&showVersion, "version", "v", false, "Display the current version of chait")
	// Add provider selection flag
	rootCmd.Flags().BoolVarP(&selectProvider, "provider", "p", false, "Interactively select a provider")
	// Add interactive mode flag to enter interactive mode
	rootCmd.Flags().BoolVarP(&interactiveMode, "interactive", "i", false, "Enter interactive mode after sending message")
	// Add model selection flag
	rootCmd.Flags().BoolVarP(&selectModelInteractive, "model", "m", false, "Interactively select a model for the current provider")
	// Add temperature setting flag
	rootCmd.Flags().BoolVarP(&setTemperatureInteractive, "temperature", "t", false, "Interactively set temperature for the current provider")

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.config/chait/config.json)")
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
		// 仅在交互模式下打印配置目录信息
		if len(os.Args) > 1 && (os.Args[1] == "-i" || os.Args[1] == "--interactive") {
			fmt.Printf("Config directory: %s\n", configDir)
		}

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
				"version":   1,
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
