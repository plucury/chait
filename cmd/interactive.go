package cmd

import (
	"fmt"
	"strings"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/mattn/go-runewidth"
	"github.com/plucury/chait/api"
	"github.com/plucury/chait/api/provider"
)

// message type enum
type MessageType string

const (
	MessageTypeSystem    MessageType = "System"
	MessageTypeUser      MessageType = "User"
	MessageTypeAssistant MessageType = "Assistant"
	MessageTypeChait     MessageType = "Chait"
	MessageTypeError     MessageType = "Error"
)

type Message struct {
	Type    MessageType
	Content string
}

func (m Message) ToChatMessage() provider.ChatMessage {
	return provider.ChatMessage{
		Role:    strings.ToLower(string(m.Type)),
		Content: m.Content,
	}
}

// WindowSizeMsg is sent when the terminal window is resized
type WindowSizeMsg struct {
	Width  int
	Height int
}

// Point represents a position in the text (line and column)
type point struct {
	line int
	col  int
}

type selectorOption struct {
	name  string
	value interface{}
}

// selectorWidget represents a generic selector UI widget
type selectorWidget struct {
	title        string           // Title to display above the options
	options      []selectorOption // List of available options
	currentIndex int              // Currently selected option index
	isActive     bool             // Whether the selector is currently active/visible
}

func (s *selectorWidget) getCurrentValue() interface{} {
	return s.options[s.currentIndex].value
}

// activate activates the selector widget
func (s *selectorWidget) activate() {
	s.isActive = true
}

// deactivate deactivates the selector widget
func (s *selectorWidget) deactivate() {
	s.isActive = false
}

// selectNext selects the next option in the list
func (s *selectorWidget) selectNext() {
	if len(s.options) == 0 {
		return
	}
	s.currentIndex = (s.currentIndex + 1) % len(s.options)
}

// selectPrevious selects the previous option in the list
func (s *selectorWidget) selectPrevious() {
	if len(s.options) == 0 {
		return
	}
	s.currentIndex = (s.currentIndex - 1 + len(s.options)) % len(s.options)
}

// selectByIndex selects an option by its index
func (s *selectorWidget) selectByIndex(index int) bool {
	if index >= 0 && index < len(s.options) {
		s.currentIndex = index
		return true
	}
	return false
}

// confirm confirms the current selection and calls the callback
func (s *selectorWidget) confirm() interface{} {
	s.deactivate()
	return s.getCurrentValue()
}

// render renders the selector widget to a string
func (s *selectorWidget) render() string {
	if !s.isActive || len(s.options) == 0 {
		return ""
	}

	var sb strings.Builder

	// Display title and instructions
	sb.WriteString("\n " + s.title + " (↑/↓ to navigate, Enter to select, ESC to cancel):\n\n")

	// Display options
	for i, option := range s.options {
		if i == s.currentIndex {
			// Highlight the selected option
			sb.WriteString(fmt.Sprintf(" > [*] %s\n", option.name))
		} else {
			sb.WriteString(fmt.Sprintf("   [ ] %s\n", option.name))
		}
	}

	return sb.String()
}

func helloMessage() Message {
	buf := strings.Builder{}
	buf.WriteString("Welcome to chait interactive mode!")
	buf.WriteString(fmt.Sprintf("\nProvider: %s (Model: %s, Temperature: %.1f)", api.GetActiveProvider().GetName(), api.GetActiveProvider().GetCurrentModel(), api.GetActiveProvider().GetCurrentTemperature()))
	buf.WriteString("\nType ':h' to see all available commands.")
	buf.WriteString("\n-----------------------------------")
	return Message{
		Type:    MessageTypeChait,
		Content: buf.String(),
	}
}

func helpMessage() Message {
	buf := strings.Builder{}
	buf.WriteString("-----------------------------------")
	buf.WriteString(fmt.Sprintf("\nProvider: %s (Model: %s, Temperature: %.1f)", api.GetActiveProvider().GetName(), api.GetActiveProvider().GetCurrentModel(), api.GetActiveProvider().GetCurrentTemperature()))
	buf.WriteString("\nAvailable commands:\n")
	buf.WriteString("- ':h' - Show this message\n")
	buf.WriteString("- ':p' - select providers\n")
	buf.WriteString("- ':m' - select models\n")
	buf.WriteString("- ':t' - Set the temperature\n")
	buf.WriteString("- ':k' - Set the API key\n")
	buf.WriteString("- ':c' - Start a new conversation\n")
	buf.WriteString("- 'ctrl+c' - Exit interactive mode\n")
	buf.WriteString("-----------------------------------")
	return Message{
		Type:    MessageTypeChait,
		Content: buf.String(),
	}
}

func systemMessage() Message {
	return Message{
		Type:    MessageTypeSystem,
		Content: "You are a helpful assistant.",
	}
}

type interactiveModel struct {
	messages    []Message
	input       []rune
	cursor      int
	respChan    <-chan provider.StreamResponse
	width       int
	height      int
	scrollPos   int
	enableInput bool

	// API key input mode
	apiKeyInputMode bool

	// Text selection related fields
	selecting      bool   // Whether we are currently selecting text
	selectionStart point  // Start position of selection
	selectionEnd   point  // End position of selection
	selectedText   string // The currently selected text

	// Selector widgets
	providerSelector    selectorWidget // Widget for selecting providers
	modelSelector       selectorWidget // Widget for selecting models
	temperatureSelector selectorWidget // Widget for selecting temperature presets
}

func (m interactiveModel) getSystemMessage() provider.ChatMessage {
	for _, msg := range m.messages {
		if msg.Type == MessageTypeSystem {
			return msg.ToChatMessage()
		}
	}
	// Return an empty chat message if no system message is found
	return provider.ChatMessage{}
}

func (m interactiveModel) getRecentMessages() []provider.ChatMessage {
	chatMessages := []provider.ChatMessage{}
	for i := len(m.messages) - 1; i >= 0; i-- {
		if m.messages[i].Type == MessageTypeAssistant || m.messages[i].Type == MessageTypeUser {
			chatMessages = append(chatMessages, m.messages[i].ToChatMessage())
			if len(chatMessages) >= 20 {
				break
			}
		}

	}
	// revert the messages
	for i, j := 0, len(chatMessages)-1; i < j; i, j = i+1, j-1 {
		chatMessages[i], chatMessages[j] = chatMessages[j], chatMessages[i]
	}

	// Add system message at the beginning and return
	return append([]provider.ChatMessage{m.getSystemMessage()}, chatMessages...)
}

func (m *interactiveModel) enterSettingAPIKeyMode() {
	m.apiKeyInputMode = true
	m.messages = append(m.messages, Message{
		Type:    MessageTypeChait,
		Content: fmt.Sprintf("Please enter your API key of %s:", api.GetActiveProvider().GetName()),
	})
	m.input = []rune{}
	m.cursor = 0
	m.enableInput = true
	m.scrollToBottom()
}

// updateSelectedText extracts the selected text based on selection points
func (m *interactiveModel) updateSelectedText() {
	// Get all formatted message lines
	allLines := m.getFormattedMessageLines()

	// Ensure selection points are ordered correctly
	start, end := m.selectionStart, m.selectionEnd

	// Swap if start is after end
	if start.line > end.line || (start.line == end.line && start.col > end.col) {
		start, end = end, start
	}

	// Ensure line indices are within bounds
	if start.line < 0 {
		start.line = 0
		start.col = 0
	}
	if start.line >= len(allLines) {
		start.line = len(allLines) - 1
		if start.line < 0 {
			start.line = 0
		}
	}
	if end.line >= len(allLines) {
		end.line = len(allLines) - 1
		if end.line < 0 {
			end.line = 0
		}
	}

	// Extract the selected text
	var selectedText strings.Builder

	for i := start.line; i <= end.line; i++ {
		if i >= len(allLines) {
			break
		}

		line := allLines[i]
		lineRunes := []rune(line)

		// Handle single line selection
		if start.line == end.line {
			// Convert visual column positions to rune indices
			startRuneIdx := visualColumnToRuneIndex(lineRunes, start.col)
			endRuneIdx := visualColumnToRuneIndex(lineRunes, end.col)

			// Ensure indices are within bounds
			if startRuneIdx < 0 {
				startRuneIdx = 0
			}
			if startRuneIdx > len(lineRunes) {
				startRuneIdx = len(lineRunes)
			}
			if endRuneIdx < 0 {
				endRuneIdx = 0
			}
			if endRuneIdx > len(lineRunes) {
				endRuneIdx = len(lineRunes)
			}

			if startRuneIdx < endRuneIdx {
				selectedText.WriteString(string(lineRunes[startRuneIdx:endRuneIdx]))
			}
		} else {
			// Handle multi-line selection
			if i == start.line {
				// First line - from start column to end of line
				startRuneIdx := visualColumnToRuneIndex(lineRunes, start.col)
				if startRuneIdx < 0 {
					startRuneIdx = 0
				}
				if startRuneIdx > len(lineRunes) {
					startRuneIdx = len(lineRunes)
				}

				selectedText.WriteString(string(lineRunes[startRuneIdx:]))
			} else if i == end.line {
				// Last line - from beginning to end column
				endRuneIdx := visualColumnToRuneIndex(lineRunes, end.col)
				if endRuneIdx < 0 {
					endRuneIdx = 0
				}
				if endRuneIdx > len(lineRunes) {
					endRuneIdx = len(lineRunes)
				}

				selectedText.WriteString("\n")
				selectedText.WriteString(string(lineRunes[:endRuneIdx]))
			} else {
				// Middle lines - entire line
				selectedText.WriteString("\n")
				selectedText.WriteString(line)
			}
		}
	}

	m.selectedText = selectedText.String()
}

// visualColumnToRuneIndex converts a visual column position to a rune index
// This handles wide characters like Chinese characters correctly
func visualColumnToRuneIndex(lineRunes []rune, visualColumn int) int {
	visualPos := 0
	for i, r := range lineRunes {
		// If we've reached or exceeded the visual column, return this index
		if visualPos >= visualColumn {
			return i
		}

		// Add the width of this rune to our visual position
		visualPos += runewidth.RuneWidth(r)
	}

	// If we get here, the visual column is beyond the end of the string
	return len(lineRunes)
}

func refreshConfig(m *interactiveModel) {
	activeProvider := api.GetActiveProvider()
	availableProviders := api.GetAvailableProviders()
	modelNames := activeProvider.GetAvailableModels()
	currentModel := activeProvider.GetCurrentModel()
	temperaturePresets := activeProvider.GetTemperaturePresets()
	currentTemperature := activeProvider.GetCurrentTemperature()
	currentProvider := activeProvider.GetName()

	// Find the current provider index in the list
	currentProviderIndex := 0
	providerOptions := make([]selectorOption, len(availableProviders))
	for i, provider := range availableProviders {
		ready := "Not Ready"
		if provider.IsReady() {
			ready = "Ready"
		}
		providerOptions[i] = selectorOption{
			name:  fmt.Sprintf("%s [%s]", provider.GetName(), ready),
			value: provider.GetName(),
		}
		if provider.GetName() == currentProvider {
			currentProviderIndex = i
		}
	}

	m.providerSelector.options = providerOptions
	m.providerSelector.currentIndex = currentProviderIndex

	// Find the current model index in the list
	currentModelIndex := 0
	modelOptions := make([]selectorOption, len(modelNames))
	for i, name := range modelNames {
		modelOptions[i] = selectorOption{
			name:  name,
			value: name,
		}
		if name == currentModel {
			currentModelIndex = i
		}
	}
	m.modelSelector.options = modelOptions
	m.modelSelector.currentIndex = currentModelIndex

	// Find the current temperature preset index in the list
	currentTemperatureIndex := 0
	temperatureOptions := make([]selectorOption, len(temperaturePresets))
	for i, preset := range temperaturePresets {
		temperatureOptions[i] = selectorOption{
			name:  fmt.Sprintf("%s (%.1f) - %s", preset.Name, preset.Value, preset.Description),
			value: preset.Value,
		}
		if preset.Value == currentTemperature {
			currentTemperatureIndex = i
		}
	}
	m.temperatureSelector.options = temperatureOptions
	m.temperatureSelector.currentIndex = currentTemperatureIndex
}

func initialInteractiveModel(input string) interactiveModel {
	hello := helloMessage()

	model := interactiveModel{
		messages:    []Message{hello, systemMessage()},
		input:       []rune{},
		cursor:      0,
		respChan:    nil,
		width:       80,
		height:      24,
		scrollPos:   0,
		enableInput: true,

		// Initialize selection fields
		selecting:      false,
		selectionStart: point{line: 0, col: 0},
		selectionEnd:   point{line: 0, col: 0},
		selectedText:   "",

		// Initialize provider selector widget
		providerSelector: selectorWidget{
			title:    "Select a provider",
			isActive: false,
		},

		// Initialize model selector widget
		modelSelector: selectorWidget{
			title:    "Select a model",
			isActive: false,
		},

		// Initialize temperature selector widget
		temperatureSelector: selectorWidget{
			title:    "Select a temperature preset",
			isActive: false,
		},
	}

	refreshConfig(&model)

	if input != "" {
		model.messages = append(model.messages, Message{
			Type:    MessageTypeUser,
			Content: input,
		})
	}

	return model
}

func (m interactiveModel) Init() tea.Cmd {
	// Request the terminal dimensions on startup
	var cmds []tea.Cmd
	cmds = append(cmds, tea.EnterAltScreen)

	// If there's a user message, automatically start streaming
	if len(m.messages) > 2 && m.messages[len(m.messages)-1].Type == MessageTypeUser {
		cmds = append(cmds, func() tea.Msg {
			return startStreamingMsg{}
		})
	}

	return tea.Batch(cmds...)
}

// Custom message types for streaming responses
type startStreamingMsg struct{}
type streamResponseMsg struct {
	Content string
	Done    bool
	Error   error
}

// Command to process streaming responses
func processStreamResponse(respChan <-chan provider.StreamResponse) tea.Cmd {
	return func() tea.Msg {
		resp, ok := <-respChan
		if !ok {
			return streamResponseMsg{Done: true}
		}
		return streamResponseMsg{
			Content: resp.Content,
			Done:    resp.Done,
			Error:   resp.Error,
		}
	}
}

func (m interactiveModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd tea.Cmd
	)

	switch msg := msg.(type) {
	// Handle window resize events
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case startStreamingMsg:
		// Check if the current provider is ready
		if !api.GetActiveProvider().IsReady() {
			// Provider is not ready, prompt for API key input
			m.enterSettingAPIKeyMode()
			return m, nil
		}

		// Start streaming chat request
		respChan, err := api.SendStreamingChatRequest(m.getRecentMessages())
		m.messages = append(m.messages, Message{
			Type:    MessageTypeAssistant,
			Content: "",
		})

		if err != nil {
			// Handle error by updating the last message
			lastIdx := len(m.messages) - 1
			m.messages[lastIdx] = Message{
				Type:    MessageTypeError,
				Content: err.Error(),
			}
			m.enableInput = true
			return m, nil
		}
		// Store the response channel in the model
		m.respChan = respChan
		return m, processStreamResponse(respChan)

	case streamResponseMsg:
		// Handle streaming response
		lastIdx := len(m.messages) - 1

		if msg.Error != nil {
			// Handle error
			m.messages[lastIdx] = Message{
				Type:    MessageTypeError,
				Content: msg.Error.Error(),
			}
			return m, nil
		}

		// Update the last message with new content
		m.messages[lastIdx] = Message{
			Type:    MessageTypeAssistant,
			Content: m.messages[lastIdx].Content + msg.Content,
		}

		// Auto-scroll to bottom when receiving new content
		m.scrollToBottom()

		// If not done, continue processing the stream
		if !msg.Done {
			// Continue processing the stream with the channel stored in the model
			return m, processStreamResponse(m.respChan)
		}
		m.enableInput = true
		return m, nil

	case tea.MouseMsg:
		mouseEvent := tea.MouseEvent(msg)

		// Handle mouse wheel events for scrolling
		switch mouseEvent.Button {
		case tea.MouseButtonWheelUp:
			m.scrollUp(3) // Scroll up 3 lines per wheel tick
			return m, nil
		case tea.MouseButtonWheelDown:
			m.scrollDown(3) // Scroll down 3 lines per wheel tick
			return m, nil
		case tea.MouseButtonLeft:
			// Handle text selection
			switch mouseEvent.Action {
			case tea.MouseActionPress:
				// Start selection
				m.selecting = true

				// Calculate the position in the text based on mouse coordinates
				// Adjust for scroll position
				linePos := mouseEvent.Y + m.scrollPos
				m.selectionStart = point{line: linePos, col: mouseEvent.X}
				m.selectionEnd = m.selectionStart
				return m, nil

			case tea.MouseActionMotion:
				// Continue selection if we're in selection mode
				if m.selecting {
					// Update the end point of the selection
					linePos := mouseEvent.Y + m.scrollPos
					m.selectionEnd = point{line: linePos, col: mouseEvent.X}

					// Extract the selected text
					m.updateSelectedText()
					return m, nil
				}

			case tea.MouseActionRelease:
				// End selection
				if m.selecting {
					// Update the end point of the selection
					linePos := mouseEvent.Y + m.scrollPos
					m.selectionEnd = point{line: linePos, col: mouseEvent.X}

					// Extract the selected text
					m.updateSelectedText()

					// Copy selected text to clipboard if not empty
					if m.selectedText != "" {
						err := clipboard.WriteAll(m.selectedText)
						if err != nil {
							// If clipboard fails, we still want to keep the selection visible
							// but we don't want to reset the selecting state
							return m, nil
						}

						// Keep selection visible for a moment after copying
						// We'll keep the selecting state true so the highlight remains visible
						return m, nil
					}

					// Only reset selecting state if there was no text or after copying
					m.selecting = false
					return m, nil
				}
			}
		}

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+p":
			// Enter provider switching mode
			m.providerSelector.activate()
			// Deactivate other selectors
			m.modelSelector.deactivate()
			m.temperatureSelector.deactivate()
			return m, nil
		case "ctrl+m":
			// Enter model switching mode
			m.modelSelector.activate()
			// Deactivate other selectors
			m.providerSelector.deactivate()
			m.temperatureSelector.deactivate()
			return m, nil
		case "ctrl+t":
			// Enter temperature switching mode
			m.temperatureSelector.activate()
			// Deactivate other selectors
			m.providerSelector.deactivate()
			m.modelSelector.deactivate()
			return m, nil
		case "pgup":
			m.scrollPageUp()
			return m, nil
		case "pgdown":
			m.scrollPageDown()
			return m, nil
		case "up":
			// Handle Up key for all selectors
			if m.providerSelector.isActive {
				m.providerSelector.selectPrevious()
				return m, nil
			} else if m.modelSelector.isActive {
				m.modelSelector.selectPrevious()
				return m, nil
			} else if m.temperatureSelector.isActive {
				m.temperatureSelector.selectPrevious()
				return m, nil
			}
			return m, nil
		case "down":
			// Handle Down key for all selectors
			if m.providerSelector.isActive {
				m.providerSelector.selectNext()
				return m, nil
			} else if m.modelSelector.isActive {
				m.modelSelector.selectNext()
				return m, nil
			} else if m.temperatureSelector.isActive {
				m.temperatureSelector.selectNext()
				return m, nil
			}
			return m, nil
		case "home":
			m.scrollToTop()
			return m, nil
		case "end":
			m.scrollToBottom()
			return m, nil
		case "alt+enter":
			newInput := make([]rune, len(m.input)+1)
			copy(newInput, m.input[:m.cursor])
			newInput[m.cursor] = '\n'
			copy(newInput[m.cursor+1:], m.input[m.cursor:])
			m.input = newInput
			m.cursor++
			return m, nil
		}
		// Handle keyboard shortcuts using string comparison to avoid conflicts

		// Handle other key types
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			// If in any selector mode, exit that mode instead of quitting
			if m.providerSelector.isActive {
				m.providerSelector.deactivate()
				refreshConfig(&m)
				return m, nil
			} else if m.modelSelector.isActive {
				m.modelSelector.deactivate()
				refreshConfig(&m)
				return m, nil
			} else if m.temperatureSelector.isActive {
				m.temperatureSelector.deactivate()
				refreshConfig(&m)
				return m, nil
			} else if m.respChan != nil {
				// If streaming is in progress, cancel it and reset
				m.respChan = nil
				m.enableInput = true
				return m, nil
			}
			return m, tea.Quit
		case tea.KeyEnter:
			// Handle Enter key based on current state
			// If in any selector mode, confirm selection and exit that mode
			if m.providerSelector.isActive {
				v := m.providerSelector.confirm()
				_ = api.SetActiveProvider(v.(string))
				refreshConfig(&m)
				return m, nil
			} else if m.modelSelector.isActive {
				v := m.modelSelector.confirm()
				_ = api.SetProviderModel(api.GetActiveProvider(), v.(string))
				refreshConfig(&m)
				return m, nil
			} else if m.temperatureSelector.isActive {
				v := m.temperatureSelector.confirm()
				_ = api.SetProviderTemperature(api.GetActiveProvider(), v.(float64))
				refreshConfig(&m)
				return m, nil
			} else if m.apiKeyInputMode {
				// Handle API key input
				apiKey := string(m.input)
				if apiKey == "" {
					return m, nil
				}

				// Set the API key
				err := api.SetAPIKey(apiKey)
				if err != nil {
					m.messages = append(m.messages, Message{
						Type:    MessageTypeError,
						Content: fmt.Sprintf("Error setting API key: %v", err),
					})
				} else {
					m.messages = append(m.messages, Message{
						Type:    MessageTypeChait,
						Content: fmt.Sprintf("API key for '%s' has been set successfully.", api.GetActiveProvider().GetName()),
					})
				}

				// Exit API key input mode
				m.apiKeyInputMode = false
				m.input = []rune{}
				m.cursor = 0
				return m, nil
			} else {
				// Handle normal Enter key press for sending messages
				userMsg := string(m.input)
				if userMsg == "" {
					return m, nil
				}

				// Add user message to the messages list
				m.messages = append(m.messages, Message{
					Type:    MessageTypeUser,
					Content: userMsg,
				})
				m.input = []rune{}
				m.cursor = 0

				// Auto-scroll to bottom when sending a new message
				m.scrollToBottom()
				m.enableInput = false

				// Return command to start streaming chat request
				return m, func() tea.Msg {
					return startStreamingMsg{}
				}
			}

		case tea.KeyLeft:
			if m.cursor > 0 {
				m.cursor--
			}
		case tea.KeyRight:
			if m.cursor < len(m.input) {
				m.cursor++
			}
		case tea.KeyDelete:
			if m.cursor < len(m.input) {
				// Delete character at cursor position
				newInput := make([]rune, len(m.input)-1)
				copy(newInput, m.input[:m.cursor])
				copy(newInput[m.cursor:], m.input[m.cursor+1:])
				m.input = newInput
			}
		case tea.KeyBackspace:
			if m.cursor > 0 {
				// Delete character before cursor position
				newInput := make([]rune, len(m.input)-1)
				copy(newInput, m.input[:m.cursor-1])
				copy(newInput[m.cursor-1:], m.input[m.cursor:])
				m.input = newInput
				m.cursor--
			}
		case tea.KeySpace:
			m.input = append(m.input, ' ')
			m.cursor++
		// case tea.KeyCtrlV:
		// 	pastedText, err := clipboard.ReadAll()
		// 	if err == nil && pastedText != "" {
		// 		// Convert processed text to runes to handle Unicode correctly
		// 		processedRunes := []rune(pastedText)

		// 		// Insert pasted text at cursor position
		// 		newInput := make([]rune, len(m.input)+len(processedRunes))
		// 		copy(newInput, m.input[:m.cursor])
		// 		copy(newInput[m.cursor:], processedRunes)
		// 		copy(newInput[m.cursor+len(processedRunes):], m.input[m.cursor:])
		// 		m.input = newInput
		// 		m.cursor += len(processedRunes)
		// 		return m, nil
		// 	}

		case tea.KeyRunes:

			// Handle number key selection for all selectors
			if len(m.input) == 1 && m.input[0] >= '1' && m.input[0] <= '9' {
				// Convert the character to an index (0-based)
				selectedIndex := int(m.input[0] - '1')

				// Apply to the active selector
				if m.providerSelector.isActive {
					if m.providerSelector.selectByIndex(selectedIndex) {
						m.providerSelector.confirm()
					}
					return m, nil
				} else if m.modelSelector.isActive {
					if m.modelSelector.selectByIndex(selectedIndex) {
						m.modelSelector.confirm()
					}
					return m, nil
				} else if m.temperatureSelector.isActive {
					if m.temperatureSelector.selectByIndex(selectedIndex) {
						m.temperatureSelector.confirm()
					}
					return m, nil
				}
			}

			// Normal text input handling
			newInput := make([]rune, len(m.input)+len(msg.Runes))
			copy(newInput, m.input[:m.cursor])
			copy(newInput[m.cursor:], msg.Runes)
			copy(newInput[m.cursor+len(msg.Runes):], m.input[m.cursor:])

			// Handle help keys for selectors
			if len(newInput) > 0 && newInput[0] == ':' {
				switch string(newInput[1:]) {
				case "p": // :p - Switch provider
					// Enter provider switching mode
					m.providerSelector.activate()
					// Deactivate other selectors
					m.modelSelector.deactivate()
					m.temperatureSelector.deactivate()
					m.input = []rune{}
					m.cursor = 0
					return m, nil
				case "m": // :m - Switch model
					// Enter model switching mode
					m.modelSelector.activate()
					// Deactivate other selectors
					m.providerSelector.deactivate()
					m.temperatureSelector.deactivate()
					m.input = []rune{}
					m.cursor = 0
					return m, nil
				case "t": // :t - Switch temperature
					// Enter temperature switching mode
					m.temperatureSelector.activate()
					// Deactivate other selectors
					m.providerSelector.deactivate()
					m.modelSelector.deactivate()
					m.input = []rune{}
					m.cursor = 0
					return m, nil
				case "h": // :h - Show help
					m.messages = append(m.messages, helpMessage())
					m.input = []rune{}
					m.cursor = 0
					m.scrollToBottom()
					return m, nil
				case "k": // :k - Set API key
					m.enterSettingAPIKeyMode()
					return m, nil
				case "c": // :c - Start a new conversation
					m.messages = []Message{systemMessage()}
					m.input = []rune{}
					m.cursor = 0
					m.scrollToBottom()
					return m, nil
				}
			}

			m.input = newInput
			m.cursor += len(msg.Runes)
		}
	}

	return m, cmd
}

// Format messages with proper wrapping for the viewport
func (m interactiveModel) formatMessages() string {
	var buf strings.Builder
	for i, msg := range m.messages {
		// Add a separator between messages except for the first one
		if i > 0 {
			buf.WriteString("\n\n")
		}

		prefixLen := 0
		typeStr := ""
		if msg.Type == MessageTypeUser {
			typeStr = "> "
		} else if msg.Type != MessageTypeChait {
			typeStr = string(msg.Type) + ": "
		}
		buf.WriteString(typeStr)
		prefixLen = len(typeStr)

		// Handle text wrapping for the content
		if m.width > 0 {
			wrappedContent := wrapText(msg.Content, m.width, prefixLen)
			buf.WriteString(wrappedContent)
		} else {
			// Fallback if width is not available
			buf.WriteString(msg.Content)
		}
	}
	return buf.String()
}

// Wrap text to fit within the terminal width
func wrapText(text string, width, prefixLen int) string {
	if width <= 0 {
		return text
	}

	var result strings.Builder
	lines := strings.Split(text, "\n")

	for lineIdx, line := range lines {
		if lineIdx > 0 {
			result.WriteString("\n")
		}

		runes := []rune(line)

		// Only apply prefix indent to the first line of each message
		currentWidth := width - prefixLen

		for len(runes) > 0 {
			// Find a good breaking point
			breakPoint := findBreakPoint(runes, currentWidth)
			result.WriteString(string(runes[:breakPoint]))

			// Add newline only if there's more text to process
			runes = runes[breakPoint:]
			if len(runes) > 0 {
				result.WriteString("\n")
				currentWidth = width
			}
		}
	}

	return result.String()
}

// Find a suitable breaking point for text wrapping
// Properly handles Unicode character width
func findBreakPoint(runes []rune, width int) int {
	if len(runes) == 0 {
		return 0
	}

	// Calculate the visual width of the text
	visualWidth := 0
	pos := 0

	for i, r := range runes {
		charWidth := runewidth.RuneWidth(r)
		// If adding this character would exceed the width
		if visualWidth+charWidth > width {
			pos = i
			break
		}
		visualWidth += charWidth
		pos = i + 1
	}

	// If all characters fit within the width
	if pos == len(runes) {
		return pos
	}

	// Try to break at whitespace before the cutoff point
	for i := pos - 1; i > 0; i-- {
		if runes[i] == ' ' {
			return i + 1 // Include the space in the current line
		}
	}

	// If no suitable whitespace breakpoint found, use the calculated position
	return pos
}

// Get the total number of lines in the formatted messages
func (m interactiveModel) getFormattedMessageLines() []string {
	formatted := m.formatMessages()
	return strings.Split(formatted, "\n")
}

// Scroll handling methods
func (m *interactiveModel) scrollUp(lines int) {
	m.scrollPos -= lines
	if m.scrollPos < 0 {
		m.scrollPos = 0
	}
}

func (m *interactiveModel) scrollDown(lines int) {
	allLines := m.getFormattedMessageLines()
	maxScroll := len(allLines) - (m.height - 3) // Reserve space for input area
	if maxScroll < 0 {
		maxScroll = 0
	}

	m.scrollPos += lines
	if m.scrollPos > maxScroll {
		m.scrollPos = maxScroll
	}
}

func (m *interactiveModel) scrollPageUp() {
	m.scrollUp(m.height / 2)
}

func (m *interactiveModel) scrollPageDown() {
	m.scrollDown(m.height / 2)
}

func (m *interactiveModel) scrollToTop() {
	m.scrollPos = 0
}

func (m *interactiveModel) scrollToBottom() {
	allLines := m.getFormattedMessageLines()
	maxScroll := len(allLines) - (m.height - 3) // Reserve space for input area
	if maxScroll < 0 {
		maxScroll = 0
	}
	m.scrollPos = maxScroll
}

func (m interactiveModel) View() string {
	// Build the UI
	var sb strings.Builder
	var input strings.Builder

	// Check if we're in provider selection mode
	if m.providerSelector.isActive {
		// Use the provider selector widget to render the UI
		return m.providerSelector.render()
	} else if m.modelSelector.isActive {
		// Use the model selector widget to render the UI
		return m.modelSelector.render()
	} else if m.temperatureSelector.isActive {
		// Use the temperature selector widget to render the UI
		return m.temperatureSelector.render()
	}

	// Get all lines from formatted messages
	allLines := m.getFormattedMessageLines()

	// Calculate visible portion based on scroll position
	visibleHeight := m.height - 3 // Reserve space for input area
	if visibleHeight < 1 {
		visibleHeight = 1
	}

	startLine := m.scrollPos
	endLine := startLine + visibleHeight

	if startLine >= len(allLines) {
		startLine = max(0, len(allLines)-1)
	}

	if endLine > len(allLines) {
		endLine = len(allLines)
	}

	// Determine if we have an active selection
	hasSelection := m.selecting && (m.selectionStart.line != m.selectionEnd.line || m.selectionStart.col != m.selectionEnd.col)

	// Ensure selection points are ordered correctly for rendering
	selStart, selEnd := m.selectionStart, m.selectionEnd
	if hasSelection {
		if selStart.line > selEnd.line || (selStart.line == selEnd.line && selStart.col > selEnd.col) {
			selStart, selEnd = selEnd, selStart
		}
	}

	// Render only the visible portion of messages
	for i := startLine; i < endLine; i++ {
		if i < len(allLines) {
			line := allLines[i]

			// Check if this line is part of the selection
			if hasSelection && i >= selStart.line && i <= selEnd.line {
				// This line has some selection
				lineRunes := []rune(line)

				// Determine selection start and end rune indices for this line
				startIdx, endIdx := 0, len(lineRunes)
				if i == selStart.line {
					startIdx = visualColumnToRuneIndex(lineRunes, selStart.col)
				}
				if i == selEnd.line {
					endIdx = visualColumnToRuneIndex(lineRunes, selEnd.col)
				}

				// Ensure indices are within bounds
				if startIdx < 0 {
					startIdx = 0
				}
				if startIdx > len(lineRunes) {
					startIdx = len(lineRunes)
				}
				if endIdx < 0 {
					endIdx = 0
				}
				if endIdx > len(lineRunes) {
					endIdx = len(lineRunes)
				}

				// Render the line with highlighted selection
				if startIdx < endIdx {
					// Write the part before selection
					sb.WriteString(string(lineRunes[:startIdx]))

					// Write the selected part with reverse video (highlighted)
					sb.WriteString("\x1b[7m") // Terminal escape code for reverse video
					sb.WriteString(string(lineRunes[startIdx:endIdx]))
					sb.WriteString("\x1b[0m") // Reset formatting

					// Write the part after selection
					sb.WriteString(string(lineRunes[endIdx:]))
				} else {
					// No selection on this line (can happen due to bounds checking)
					sb.WriteString(line)
				}
			} else {
				// No selection on this line
				sb.WriteString(line)
			}

			sb.WriteString("\n")
		}
	}

	// Calculate if we're at the bottom of the conversation
	allLinesCount := len(allLines)
	maxScroll := allLinesCount - visibleHeight
	if maxScroll < 0 {
		maxScroll = 0
	}
	isAtBottom := m.scrollPos >= maxScroll

	// Only show input prompt when at the bottom of the conversation
	if m.enableInput && isAtBottom {

		// Render the input with cursor
		inputBeforeCursor := string(m.input[:m.cursor])
		inputAfterCursor := string(m.input[m.cursor:])
		input.WriteString(inputBeforeCursor)
		input.WriteString("|")
		input.WriteString(inputAfterCursor)

		sb.WriteString("\n> ")
		sb.WriteString(wrapText(input.String(), m.width, 2))
	}

	return sb.String()
}

func StartInteractiveMode(input string) error {
	p := tea.NewProgram(
		initialInteractiveModel(input),
		tea.WithAltScreen(),       // Use the full terminal in alternate screen mode
		tea.WithMouseAllMotion(),  // Enable mouse support for all motion
		tea.WithMouseCellMotion(), // Enable mouse cell motion events
	)

	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		return err
	}
	return nil
}
