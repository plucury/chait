package util

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

// IsDebugMode returns true if debug mode is enabled in the configuration
func IsDebugMode() bool {
	return viper.GetBool("debug")
}

// DebugLog prints a debug message if debug mode is enabled
func DebugLog(format string, args ...interface{}) {
	if IsDebugMode() {
		timestamp := time.Now().Format("2006-01-02 15:04:05")
		fmt.Printf("[DEBUG %s] ", timestamp)
		fmt.Printf(format, args...)
		fmt.Println()
	}
}
