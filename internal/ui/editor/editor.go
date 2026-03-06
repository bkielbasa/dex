package editor

import (
	"github.com/bklimczak/dex/internal/ui/completer"
	"github.com/bklimczak/dex/internal/ui/styles"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type ExecuteMsg struct {
	Query string
}

type CloseMsg struct{}

type Model struct {
	textarea  textarea.Model
	completer completer.Model
	width     int
	height    int
	history   []string
	histIdx   int
}

func New() Model {
	ta := textarea.New()
	ta.Placeholder = "Write your SQL query here..."
	ta.ShowLineNumbers = true
	ta.Focus()
	return Model{
		textarea:  ta,
		completer: completer.New(),
		histIdx:   -1,
	}
}

func (m *Model) SetSize(w, h int) {
	m.width = w
	m.height = h
	m.textarea.SetWidth(w - 8)
	m.textarea.SetHeight(h - 8)
}

func (m *Model) SetHistory(h []string) {
	m.history = h
	m.histIdx = len(h)
}

func (m *Model) Completer() *completer.Model {
	return &m.completer
}

func (m Model) Init() tea.Cmd {
	return textarea.Blink
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			if m.completer.Active() {
				m.completer.Reset()
				return m, nil
			}
			return m, func() tea.Msg { return CloseMsg{} }
		case "ctrl+enter":
			m.completer.Reset()
			query := m.textarea.Value()
			if query != "" {
				return m, func() tea.Msg { return ExecuteMsg{Query: query} }
			}
			return m, nil
		case "ctrl+n":
			if m.completer.Active() {
				m.completer.Next()
				return m, nil
			}
			// History navigation
			if m.histIdx < len(m.history)-1 {
				m.histIdx++
				m.textarea.SetValue(m.history[m.histIdx])
			} else {
				m.histIdx = len(m.history)
				m.textarea.SetValue("")
			}
			return m, nil
		case "ctrl+p":
			if m.completer.Active() {
				m.completer.Prev()
				return m, nil
			}
			if len(m.history) > 0 && m.histIdx > 0 {
				m.histIdx--
				m.textarea.SetValue(m.history[m.histIdx])
			}
			return m, nil
		case "tab":
			if m.completer.Active() {
				if full, ok := m.completer.Accept(); ok {
					val := m.textarea.Value()
					line, col := m.textarea.Line(), m.textarea.LineInfo().ColumnOffset
					pos := findPos(val, line, col)
					prefix := extractWordBackward(val, pos)
					before := val[:pos-len(prefix)]
					after := val[pos:]
					m.textarea.SetValue(before + full + after)
					m.completer.Reset()
				}
				return m, nil
			}
		}
	}

	var cmd tea.Cmd
	m.textarea, cmd = m.textarea.Update(msg)

	// Update completer
	val := m.textarea.Value()
	line, col := m.textarea.Line(), m.textarea.LineInfo().ColumnOffset
	pos := findPos(val, line, col)
	m.completer.Update(val, pos)

	return m, cmd
}

func (m Model) View() string {
	title := styles.ModalTitle.Render("SQL Editor")
	helpText := "Ctrl+Enter: execute | Ctrl+n: complete | Esc: close"
	help := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(helpText)

	content := title + "\n\n" + m.textarea.View()
	if m.completer.Active() {
		content += "\n" + m.completer.View()
	}
	content += "\n\n" + help

	w := 100
	if m.width > 0 && m.width-6 < w {
		w = m.width - 6
	}

	return styles.ModalOverlay.
		Width(w).
		Render(content)
}

func findPos(text string, line, col int) int {
	pos := 0
	currentLine := 0
	for i, ch := range text {
		if currentLine == line {
			if pos-lineStart(text, i) >= col {
				return i
			}
		}
		if ch == '\n' {
			currentLine++
		}
		pos = i + 1
	}
	return len(text)
}

func lineStart(text string, pos int) int {
	for i := pos - 1; i >= 0; i-- {
		if text[i] == '\n' {
			return i + 1
		}
	}
	return 0
}

func extractWordBackward(text string, pos int) string {
	if pos <= 0 || pos > len(text) {
		return ""
	}
	start := pos
	for start > 0 {
		r := text[start-1]
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
			start--
		} else {
			break
		}
	}
	return text[start:pos]
}
