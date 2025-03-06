# chait

A Golang command-line tool based on Cobra for managing configuration data and interacting with AI providers.

## Features

- Configuration data is stored in `~/.config/chait/config.json`
- Supports getting, setting, listing, and resetting configurations
- Uses Viper for configuration management, supporting nested configuration items
- Supports multiple AI providers (OpenAI, Deepseek)
- Interactive chat mode with model and temperature settings
- Provider-specific temperature ranges (OpenAI: 0-1, Deepseek: 0-2)
- Automatic conversation history clearing when switching providers or models
- Debug mode for troubleshooting and development with timestamped logs

## Installation

```bash
go install github.com/plucury/chait@latest
```

## Usage

### Basic Commands

```bash
# Show help information
chait --help

# Display the current version
chait --version or chait -v

# Interactively select a provider
chait --provider or chait -p

# Start interactive chat mode
chait
```

### Interactive Mode Commands

```bash
# Show help information
:help or :h

# Start a new conversation
:clear or :c

# Switch between available models for the current provider
:model

# Set the temperature parameter
:temperature or :temp

# Configure or switch provider
:provider

# Exit interactive mode
:quit or :q
```

## Development

### Building

```bash
go build -o chait
```

### Running Commands During Development

```bash
go run main.go [command] [args]
```

### Adding New Providers

Implement the Provider interface defined in `api/provider/provider.go` and register it in the init function.
