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
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "chait",
	Short: "A command-line tool",
	Long: `A command-line tool built with Cobra.
This tool allows you to manage configurations stored in ~/.config/chait/config.json.`,
	Run: func(cmd *cobra.Command, args []string) {
		// 检查是否显示可用的 provider 列表
		if showProviders {
			// 获取所有可用的 provider 名称
			providerNames := api.GetAvailableProviders()

			// 加载所有 provider 的配置
			for _, name := range providerNames {
				providerConfig := viper.GetStringMap(fmt.Sprintf("providers.%s", name))

				// 打印调试信息
				fmt.Printf("Loading config for provider %s\n", name)
				fmt.Printf("Config from viper: %v\n", providerConfig)

				config := make(map[string]interface{})
				for k, v := range providerConfig {
					config[k] = v
				}

				// 检查错误
				if err := api.LoadProviderConfig(name, config); err != nil {
					fmt.Printf("Error loading config for provider %s: %v\n", name, err)
				}
			}

			// 获取所有就绪的 provider
			readyProviders := api.GetReadyProviders()

			// 显示所有 provider 信息
			fmt.Println("Available providers:")
			for i, name := range providerNames {
				p, _ := api.GetProvider(name)
				readyStatus := "not ready"
				for _, rp := range readyProviders {
					if rp.GetName() == name {
						readyStatus = "ready"
						break
					}
				}
				fmt.Printf("  %d. %s (%s)\n", i+1, name, readyStatus)
				fmt.Printf("     Default model: %s\n", p.GetDefaultModel())
				fmt.Printf("     Available models: %s\n", strings.Join(p.GetAvailableModels(), ", "))
			}
			return
		}

		// 从配置中获取当前使用的 provider
		providerName := viper.GetString("provider")

		// 如果 provider 配置为空，则提示用户选择
		if providerName == "" {
			fmt.Println("No provider selected. Let's choose one.")
			// 提示用户选择并配置 provider
			if err := configureProvider(); err != nil {
				fmt.Printf("Error configuring provider: %v\n", err)
				return
			}

			// 从配置中重新获取当前使用的 provider
			providerName = viper.GetString("provider")
			if providerName == "" {
				// 如果仍然为空，使用默认值
				providerName = api.DefaultProvider
			}
		}

		// 加载 provider 配置
		providerConfig := viper.GetStringMap(fmt.Sprintf("providers.%s", providerName))

		// 将 viper 配置转换为 map[string]interface{}
		config := make(map[string]interface{})
		for k, v := range providerConfig {
			config[k] = v
		}

		// 加载 provider 配置
		if err := api.LoadProviderConfig(providerName, config); err != nil {
			fmt.Printf("Error loading provider config: %v\n", err)
			return
		}

		// 获取所有就绪的 provider
		readyProviders := api.GetReadyProviders()

		// 检查是否有可用的 provider
		if len(readyProviders) == 0 {
			fmt.Println("No ready providers found. Let's configure one.")
			// 提示用户选择并配置 provider
			if err := configureProvider(); err != nil {
				fmt.Printf("Error configuring provider: %v\n", err)
				return
			}

			// 重新获取就绪的 provider
			readyProviders = api.GetReadyProviders()
			if len(readyProviders) == 0 {
				fmt.Println("Still no ready providers. Exiting.")

				// 调试信息
				fmt.Println("DEBUG: Checking provider status...")
				providerNames := api.GetAvailableProviders()
				for _, name := range providerNames {
					p, exists := api.GetProvider(name)
					if exists {
						fmt.Printf("DEBUG: Provider %s exists, IsReady: %v, API Key set: %v\n",
							name, p.IsReady(), p.GetAPIKey() != "")
					}
				}
				return
			}
		}

		// 获取活跃的 provider
		provider := api.GetActiveProvider()

		// 检查当前活跃的 provider 是否就绪
		if !provider.IsReady() {
			// 如果当前活跃的 provider 不就绪，但有其他就绪的 provider，则切换到第一个就绪的 provider
			if err := api.SetActiveProvider(readyProviders[0].GetName()); err != nil {
				fmt.Printf("Error setting active provider: %v\n", err)
				return
			}
			provider = readyProviders[0]
			fmt.Printf("Switched to ready provider: %s\n", provider.GetName())
		}

		// API key 已设置，进入交互环境
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

// 是否显示可用的 provider 列表
var showProviders bool

// configureProvider 函数提示用户选择并配置 provider
func configureProvider() error {
	// 创建输入读取器
	reader := bufio.NewReader(os.Stdin)

	// 获取所有可用的 provider 名称
	providerNames := api.GetAvailableProviders()
	if len(providerNames) == 0 {
		return fmt.Errorf("no available providers found")
	}

	// 显示可用的 provider 列表
	fmt.Println("Available providers:")
	for i, name := range providerNames {
		p, _ := api.GetProvider(name)
		fmt.Printf("  %d. %s\n", i+1, name)
		fmt.Printf("     Default model: %s\n", p.GetDefaultModel())
		fmt.Printf("     Available models: %s\n", strings.Join(p.GetAvailableModels(), ", "))
	}

	// 提示用户选择 provider
	fmt.Print("\nSelect a provider (enter number): ")
	choiceStr, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("error reading input: %v", err)
	}

	// 处理输入
	choiceStr = strings.TrimSpace(choiceStr)
	choice, err := strconv.Atoi(choiceStr)
	if err != nil || choice < 1 || choice > len(providerNames) {
		return fmt.Errorf("invalid choice: %v", err)
	}

	// 获取选择的 provider 名称
	selectedProvider := providerNames[choice-1]

	// 设置为活跃 provider
	if err := api.SetActiveProvider(selectedProvider); err != nil {
		return fmt.Errorf("error setting active provider: %v", err)
	}

	// 将选择的 provider 保存到配置文件
	viper.Set("provider", selectedProvider)

	// 写入配置文件
	if err := viper.WriteConfig(); err != nil {
		fmt.Printf("Error saving provider setting: %v\n", err)
	}

	// 加载 provider 配置
	providerConfig := viper.GetStringMap(fmt.Sprintf("providers.%s", selectedProvider))
	// print config detail for debugging
	config := make(map[string]interface{})
	for k, v := range providerConfig {
		config[k] = v
	}

	// 加载 provider 配置
	if err := api.LoadProviderConfig(selectedProvider, config); err != nil {
		return fmt.Errorf("error loading provider config: %v", err)
	}

	// 获取 provider 实例
	provider, exists := api.GetProvider(selectedProvider)
	if !exists {
		return fmt.Errorf("provider %s not found", selectedProvider)
	}

	// 检查 API key 是否已设置
	if provider.GetAPIKey() == "" {
		// 提示用户输入 API key
		fmt.Printf("Enter API key for %s: ", selectedProvider)
		apiKeyStr, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("error reading API key: %v", err)
		}

		// 处理输入
		apiKey := strings.TrimSpace(apiKeyStr)

		// 设置 API key
		provider.SetAPIKey(apiKey)

		// 保存 provider 配置
		config := make(map[string]interface{})
		provider.SaveConfig(config)

		// 保存到 viper
		for k, v := range config {
			viper.Set(fmt.Sprintf("providers.%s.%s", selectedProvider, k), v)
		}

		// 写入配置文件
		if err := viper.WriteConfig(); err != nil {
			return fmt.Errorf("error saving API key: %v", err)
		}

		// 重新加载配置以确保 API key 生效
		if err := api.LoadProviderConfig(selectedProvider, config); err != nil {
			return fmt.Errorf("error reloading provider config: %v", err)
		}

		fmt.Printf("%s API key set successfully!\n", selectedProvider)
	}

	// 选择模型
	availableModels := provider.GetAvailableModels()
	if len(availableModels) > 1 {
		// 显示可用的模型列表
		fmt.Println("\nAvailable models:")
		for i, model := range availableModels {
			fmt.Printf("  %d. %s\n", i+1, model)
		}

		// 提示用户选择模型
		fmt.Print("\nSelect a model (enter number or press Enter for default): ")
		modelInputStr, _ := reader.ReadString('\n')

		// 处理输入
		modelInput := strings.TrimSpace(modelInputStr)

		// 如果用户输入了模型选择
		if modelInput != "" {
			modelChoice, err := strconv.Atoi(modelInput)
			if err != nil || modelChoice < 1 || modelChoice > len(availableModels) {
				fmt.Println("Invalid choice, using default model.")
			} else {
				// 设置选择的模型
				selectedModel := availableModels[modelChoice-1]
				if err := provider.SetCurrentModel(selectedModel); err != nil {
					fmt.Printf("Error setting model: %v, using default model.\n", err)
				} else {
					// 保存 provider 配置
					config := make(map[string]interface{})
					provider.SaveConfig(config)

					// 保存到 viper
					for k, v := range config {
						viper.Set(fmt.Sprintf("providers.%s.%s", selectedProvider, k), v)
					}

					// 写入配置文件
					if err := viper.WriteConfig(); err != nil {
						fmt.Printf("Error saving model setting: %v\n", err)
					}

					// 重新加载配置以确保模型设置生效
					if err := api.LoadProviderConfig(selectedProvider, config); err != nil {
						fmt.Printf("Error reloading provider config: %v\n", err)
					}

					fmt.Printf("Model set to: %s\n", selectedModel)
				}
			}
		}
	}

	// 选择温度
	temperaturePresets := provider.GetTemperaturePresets()
	if len(temperaturePresets) > 0 {
		// 显示可用的温度预设列表
		fmt.Println("\nAvailable temperature presets:")
		for i, preset := range temperaturePresets {
			fmt.Printf("  %d. %s (%.1f) - %s\n", i+1, preset.Name, preset.Value, preset.Description)
		}

		// 提示用户选择温度预设
		fmt.Print("\nSelect a temperature preset (enter number or press Enter for default): ")
		tempInputStr, _ := reader.ReadString('\n')

		// 处理输入
		tempInput := strings.TrimSpace(tempInputStr)

		// 如果用户输入了温度预设选择
		if tempInput != "" {
			tempChoice, err := strconv.Atoi(tempInput)
			if err != nil || tempChoice < 1 || tempChoice > len(temperaturePresets) {
				fmt.Println("Invalid choice, using default temperature.")
			} else {
				// 设置选择的温度预设
				selectedTemp := temperaturePresets[tempChoice-1].Value
				if err := provider.SetCurrentTemperature(selectedTemp); err != nil {
					fmt.Printf("Error setting temperature: %v, using default temperature.\n", err)
				} else {
					// 保存 provider 配置
					config := make(map[string]interface{})
					provider.SaveConfig(config)

					// 保存到 viper
					for k, v := range config {
						viper.Set(fmt.Sprintf("providers.%s.%s", selectedProvider, k), v)
					}

					// 写入配置文件
					if err := viper.WriteConfig(); err != nil {
						fmt.Printf("Error saving temperature setting: %v\n", err)
					}

					// 重新加载配置以确保温度设置生效
					if err := api.LoadProviderConfig(selectedProvider, config); err != nil {
						fmt.Printf("Error reloading provider config: %v\n", err)
					}

					fmt.Printf("Temperature set to: %.1f\n", selectedTemp)
				}
			}
		}
	}

	// 最终检查提供商是否就绪
	if !provider.IsReady() {
		fmt.Printf("WARNING: Provider %s is still not ready after configuration.\n", selectedProvider)
		fmt.Println("Please check your API key and try again.")
	} else {
		fmt.Printf("Provider %s is now ready to use.\n", selectedProvider)
	}

	return nil
}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.config/chait/config.json)")
	rootCmd.PersistentFlags().BoolVar(&showProviders, "providers", false, "show available providers")
}

// initConfig reads in config file and ENV variables if set.
// Example config.json:
//
//	{
//	  "version": "1.0.0",
//	  "providers": {
//	    "deepseek": {
//	      "api_key": "your_api_key"
//	    }
//	  }
//	}
//
// 交互模式函数
func startInteractiveMode() {
	// 获取活跃的 provider
	provider := api.GetActiveProvider()

	// 获取当前模型和温度
	currentModel := provider.GetCurrentModel()
	currentTemperature := provider.GetCurrentTemperature()
	fmt.Println("Welcome to chait interactive mode!")
	fmt.Println("Type ':quit' or ':q' to exit.")
	fmt.Println("Type ':help' or ':h' for available commands.")
	fmt.Println("Type ':clear' or ':c' to start a new conversation.")
	fmt.Println("Type ':model' to switch between available models.")
	fmt.Println("Type ':temperature' or ':temp' to set the temperature.")
	fmt.Println("-----------------------------------")

	// 使用 readline 库来处理终端输入，提供更好的行编辑功能
	rl, err := readline.New("> ")
	if err != nil {
		fmt.Printf("Error initializing readline: %v\n", err)
		return
	}
	defer rl.Close()

	// 保存对话历史
	var messages []api.ChatMessage

	// 添加系统消息
	messages = append(messages, api.ChatMessage{
		Role:    "system",
		Content: "You are a helpful assistant.",
	})

	for {
		// 使用 readline 读取用户输入，提供更好的行编辑功能
		input, err := rl.Readline()
		if err != nil { // io.EOF, readline.ErrInterrupt
			if err.Error() == "Interrupt" {
				fmt.Println("\nUse :quit or :q to exit")
				continue
			}
			break
		}

		// 检查是否是命令（以冒号开头）
		if len(input) > 0 && input[0] == ':' {
			// 去除冒号
			cmd := input[1:]

			// 处理退出命令
			if cmd == "quit" || cmd == "q" {
				fmt.Println("Goodbye!")
				break
			}

			// 处理帮助命令
			if cmd == "help" || cmd == "h" {
				fmt.Println("Available commands:")
				fmt.Println("  :help, :h        - Show this help message")
				fmt.Println("  :clear, :c       - Start a new conversation")
				fmt.Println("  :model           - Switch between available models")
				fmt.Println("  :temperature, :temp - Set the temperature parameter")
				fmt.Println("  :providers       - List all ready providers")
				fmt.Println("  :quit, :q        - Exit the interactive mode")
				continue
			}

			// 处理清除对话历史命令
			if cmd == "clear" || cmd == "c" {
				messages = messages[:1] // 只保留系统消息
				fmt.Println("Conversation history cleared.")
				continue
			}

			// 处理 providers 命令
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

			// 处理温度设置命令
			if cmd == "temperature" || cmd == "temp" {
				fmt.Printf("Current temperature: %.1f\n\n", currentTemperature)
				fmt.Println("Available temperature presets:")
				for i, preset := range api.TemperaturePresets {
					fmt.Printf("  %d. %s (%.1f) - %s%s\n", i+1, preset.Name, preset.Value, preset.Description, func() string {
						if preset.Value == currentTemperature {
							return " (current)"
						}
						return ""
					}())
				}
				fmt.Println("  C. Custom - Enter a custom temperature value (0.0-2.0)")

				// 使用 readline 实例读取温度选择输入
				rl.SetPrompt("\nEnter preset number or 'C' for custom (or press Enter to cancel): ")
				tempInput, err := rl.Readline()
				rl.SetPrompt("> ") // 恢复原始提示符
				if err != nil {
					fmt.Printf("Error reading input: %v\n", err)
					continue
				}
				if tempInput == "" {
					fmt.Println("Temperature change canceled.")
					continue
				}

				// 处理自定义温度
				if tempInput == "C" || tempInput == "c" {
					// 使用 readline 实例读取自定义温度输入
					rl.SetPrompt("Enter custom temperature (0.0-2.0): ")
					customTemp, err := rl.Readline()
					rl.SetPrompt("> ") // 恢复原始提示符
					if err != nil {
						fmt.Printf("Error reading input: %v\n", err)
						continue
					}
					tempValue, err := strconv.ParseFloat(customTemp, 64)
					if err != nil || tempValue < 0 || tempValue > 2 {
						fmt.Println("Invalid temperature value. Please enter a number between 0.0 and 2.0.")
						continue
					}

					// 设置新的温度
					currentTemperature = tempValue
				} else {
					// 处理预设温度
					tempNum, err := strconv.Atoi(tempInput)
					if err != nil || tempNum < 1 || tempNum > len(api.TemperaturePresets) {
						fmt.Println("Invalid preset number. Please try again.")
						continue
					}

					// 设置新的温度
					currentTemperature = api.TemperaturePresets[tempNum-1].Value
				}

				// 设置 provider 的温度
				provider := api.GetActiveProvider()
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
					fmt.Printf("Temperature set to %.1f and saved to config.\n", currentTemperature)
				}
				continue
			}

			// 处理模型切换命令
			if cmd == "model" {
				// 获取当前 provider
				provider := api.GetActiveProvider()
				// 使用已经在外部声明的 currentModel 变量
				currentModel = provider.GetCurrentModel()
				
				fmt.Printf("Current model: %s\n\n", currentModel)
				fmt.Println("Available models for provider: " + provider.GetName())
				
				// 获取当前 provider 的可用模型
				availableModels := provider.GetAvailableModels()
				if len(availableModels) == 0 {
					fmt.Println("No available models found for this provider.")
					continue
				}
				
				// 显示可用模型
				for i, model := range availableModels {
					fmt.Printf("  %d. %s%s\n", i+1, model, func() string {
						if model == currentModel {
							return " (current)"
						}
						return ""
					}())
				}

				// 使用 readline 实例读取模型选择输入
				rl.SetPrompt("\nEnter model number to switch (or press Enter to cancel): ")
				modelInput, err := rl.Readline()
				rl.SetPrompt("> ") // 恢复原始提示符
				if err != nil {
					fmt.Printf("Error reading input: %v\n", err)
					continue
				}
				if modelInput == "" {
					fmt.Println("Model switch canceled.")
					continue
				}

				// 转换用户输入为整数
				modelNum, err := strconv.Atoi(modelInput)
				if err != nil {
					fmt.Println("Invalid model number. Please try again.")
					continue
				}

				// 保存旧模型
				oldModel := currentModel

				// 重用之前获取的可用模型列表
				
				// 设置新模型
				if modelNum < 1 || modelNum > len(availableModels) {
					fmt.Println("Invalid model number. Please try again.")
					continue
				}
				
				newModel := availableModels[modelNum-1]
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
				}

				// 如果模型已经改变，清除对话历史
				if oldModel != currentModel {
					messages = messages[:1] // 只保留系统消息
					fmt.Printf("Switched to model: %s. Conversation history cleared.\n", currentModel)
				} else {
					fmt.Printf("Already using model: %s\n", currentModel)
				}
				continue
			}

			// 如果是未知命令
			fmt.Printf("Unknown command: %s\n", cmd)
			fmt.Println("Type :help for available commands.")
			continue
		}

		// 处理用户输入的其他命令
		if input != "" {
			// 添加用户消息到历史
			messages = append(messages, api.ChatMessage{
				Role:    "user",
				Content: input,
			})

			// 发送请求到 AI provider
			fmt.Println("Thinking...")
			response, err := api.SendChatRequest("", messages, "", 0)
			if err != nil {
				// 处理特定错误
				errMsg := err.Error()
				fmt.Printf("\nError: %v\n\n", errMsg)

				// 检查是否是余额不足错误
				if strings.Contains(errMsg, "insufficient balance") || strings.Contains(errMsg, "Insufficient Balance") {
					fmt.Println("Your Deepseek API account has insufficient balance.")
					fmt.Println("Please check your account at https://platform.deepseek.com/")
					fmt.Println("You can continue using the CLI with a different API key by running:")
					fmt.Println("  chait config providers.deepseek.api_key YOUR_NEW_API_KEY")
				}

				// 从历史中移除最后一条用户消息，因为请求失败
				messages = messages[:len(messages)-1]
				continue
			}

			// 打印 AI 的回复
			fmt.Println("\n" + response + "\n")

			// 添加 AI 回复到历史
			messages = append(messages, api.ChatMessage{
				Role:    "assistant",
				Content: response,
			})

			// 如果历史太长，可以裁剪
			if len(messages) > 20 {
				// 保留系统消息和最近的对话
				messages = append(messages[:1], messages[len(messages)-19:]...)
			}
		}
	}
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

			// 获取所有可用的 provider
			providers := api.GetAvailableProviders()

			// 创建默认配置
			defaultConfig := map[string]interface{}{
				"version":   "1.0.0",
				"provider":  "", // 当前使用的 provider，空字符串表示需要用户选择
				"providers": map[string]interface{}{},
			}

			// 为每个 provider 创建默认配置
			providersConfig := defaultConfig["providers"].(map[string]interface{})
			for _, providerName := range providers {
				p, _ := api.GetProvider(providerName)
				config := make(map[string]interface{})
				p.SaveConfig(config)
				providersConfig[providerName] = config
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
