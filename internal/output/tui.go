package output

import (
	"fmt"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sl/prompt-builder-agent/internal/llm"
)

// TUIRenderer renders streaming output using Bubbletea TUI
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
	viewport viewport.Model
	content  string
	done     bool
	err      error
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
		}

	case tea.WindowSizeMsg:
		m.viewport.Width = msg.Width - 4
		m.viewport.Height = msg.Height - 4

	case chunkMsg:
		m.content += string(msg)
		m.viewport.SetContent(m.content)

	case doneMsg:
		m.done = true
		return m, tea.Quit

	case errorMsg:
		m.err = error(msg)
		m.viewport.SetContent(fmt.Sprintf("Error: %v", m.err))
		return m, tea.Quit
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

	return fmt.Sprintf("%s %s\n%s\n\nPress 'q' or Ctrl+C to exit", header, status, m.viewport.View())
}
