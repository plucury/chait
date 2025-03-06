package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// configCmd represents the config command
var configCmd = &cobra.Command{
	Use:   "config [key] [value]",
	Short: "Set configuration values",
	Long: `Set configuration values in ~/.config/chain/config.json.
Example:
  chait config providers.deepseek.api_key YOUR_API_KEY`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 2 {
			fmt.Println("Error: config requires a key and value")
			return
		}
		
		key := args[0]
		value := args[1]
		setConfig(key, value)
	},
}

func setConfig(key, value string) {
	// Try to convert string values to appropriate types
	switch strings.ToLower(value) {
	case "true":
		viper.Set(key, true)
	case "false":
		viper.Set(key, false)
	default:
		// Try to parse as number if possible
		if num, err := parseNumber(value); err == nil {
			viper.Set(key, num)
		} else {
			viper.Set(key, value)
		}
	}

	if err := viper.WriteConfig(); err != nil {
		fmt.Printf("Error writing config: %v\n", err)
		return
	}
	fmt.Printf("Set '%s' to '%v'\n", key, viper.Get(key))
}

// parseNumber tries to parse a string as an int or float
func parseNumber(s string) (interface{}, error) {
	// Try to parse as int
	if i, err := fmt.Sscanf(s, "%d", new(int)); err == nil && i == 1 {
		var result int
		fmt.Sscanf(s, "%d", &result)
		return result, nil
	}
	
	// Try to parse as float
	if i, err := fmt.Sscanf(s, "%f", new(float64)); err == nil && i == 1 {
		var result float64
		fmt.Sscanf(s, "%f", &result)
		return result, nil
	}
	
	return nil, fmt.Errorf("not a number")
}

func init() {
	rootCmd.AddCommand(configCmd)
}
