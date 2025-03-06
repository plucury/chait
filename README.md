# chait

🤖 **Chat with AI directly from your command line!**

Chait is a Golang-based command-line tool that allows you to have natural, fluid conversations with various AI models without leaving your terminal. Whether you're a developer, system administrator, or command-line enthusiast, chait provides a convenient AI interaction experience right where you work.

## Key Features

### 💬 Command-Line AI Chat
- **Seamless Terminal Experience**: Chat with AI directly in your familiar command-line environment without switching applications
- **Interactive Chat Mode**: Support for multi-turn conversations with context continuity
- **Instant Responses**: Quickly get AI answers to boost your productivity

### 🔄 Multi-Model Support
- **Multiple Providers**: Currently supports major AI providers including OpenAI, Deepseek, and more
- **Flexible Model Switching**: Easily switch between different AI models
- **Customizable Parameters**: Adjust temperature and other parameters to control response creativity

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
