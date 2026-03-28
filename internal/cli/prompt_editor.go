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
	Edit:    key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "edit in $EDITOR")),
	Send:    key.NewBinding(key.WithKeys("s", "enter"), key.WithHelp("s/enter", "send")),
	Cancel:  key.NewBinding(key.WithKeys("q", "esc", "ctrl+c"), key.WithHelp("q/esc", "cancel")),
	Confirm: key.NewBinding(key.WithKeys("y"), key.WithHelp("y", "confirm & send")),
}

type promptEditorModel struct {
	prompt    string
	edited    string
	engine    string
	mode      string // "confirm"
	viewport  viewport.Model
	width     int
	height    int
	cancelled bool
}

func newPromptEditorModel(prompt, engine string) promptEditorModel {
	vp := viewport.New(80, 20)
	vp.Style = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("10"))
	vp.SetContent(prompt)

	return promptEditorModel{
		prompt:   prompt,
		edited:   prompt,
		engine:   engine,
		mode:     "confirm",
		viewport: vp,
	}
}

func (m promptEditorModel) Init() tea.Cmd { return nil }

func (m promptEditorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, editorKeys.Confirm):
			return m, tea.Quit
		case key.Matches(msg, editorKeys.Edit):
			edited, err := openExternalEditor(m.edited)
			if err == nil {
				m.edited = edited
				m.viewport.SetContent(edited)
			}
			return m, nil
		case key.Matches(msg, editorKeys.Cancel):
			m.cancelled = true
			return m, tea.Quit
		default:
			m.viewport, cmd = m.viewport.Update(msg)
			return m, cmd
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		headerH := 5 // title + meta + padding
		footerH := 2 // statusbar
		m.viewport.Width = msg.Width - 4
		m.viewport.Height = msg.Height - headerH - footerH
		return m, nil
	}

	return m, cmd
}

func (m promptEditorModel) View() string {
	width := m.width
	if width == 0 {
		width = 80
	}

	// ── Header ───────────────────────────────────────────────
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("39")).
		Render("📝 Prompt Preview & Editor")

	var engineBadge string
	if m.engine != "" && m.engine != "none" {
		engineBadge = lipgloss.NewStyle().
			Background(lipgloss.Color("55")).
			Foreground(lipgloss.Color("183")).
			Padding(0, 1).
			Render(m.engine)
	}

	headerRight := engineBadge
	header := lipgloss.NewStyle().Width(width).Render(
		lipgloss.JoinHorizontal(lipgloss.Left,
			title,
			lipgloss.NewStyle().
				Width(width-lipgloss.Width(title)-lipgloss.Width(headerRight)).
				Render(""),
			headerRight,
		),
	)

	// ── Meta row ─────────────────────────────────────────────
	statusText := lipgloss.NewStyle().
		Foreground(lipgloss.Color("10")).
		Bold(true).
		Render("✓ Ready to send")

	charCount := len([]rune(m.edited))
	tokenEst := charCount / 4
	if tokenEst < 1 && charCount > 0 {
		tokenEst = 1
	}
	metaRight := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8")).
		Render(fmt.Sprintf("%d chars · ~%d tokens", charCount, tokenEst))

	if m.edited != m.prompt {
		metaRight = lipgloss.NewStyle().
			Foreground(lipgloss.Color("11")).
			Render(fmt.Sprintf("(edited) %d chars · ~%d tokens", charCount, tokenEst))
	}

	meta := lipgloss.NewStyle().Width(width).Render(
		lipgloss.JoinHorizontal(lipgloss.Left,
			statusText,
			lipgloss.NewStyle().
				Width(width-lipgloss.Width(statusText)-lipgloss.Width(metaRight)).
				Render(""),
			metaRight,
		),
	)

	// ── Viewport (scrollable prompt content) ─────────────────
	viewportView := m.viewport.View()

	// ── Statusbar ────────────────────────────────────────────
	type keyHint struct{ key, desc string }
	renderKeys := func(hints []keyHint) string {
		result := ""
		for _, h := range hints {
			k := lipgloss.NewStyle().
				Background(lipgloss.Color("240")).
				Foreground(lipgloss.Color("255")).
				Padding(0, 1).
				Render(h.key)
			d := lipgloss.NewStyle().
				Foreground(lipgloss.Color("245")).
				Render(" " + h.desc + "  ")
			result += k + d
		}
		return result
	}

	keys := renderKeys([]keyHint{
		{"y", "send"}, {"e", "edit"}, {"q", "cancel"}, {"↑↓", "scroll"},
	})

	statusbar := lipgloss.NewStyle().
		Background(lipgloss.Color("237")).
		Width(width).
		Padding(0, 1).
		Render(keys)

	return lipgloss.NewStyle().
		Width(width).
		Height(m.height).
		Render(fmt.Sprintf("%s\n%s\n\n%s\n%s", header, meta, viewportView, statusbar))
}

// PromptEditor allows user to preview and edit prompt before sending
func PromptEditor(prompt, engine string) (string, bool, error) {
	m := newPromptEditorModel(prompt, engine)
	p := tea.NewProgram(m, tea.WithAltScreen())

	finalModel, err := p.Run()
	if err != nil {
		return "", false, fmt.Errorf("editor error: %w", err)
	}

	model := finalModel.(promptEditorModel)

	if model.cancelled || model.edited == "" {
		return "", false, nil
	}

	return model.edited, true, nil
}

// openExternalEditor opens the system's default editor to edit the prompt
func openExternalEditor(content string) (string, error) {
	tmpFile, err := os.CreateTemp("", "prompt-edit-*.txt")
	if err != nil {
		return content, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(content); err != nil {
		return content, fmt.Errorf("failed to write temp file: %w", err)
	}
	tmpFile.Close()

	editor := os.Getenv("EDITOR")
	if editor == "" {
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

	cmd := exec.Command(editor, tmpFile.Name())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return content, fmt.Errorf("editor failed: %w", err)
	}

	editedBytes, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		return content, fmt.Errorf("failed to read edited file: %w", err)
	}

	return strings.TrimSpace(string(editedBytes)), nil
}
