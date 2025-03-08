# chait

🤖 **Chat with AI directly from your command line!**

Chait is a Golang-based command-line tool that allows you to have natural, fluid conversations with various AI models without leaving your terminal. Whether you're a developer, system administrator, or command-line enthusiast, chait provides a convenient AI interaction experience right where you work.

## Key Features

### 💬 Command-Line AI Chat
- **Seamless Terminal Experience**: Chat with AI directly in your familiar command-line environment without switching applications
- **Interactive Chat Mode**: Support for multi-turn conversations with context continuity
- **Non-Interactive Mode**: Quick one-off queries without entering interactive mode using -n flag
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

## Usage

### Basic Commands

```bash
# Show help information
chait --help

# Display the current version
chait --version or chait -v

# Interactively select a provider
chait --provider or chait -p

# Start interactive chat mode (default)
chait
```

### Quick Query Mode
Use `-n` (or `--no-interaction`) flag to get quick answers without entering interactive mode:

```bash
# Single question
chait -n "Your question here"

# Multiple arguments
chait -n "Tell me about" "the history of" "AI"
```

### Piped Input
Chait supports receiving input from pipes, which is useful for processing command outputs or file contents:

```bash
# Process command output with AI
ls -la | chait -n "Explain what these files are"

# Get AI explanation of a code file
cat main.go | chait -n "Explain this code"

# Analyze log files
grep ERROR app.log | chait -n "What might be causing these errors?"
```

#### Piped Input with Interactive Mode
When using piped input without the `-n` flag, chait will process the input and then enter interactive mode, allowing for follow-up questions:

```bash
# Process command output and enter interactive mode for follow-up questions
ls -la | chait

# Analyze a file and have a conversation about it
cat config.json | chait

# Review git changes and discuss them interactively
git diff | chait
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
