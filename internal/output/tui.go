package output

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/meruem/prompt-builder-agent/internal/llm"
)

// TUIRenderer renders streaming output using Bubbletea TUI with markdown support
type TUIRenderer struct {
	engine string
}

// NewTUIRenderer creates a new TUI renderer
func NewTUIRenderer(engine string) *TUIRenderer {
	return &TUIRenderer{engine: engine}
}

// RenderStream implements StreamRenderer interface
func (r *TUIRenderer) RenderStream(ch <-chan llm.Chunk) error {
	m := newModel(r.engine)
	p := tea.NewProgram(m, tea.WithAltScreen())

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

	_, err := p.Run()

	// Drain any remaining items so the upstream tee goroutine is never
	// blocked trying to send to ch after the TUI has exited.
	for range ch {
	}

	if err != nil {
		return fmt.Errorf("TUI render error: %w", err)
	}
	return nil
}

// Messages
type chunkMsg string
type doneMsg struct{}
type errorMsg error
type tickMsg time.Time
type clearCopyMsg struct{}

// Model
type tuiModel struct {
	viewport   viewport.Model
	content    string
	engine     string
	done       bool
	err        error
	copyStatus string
	loading    bool
	spinner    int
	width      int
	height     int
}

func newModel(engine string) tuiModel {
	vp := viewport.New(80, 20)
	vp.Style = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("238"))

	return tuiModel{
		viewport: vp,
		engine:   engine,
	}
}

func (m tuiModel) Init() tea.Cmd {
	return tea.Tick(time.Millisecond*100, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "c":
			if m.content != "" {
				if err := copyToClipboard(m.content); err != nil {
					m.copyStatus = fmt.Sprintf("✗ Copy failed: %v", err)
				} else {
					m.copyStatus = "✓ Copied!"
				}
				// auto-clear after 2s
				return m, tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
					return clearCopyMsg{}
				})
			}
			return m, nil
		case "up", "k":
			m.viewport.LineUp(1)
		case "down", "j":
			m.viewport.LineDown(1)
		case "pgup":
			m.viewport.ViewUp()
		case "pgdown", " ":
			m.viewport.ViewDown()
		case "home", "g":
			m.viewport.GotoTop()
		case "end", "G":
			m.viewport.GotoBottom()
		default:
			m.viewport, cmd = m.viewport.Update(msg)
		}
		return m, cmd

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		headerH := 3 // header + separator
		footerH := 2 // statusbar
		m.viewport.Width = msg.Width - 2
		m.viewport.Height = msg.Height - headerH - footerH
		return m, nil

	case tickMsg:
		if m.content == "" && !m.done && m.err == nil {
			m.loading = true
			m.spinner = (m.spinner + 1) % 10
			return m, tea.Tick(time.Millisecond*100, func(t time.Time) tea.Msg {
				return tickMsg(t)
			})
		}
		return m, nil

	case clearCopyMsg:
		m.copyStatus = ""
		return m, nil

	case chunkMsg:
		m.content += string(msg)
		m.loading = false
		m.viewport.SetContent(renderMarkdown(m.content, m.viewport.Width))
		m.viewport.GotoBottom()
		return m, nil

	case doneMsg:
		m.done = true
		m.loading = false
		return m, nil

	case errorMsg:
		m.err = error(msg)
		m.loading = false
		m.viewport.SetContent(lipgloss.NewStyle().
			Foreground(lipgloss.Color("1")).
			Render(fmt.Sprintf("Error: %v", m.err)))
		return m, nil
	}

	return m, cmd
}

func (m tuiModel) View() string {
	// ── Header ──────────────────────────────────────────────
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("39")).
		Render("AI Response")

	var statusBadge string
	switch {
	case m.err != nil:
		statusBadge = lipgloss.NewStyle().
			Background(lipgloss.Color("52")).
			Foreground(lipgloss.Color("9")).
			Padding(0, 1).
			Render("✗ Error")
	case m.done:
		statusBadge = lipgloss.NewStyle().
			Background(lipgloss.Color("22")).
			Foreground(lipgloss.Color("10")).
			Padding(0, 1).
			Render("✓ Done")
	case m.loading && m.content == "":
		spinners := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		statusBadge = lipgloss.NewStyle().
			Background(lipgloss.Color("58")).
			Foreground(lipgloss.Color("11")).
			Padding(0, 1).
			Render(spinners[m.spinner] + " Waiting...")
	default:
		statusBadge = lipgloss.NewStyle().
			Background(lipgloss.Color("58")).
			Foreground(lipgloss.Color("11")).
			Padding(0, 1).
			Render("⏳ Streaming...")
	}

	var engineBadge string
	if m.engine != "" {
		engineBadge = lipgloss.NewStyle().
			Background(lipgloss.Color("55")).
			Foreground(lipgloss.Color("183")).
			Padding(0, 1).
			Render(m.engine)
	}

	headerRight := lipgloss.JoinHorizontal(lipgloss.Center, engineBadge, " ", statusBadge)
	headerWidth := m.width
	if headerWidth == 0 {
		headerWidth = 80
	}
	header := lipgloss.NewStyle().
		Width(headerWidth).
		Render(
			lipgloss.JoinHorizontal(lipgloss.Left,
				title,
				lipgloss.NewStyle().
					Width(headerWidth-lipgloss.Width(title)-lipgloss.Width(headerRight)).
					Render(""),
				headerRight,
			),
		)

	separator := lipgloss.NewStyle().
		Foreground(lipgloss.Color("238")).
		Render(repeatChar("─", headerWidth))

	// ── Viewport ─────────────────────────────────────────────
	var viewportView string
	if m.content == "" && m.loading {
		spinners := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		loadingText := lipgloss.NewStyle().
			Foreground(lipgloss.Color("11")).
			Bold(true).
			Render(fmt.Sprintf("  %s  Waiting for %s...", spinners[m.spinner], engineLabel(m.engine)))
		m.viewport.SetContent("\n\n" + loadingText + "\n\n" +
			lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Italic(true).
				Render("  This may take a few moments"))
	}
	viewportView = m.viewport.View()

	// ── Statusbar ────────────────────────────────────────────
	var leftStatus string
	if m.copyStatus != "" {
		isCopyOk := strings.HasPrefix(m.copyStatus, "✓")
		fg := lipgloss.Color("10")
		if !isCopyOk {
			fg = lipgloss.Color("9")
		}
		leftStatus = lipgloss.NewStyle().Foreground(fg).Render(m.copyStatus)
	}

	var keys string
	if m.content != "" {
		keys = renderKeys([]keyHint{
			{"↑↓", "scroll"}, {"c", "copy"}, {"q", "quit"},
		})
	} else {
		keys = renderKeys([]keyHint{{"ctrl+c", "exit"}})
	}

	sbWidth := headerWidth
	statusbar := lipgloss.NewStyle().
		Background(lipgloss.Color("237")).
		Width(sbWidth).
		Render(
			lipgloss.JoinHorizontal(lipgloss.Left,
				lipgloss.NewStyle().Background(lipgloss.Color("237")).Padding(0, 1).Render(leftStatus),
				lipgloss.NewStyle().
					Background(lipgloss.Color("237")).
					Width(sbWidth-lipgloss.Width(leftStatus)-2-lipgloss.Width(keys)-2).
					Render(""),
				lipgloss.NewStyle().Background(lipgloss.Color("237")).Padding(0, 1).Render(keys),
			),
		)

	return fmt.Sprintf("%s\n%s\n%s\n%s", header, separator, viewportView, statusbar)
}

// ── Helpers ──────────────────────────────────────────────────────────────────

type keyHint struct{ key, desc string }

func renderKeys(hints []keyHint) string {
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

func repeatChar(ch string, n int) string {
	result := ""
	for i := 0; i < n; i++ {
		result += ch
	}
	return result
}

func engineLabel(engine string) string {
	if engine == "" {
		return "AI"
	}
	return engine
}

func renderMarkdown(content string, width int) string {
	if width <= 0 {
		width = 80
	}
	r, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return content
	}
	out, err := r.Render(content)
	if err != nil {
		return content
	}
	return out
}
