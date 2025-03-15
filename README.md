# chait

🤖 **Chat with AI directly from your command line and more!**

## Quick Start

```bash
# Ask a quick question (non-interactive mode is default)
chait "What is the capital of France?"

# Process command output with AI
ls -la | chait "Explain what these files are"

# Get AI explanation of a code file
cat main.go | chait "Explain this code"

# Start interactive chat mode for multi-turn conversations
chait -i

# Start interactive mode with an initial question
# This will automatically send the question to the AI
chait -i "Tell me about quantum computing"

# Analyze a file and have a conversation about it
cat config.json | chait -i
```

## Overview

Chait is a Golang-based command-line tool that allows you to have natural, fluid conversations with various AI models without leaving your terminal. Whether you're a developer, system administrator, or command-line enthusiast, chait provides a convenient AI interaction experience right where you work.

## Key Features

### 💬 Command-Line AI Chat
- **Seamless Terminal Experience**: Chat with AI directly in your familiar command-line environment without switching applications
- **Quick Query Mode**: By default, get quick answers without entering interactive mode
- **Interactive Chat Mode**: Support for multi-turn conversations with context continuity using -i flag
- **Streaming Responses**: See AI responses in real-time as they're generated
- **Instant Responses**: Quickly get AI answers to boost your productivity

### 🔄 Multi-Model Support
- **Multiple Providers**: Currently supports major AI providers including OpenAI, Deepseek, Grok, and more
- **Flexible Model Switching**: Easily switch between different AI models
- **Customizable Parameters**: Adjust temperature and other parameters to control response creativity

## Installation

### Option 1: Using Go Install

If you have Go installed on your system, you can install chait directly using the Go toolchain:

```bash
go install github.com/plucury/chait@latest
```

### Option 2: Download from GitHub Releases

You can also download pre-compiled binaries from the [GitHub Releases page](https://github.com/plucury/chait/releases):

```bash
# For macOS (Apple Silicon)
curl -L https://github.com/plucury/chait/releases/latest/download/chait-darwin-arm64 -o chait
chmod +x chait
sudo mv chait /usr/local/bin/

# For macOS (Intel)
curl -L https://github.com/plucury/chait/releases/latest/download/chait-darwin-amd64 -o chait
chmod +x chait
sudo mv chait /usr/local/bin/

# For Linux (AMD64)
curl -L https://github.com/plucury/chait/releases/latest/download/chait-linux-amd64 -o chait
chmod +x chait
sudo mv chait /usr/local/bin/

# For Linux (ARM64)
curl -L https://github.com/plucury/chait/releases/latest/download/chait-linux-arm64 -o chait
chmod +x chait
sudo mv chait /usr/local/bin/
```

Alternatively, you can manually download the appropriate binary for your system from the [Releases page](https://github.com/plucury/chait/releases), make it executable, and move it to a directory in your PATH.

## Supported Providers

### OpenAI
- Models: gpt-4o, gpt-4o-mini, gpt-4.5, o1, o3-mini
- Temperature range: 0.0-1.0

### Deepseek
- Models: deepseek-chat, deepseek-reasoner
- Temperature range: 0.0-2.0

### Grok
- Models: grok-2-1212
- Temperature range: 0.0-2.0 (Higher values like 0.8 make output more random, lower values like 0.2 make it more focused)

## Usage Guide

### Command Structure

```bash
chait [options] ["your question here"]
```

### Common Options

```bash
-i, --interactive    # Enter interactive mode for multi-turn conversations
-p, --provider       # Interactively select a provider
-m, --model          # Interactively select a model for the current provider
-t, --temperature    # Interactively set temperature for the current provider
-v, --version        # Display the current version
--help               # Show help information
```

### Usage Modes

#### 1. Quick Query Mode (Default)

By default, chait runs in non-interactive mode for quick answers:

```bash
# Single question
chait "What is the capital of France?"

# Multiple arguments combined as one question
chait "Tell me about" "the history of" "AI"
```

#### 2. Interactive Mode

Use `-i` flag to enter interactive mode for multi-turn conversations:

```bash
# Start interactive mode with a question
chait -i "Tell me about quantum computing"

# Start interactive mode without an initial question
chait -i
```

#### 3. Model Selection

Interactively select a model for the current provider:

```bash
# Select a model interactively
chait -m

```

#### 4. Temperature Setting

Interactively set the temperature for the current provider:

```bash
# Set temperature interactively
chait -t

```

#### 5. Piped Input

Process command outputs or file contents:

```bash
# Process command output with AI (non-interactive mode)
ls -la | chait "Explain these files"

# Analyze code
cat main.go | chait "What does this code do?"

# Analyze logs
grep ERROR app.log | chait "Explain these errors"

# Process input and enter interactive mode for follow-up questions
ls -la | chait -i "Explain these files"
cat config.json | chait -i
git diff | chait -i
```

### Interactive Mode Commands

When in interactive mode, you can use these special commands:

```
:h              # Show help information
:c       # Start a new conversation
:m              # Switch between available models
:t              # Set the temperature parameter
:p              # Configure or switch provider
:k              # Set the API key for the current provider
ctrl+c          # Exit interactive mode
```

### Interactive Mode Features

#### API Key Management
- **Direct API Key Input**: If the current provider is not ready (missing API key), you'll be prompted to enter your API key directly in the interactive mode
- **Manual API Key Setting**: Use the `:k` command to set or update the API key for the current provider at any time
- **Persistent Configuration**: API keys are securely saved to your configuration file for future sessions

#### User Interface
- **Full-Screen Terminal UI**: Utilizes the entire terminal window for a distraction-free experience
- **Message History**: View your entire conversation history with clear visual distinction between user and AI messages
- **Real-Time Streaming**: See AI responses as they're generated in real-time
- **Text Selection**: Select and copy text from the conversation using mouse or keyboard
- **Scrolling**: Navigate through long conversations with keyboard shortcuts
- **Visual Feedback**: Different message types (System, User, Assistant, Error) are visually distinguished

#### Keyboard Navigation
- **PageUp/PageDown**: Scroll through conversation history one page at a time
- **Home/End**: Jump to the beginning or end of the current input
- **Ctrl+Home/Ctrl+End**: Jump to the top or bottom of the conversation history
- **Enter**: Send your message or confirm selection
- **Esc**: Cancel current selection or operation

#### Error Handling
- **Clear Error Messages**: Errors are displayed with distinct formatting to help troubleshoot issues
- **API Connection Errors**: Automatically detects and reports issues with API connections
- **Provider Configuration**: Guides you through fixing configuration issues when they occur
