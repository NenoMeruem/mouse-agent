package output

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sl/prompt-builder-agent/internal/llm"
)

// TUIRenderer renders streaming output using Bubbletea TUI with markdown support
type TUIRenderer struct {
	content string
	done    bool
	err     error
}

// NewTUIRenderer creates a new TUI renderer
func NewTUIRenderer() *TUIRenderer {
	return &TUIRenderer{}
}

// RenderStream implements StreamRenderer interface
func (r *TUIRenderer) RenderStream(ch <-chan llm.Chunk) error {
	m := newModel()

	// Start the tea program
	p := tea.NewProgram(m, tea.WithAltScreen())

	// Create a channel for chunks to communicate with the program
	go func() {
		for chunk := range ch {
			if chunk.Err != nil {
				p.Send(errorMsg(chunk.Err))
			} else if chunk.Done {
				p.Send(doneMsg{})
			} else {
				p.Send(chunkMsg(chunk.Text))
			}
		}
	}()

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("TUI render error: %w", err)
	}

	return nil
}

// Messages for the tea program
type chunkMsg string
type doneMsg struct{}
type errorMsg error
type tickMsg time.Time

// Model for Bubbletea
type tuiModel struct {
	viewport   viewport.Model
	content    string
	done       bool
	err        error
	copied     bool
	copyStatus string
	loading    bool
	spinner    int
}

// newModel creates a new TUI model
func newModel() tuiModel {
	vp := viewport.New(80, 20)
	vp.Style = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62"))

	return tuiModel{
		viewport: vp,
	}
}

// Init implements tea.Model
func (m tuiModel) Init() tea.Cmd {
	// Start loading animation if no content
	if m.content == "" {
		return tea.Tick(time.Millisecond*100, func(t time.Time) tea.Msg {
			return tickMsg(t)
		})
	}
	return nil
}

// Update implements tea.Model
func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle scroll keys - delegate to viewport first
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "c":
			// Copy content to clipboard
			if m.content != "" {
				if err := copyToClipboard(m.content); err != nil {
					m.copyStatus = fmt.Sprintf("❌ Copy failed: %v", err)
				} else {
					m.copied = true
					m.copyStatus = "✓ Copied to clipboard!"
				}
			} else {
				m.copyStatus = "❌ No content to copy"
			}
			return m, nil
		case "up", "k":
			m.viewport.LineUp(1)
			return m, nil
		case "down", "j":
			m.viewport.LineDown(1)
			return m, nil
		case "pgup":
			m.viewport.ViewUp()
			return m, nil
		case "pgdown", " ":
			m.viewport.ViewDown()
			return m, nil
		case "home", "g":
			m.viewport.GotoTop()
			return m, nil
		case "end", "G":
			m.viewport.GotoBottom()
			return m, nil
		default:
			// Let viewport handle other keys
			m.viewport, cmd = m.viewport.Update(msg)
			return m, cmd
		}

	case tea.WindowSizeMsg:
		// Recalculate viewport dimensions accounting for header and footer
		headerHeight := 2
		footerHeight := 2
		if m.copyStatus != "" {
			footerHeight += 2
		}
		m.viewport.Width = msg.Width - 4
		m.viewport.Height = msg.Height - headerHeight - footerHeight - 2
		return m, nil

	case tickMsg:
		// Loading animation tick
		if m.content == "" && !m.done && m.err == nil {
			m.loading = true
			m.spinner = (m.spinner + 1) % 4
			return m, tea.Tick(time.Millisecond*100, func(t time.Time) tea.Msg {
				return tickMsg(t)
			})
		}
		return m, nil

	case chunkMsg:
		m.content += string(msg)
		m.loading = false
		m.viewport.SetContent(formatMarkdown(m.content))
		// Auto-scroll to bottom when new content arrives
		m.viewport.GotoBottom()
		// Reset copy status when new content arrives
		if m.copied {
			m.copied = false
			m.copyStatus = ""
		}
		return m, nil

	case doneMsg:
		m.done = true
		m.loading = false
		// Don't quit immediately - let user read the output and press 'q' to exit
		return m, nil

	case errorMsg:
		m.err = error(msg)
		m.loading = false
		m.viewport.SetContent(fmt.Sprintf("Error: %v", m.err))
		// Don't quit immediately - let user read the error and press 'q' to exit
		return m, nil
	}

	return m, cmd
}

// View implements tea.Model
func (m tuiModel) View() string {
	var status string
	if m.done {
		status = lipgloss.NewStyle().
			Foreground(lipgloss.Color("2")).
			Render("✓ Done")
	} else if m.err != nil {
		status = lipgloss.NewStyle().
			Foreground(lipgloss.Color("1")).
			Render(fmt.Sprintf("✗ Error: %v", m.err))
	} else if m.loading && m.content == "" {
		// Show animated loading spinner
		spinnerChars := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		spinner := spinnerChars[m.spinner%len(spinnerChars)]
		status = lipgloss.NewStyle().
			Foreground(lipgloss.Color("3")).
			Render(fmt.Sprintf("%s Waiting for response...", spinner))
	} else {
		status = lipgloss.NewStyle().
			Foreground(lipgloss.Color("3")).
			Render("⏳ Streaming...")
	}

	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("39")).
		Render("AI Response")

	// Build viewport content
	var viewportContent string
	if m.content == "" && m.loading {
		// Show loading animation in viewport
		loadingText := renderLoadingAnimation(m.spinner)
		m.viewport.SetContent(loadingText)
		viewportContent = m.viewport.View()
	} else if m.content == "" {
		// Show empty state
		emptyText := lipgloss.NewStyle().
			Foreground(lipgloss.Color("8")).
			Italic(true).
			Render("Waiting for AI response...")
		m.viewport.SetContent(emptyText)
		viewportContent = m.viewport.View()
	} else {
		viewportContent = m.viewport.View()
	}

	// Build footer with instructions
	var footer string
	if m.copyStatus != "" {
		copyMsg := lipgloss.NewStyle().
			Foreground(lipgloss.Color("2")).
			Render(m.copyStatus)
		if strings.HasPrefix(m.copyStatus, "❌") {
			copyMsg = lipgloss.NewStyle().
				Foreground(lipgloss.Color("1")).
				Render(m.copyStatus)
		}
		footer = fmt.Sprintf("\n%s\n", copyMsg)
	}

	// Build instructions based on state
	var instructions string
	if m.content != "" {
		instructions = "↑↓/PgUp/PgDn/Home/End: scroll | 'c': copy | 'q': quit"
	} else {
		instructions = "Press 'q' or Ctrl+C to exit"
	}

	return fmt.Sprintf("%s %s\n%s\n\n%s%s",
		header, status, viewportContent, instructions, footer)
}

// renderLoadingAnimation creates a loading animation
func renderLoadingAnimation(spinnerIdx int) string {
	spinnerChars := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	spinnerChar := spinnerChars[spinnerIdx%len(spinnerChars)]

	lines := []string{
		"",
		"",
		lipgloss.NewStyle().
			Foreground(lipgloss.Color("3")).
			Bold(true).
			Render(fmt.Sprintf("  %s  Waiting for AI response...", spinnerChar)),
		"",
		lipgloss.NewStyle().
			Foreground(lipgloss.Color("8")).
			Italic(true).
			Render("  This may take a few moments"),
		"",
		"",
	}

	return strings.Join(lines, "\n")
}

// formatMarkdown applies basic markdown styling to text
func formatMarkdown(text string) string {
	lines := strings.Split(text, "\n")
	var result []string

	codeBlock := false

	for _, line := range lines {
		// Code blocks (```...```)
		if strings.HasPrefix(line, "```") {
			codeBlock = !codeBlock
			result = append(result, lipgloss.NewStyle().
				Foreground(lipgloss.Color("8")).
				Italic(true).
				Render(line))
			continue
		}

		if codeBlock {
			result = append(result, lipgloss.NewStyle().
				Background(lipgloss.Color("237")).
				Foreground(lipgloss.Color("2")).
				Render("  "+line))
			continue
		}

		// Headings (# ## ###)
		if strings.HasPrefix(line, "### ") {
			result = append(result, lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("208")).
				Render("  "+strings.TrimPrefix(line, "### ")))
			continue
		}

		if strings.HasPrefix(line, "## ") {
			result = append(result, lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("33")).
				Render(strings.TrimPrefix(line, "## ")))
			continue
		}

		if strings.HasPrefix(line, "# ") {
			result = append(result, lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("39")).
				Render(strings.TrimPrefix(line, "# ")))
			continue
		}

		// Bold (**text**)
		line = styleBold(line)

		// Italic (*text*)
		line = styleItalic(line)

		// Inline code (`code`)
		line = styleInlineCode(line)

		// Lists (- or *)
		if strings.HasPrefix(line, "- ") {
			result = append(result, lipgloss.NewStyle().
				Foreground(lipgloss.Color("2")).
				Render("• "+strings.TrimPrefix(line, "- ")))
			continue
		}

		result = append(result, line)
	}

	return strings.Join(result, "\n")
}

// styleBold applies bold style to **text** patterns
func styleBold(text string) string {
	parts := strings.Split(text, "**")
	for i := 1; i < len(parts); i += 2 {
		if i < len(parts) {
			parts[i] = lipgloss.NewStyle().
				Bold(true).
				Render(parts[i])
		}
	}
	return strings.Join(parts, "")
}

// styleItalic applies italic style to *text* patterns
func styleItalic(text string) string {
	parts := strings.Split(text, "*")
	for i := 1; i < len(parts); i += 2 {
		if i < len(parts) && !strings.Contains(parts[i], "*") {
			parts[i] = lipgloss.NewStyle().
				Italic(true).
				Foreground(lipgloss.Color("11")).
				Render(parts[i])
		}
	}
	return strings.Join(parts, "*")
}

// styleInlineCode applies code style to `code` patterns
func styleInlineCode(text string) string {
	parts := strings.Split(text, "`")
	for i := 1; i < len(parts); i += 2 {
		if i < len(parts) {
			parts[i] = lipgloss.NewStyle().
				Background(lipgloss.Color("237")).
				Foreground(lipgloss.Color("10")).
				Render(parts[i])
		}
	}
	return strings.Join(parts, "`")
}

// copyToClipboard copies text to clipboard using pbcopy (macOS)
func copyToClipboard(text string) error {
	cmd := exec.Command("pbcopy")
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}
