package output

import (
	"fmt"
	"os/exec"
	"strings"

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

// Model for Bubbletea
type tuiModel struct {
	viewport   viewport.Model
	content    string
	done       bool
	err        error
	copied     bool
	copyStatus string
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
	return nil
}

// Update implements tea.Model
func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
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
		}

	case tea.WindowSizeMsg:
		m.viewport.Width = msg.Width - 4
		m.viewport.Height = msg.Height - 4

	case chunkMsg:
		m.content += string(msg)
		m.viewport.SetContent(formatMarkdown(m.content))
		// Reset copy status when new content arrives
		if m.copied {
			m.copied = false
			m.copyStatus = ""
		}

	case doneMsg:
		m.done = true
		// Don't quit immediately - let user read the output and press 'q' to exit
		return m, nil

	case errorMsg:
		m.err = error(msg)
		m.viewport.SetContent(fmt.Sprintf("Error: %v", m.err))
		// Don't quit immediately - let user read the error and press 'q' to exit
		return m, nil
	}

	return m, nil
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
	} else {
		status = lipgloss.NewStyle().
			Foreground(lipgloss.Color("3")).
			Render("⏳ Streaming...")
	}

	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("39")).
		Render("AI Response")

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
		instructions = "Press 'c' to copy | 'q' or Ctrl+C to exit"
	} else {
		instructions = "Press 'q' or Ctrl+C to exit"
	}

	return fmt.Sprintf("%s %s\n%s\n\n%s%s",
		header, status, m.viewport.View(), instructions, footer)
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
