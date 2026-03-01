package cli

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type editorKeyMap struct {
	Edit    key.Binding
	Send    key.Binding
	Cancel  key.Binding
	Confirm key.Binding
}

var editorKeys = editorKeyMap{
	Edit:    key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "edit")),
	Send:    key.NewBinding(key.WithKeys("s", "enter"), key.WithHelp("s/enter", "send")),
	Cancel:  key.NewBinding(key.WithKeys("q", "esc", "ctrl+c"), key.WithHelp("q/esc", "cancel")),
	Confirm: key.NewBinding(key.WithKeys("y"), key.WithHelp("y", "confirm")),
}

type promptEditorModel struct {
	prompt   string
	edited   string
	mode     string // "preview", "edit", "confirm"
	viewport viewport.Model
	width    int
	height   int
	showHelp bool
}

func newPromptEditorModel(prompt string) promptEditorModel {
	vp := viewport.New(80, 20)
	vp.Style = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62"))
	vp.SetContent(prompt)

	return promptEditorModel{
		prompt:   prompt,
		edited:   prompt,
		mode:     "preview",
		viewport: vp,
		showHelp: true,
	}
}

func (m promptEditorModel) Init() tea.Cmd {
	return nil
}

func (m promptEditorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.mode {
		case "preview":
			switch {
			case key.Matches(msg, editorKeys.Edit):
				// Open external editor
				edited, err := openExternalEditor(m.prompt)
				if err != nil {
					// If editor fails, just proceed with original
					m.mode = "confirm"
					return m, nil
				}
				m.edited = edited
				m.viewport.SetContent(edited)
				m.mode = "confirm"
				return m, nil
			case key.Matches(msg, editorKeys.Send):
				m.mode = "confirm"
				return m, nil
			case key.Matches(msg, editorKeys.Cancel):
				return m, tea.Quit
			default:
				// Allow scrolling in preview
				m.viewport, cmd = m.viewport.Update(msg)
				return m, cmd
			}

		case "confirm":
			switch {
			case key.Matches(msg, editorKeys.Confirm):
				// User confirmed, return edited prompt
				return m, tea.Quit
			case key.Matches(msg, editorKeys.Cancel):
				// Go back to preview
				m.mode = "preview"
				return m, nil
			default:
				// Allow scrolling in confirm
				m.viewport, cmd = m.viewport.Update(msg)
				return m, cmd
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewport.Width = msg.Width - 4
		m.viewport.Height = msg.Height - 10
		return m, nil
	}

	return m, cmd
}

func (m promptEditorModel) View() string {
	var s strings.Builder

	// Header
	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("39")).
		Padding(0, 1).
		Render("📝 Prompt Preview & Editor")

	s.WriteString(header + "\n\n")

	switch m.mode {
	case "preview":
		// Preview mode
		info := lipgloss.NewStyle().
			Foreground(lipgloss.Color("8")).
			Render(fmt.Sprintf("Engine: %s | Length: %d chars", "LLM", len(m.prompt)))

		s.WriteString(info + "\n\n")

		// Prompt content with border
		promptBox := lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(1, 2).
			Width(m.width - 4).
			Render(m.prompt)

		s.WriteString(promptBox + "\n\n")

		// Help text
		var help string
		if m.mode == "preview" {
			help = lipgloss.NewStyle().
				Foreground(lipgloss.Color("8")).
				Render("Press 'e' to edit | 's' or Enter to send | 'q' to cancel | ↑↓ to scroll")
		} else {
			help = lipgloss.NewStyle().
				Foreground(lipgloss.Color("8")).
				Render("Press 'y' to confirm and send | 'q' to go back | ↑↓ to scroll")
		}
		s.WriteString(help)

	case "confirm":
		// Confirm mode
		info := lipgloss.NewStyle().
			Foreground(lipgloss.Color("2")).
			Bold(true).
			Render("✓ Ready to send")

		s.WriteString(info + "\n\n")

		// Show what will be sent
		var contentToShow string
		if m.edited != m.prompt {
			contentToShow = m.edited
			diff := lipgloss.NewStyle().
				Foreground(lipgloss.Color("3")).
				Render("(Edited - " + fmt.Sprintf("%d chars", len(m.edited)) + ")")
			s.WriteString(diff + "\n\n")
		} else {
			contentToShow = m.prompt
		}

		promptBox := lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("2")).
			Padding(1, 2).
			Width(m.width - 4).
			Render(contentToShow)

		s.WriteString(promptBox + "\n\n")

		// Help text
		help := lipgloss.NewStyle().
			Foreground(lipgloss.Color("8")).
			Render("Press 'y' to confirm and send | 'q' to go back")

		s.WriteString(help)
	}

	return lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Render(s.String())
}

// PromptEditor allows user to preview and edit prompt before sending
func PromptEditor(prompt string) (string, bool, error) {
	m := newPromptEditorModel(prompt)
	p := tea.NewProgram(m, tea.WithAltScreen())

	finalModel, err := p.Run()
	if err != nil {
		return "", false, fmt.Errorf("editor error: %w", err)
	}

	model := finalModel.(promptEditorModel)

	// Check if user cancelled (didn't confirm)
	if model.mode != "confirm" || model.edited == "" {
		return "", false, nil
	}

	return model.edited, true, nil
}

// openExternalEditor opens the system's default editor (vim/nano/vi) to edit the prompt
func openExternalEditor(content string) (string, error) {
	// Create temp file
	tmpFile, err := os.CreateTemp("", "prompt-edit-*.txt")
	if err != nil {
		return content, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write content to temp file
	if _, err := tmpFile.WriteString(content); err != nil {
		return content, fmt.Errorf("failed to write temp file: %w", err)
	}
	tmpFile.Close()

	// Determine editor (prefer $EDITOR, fallback to vim/nano)
	editor := os.Getenv("EDITOR")
	if editor == "" {
		// Try common editors
		for _, e := range []string{"vim", "nano", "vi"} {
			if _, err := exec.LookPath(e); err == nil {
				editor = e
				break
			}
		}
		if editor == "" {
			return content, fmt.Errorf("no editor found (set $EDITOR or install vim/nano)")
		}
	}

	// Open editor
	cmd := exec.Command(editor, tmpFile.Name())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return content, fmt.Errorf("editor failed: %w", err)
	}

	// Read edited content
	editedBytes, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		return content, fmt.Errorf("failed to read edited file: %w", err)
	}

	return strings.TrimSpace(string(editedBytes)), nil
}
