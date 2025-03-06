package cmd

import (
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
This tool allows you to manage configurations stored in ~/.config/chain/config.json.`,
	Run: func(cmd *cobra.Command, args []string) {
		// 检查是否已设置 API key
		deepseekAPIKey := viper.GetString("providers.deepseek.api_key")
		if deepseekAPIKey == "" {
			fmt.Println("Deepseek API key is not set. Please set it first using:")
			fmt.Println("  chait config providers.deepseek.api_key YOUR_API_KEY")
			return
		}

		// API key 已设置，进入交互环境
		startInteractiveMode(deepseekAPIKey)
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

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.config/chain/config.json)")
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
func startInteractiveMode(apiKey string) {
	// 当前使用的模型和温度
	currentModel := api.DefaultModel
	
	// 获取配置文件中的温度设置，如果没有则使用默认值
	currentTemperature := viper.GetFloat64("providers.deepseek.temperature")
	if currentTemperature == 0 {
		currentTemperature = api.DefaultTemperature
		// 将默认温度保存到配置文件
		viper.Set("providers.deepseek.temperature", currentTemperature)
		viper.WriteConfig()
	}
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
				fmt.Println("  :quit, :q        - Exit the interactive mode")
				continue
			}

			// 处理清除对话历史命令
			if cmd == "clear" || cmd == "c" {
				messages = messages[:1] // 只保留系统消息
				fmt.Println("Conversation history cleared.")
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
				
				// 保存温度设置到配置文件
				viper.Set("providers.deepseek.temperature", currentTemperature)
				if err := viper.WriteConfig(); err != nil {
					fmt.Printf("Error saving temperature setting: %v\n", err)
				} else {
					fmt.Printf("Temperature set to %.1f and saved to config.\n", currentTemperature)
				}
				continue
			}
			
			// 处理模型切换命令
			if cmd == "model" {
				fmt.Printf("Current model: %s\n\n", currentModel)
				fmt.Println("Available models:")
				for i, model := range api.AvailableModels {
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
				if err != nil || modelNum < 1 || modelNum > len(api.AvailableModels) {
					fmt.Println("Invalid model number. Please try again.")
					continue
				}

				// 切换模型
				oldModel := currentModel
				currentModel = api.AvailableModels[modelNum-1]

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

			// 发送请求到 Deepseek API
			fmt.Println("Thinking...")
			response, err := api.SendChatRequest(apiKey, messages, currentModel, currentTemperature)
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

		// Set up config in ~/.config/chain directory with name "config.json"
		configDir = filepath.Join(home, ".config", "chain")
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

			defaultConfig := map[string]interface{}{
				"version": "1.0.0",
				"providers": map[string]interface{}{
					"deepseek": map[string]interface{}{
						"api_key": "",
					},
				},
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
				fmt.Println("\nNOTICE: Please set your Deepseek API key using:")
				fmt.Println("  chait config providers.deepseek.api_key YOUR_API_KEY")
			}
		} else {
			fmt.Printf("Error reading config file: %v\n", err)
		}
	} else {
		fmt.Printf("Using config file: %s\n", viper.ConfigFileUsed())

		// Check if Deepseek API key is set
		deepseekAPIKey := viper.GetString("providers.deepseek.api_key")
		if deepseekAPIKey == "" {
			fmt.Println("\nWARNING: Deepseek API key is not set.")
			fmt.Print("Would you like to set it now? (y/n): ")

			var response string
			fmt.Scanln(&response)

			if response == "y" || response == "Y" {
				fmt.Print("Enter your Deepseek API key: ")
				var apiKey string
				fmt.Scanln(&apiKey)

				if apiKey != "" {
					// Set the API key in viper
					viper.Set("providers.deepseek.api_key", apiKey)

					// Save to config file
					if err := viper.WriteConfig(); err != nil {
						fmt.Printf("Error saving API key: %v\n", err)
					} else {
						fmt.Println("API key saved successfully!")
					}
				} else {
					fmt.Println("No API key entered. You can set it later using:")
					fmt.Println("  chait config providers.deepseek.api_key YOUR_API_KEY")
				}
			} else {
				fmt.Println("You can set it later using:")
				fmt.Println("  chait config providers.deepseek.api_key YOUR_API_KEY")
			}
		}
	}
}
