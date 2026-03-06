package querybar

import (
	"github.com/bklimczak/dex/internal/ui/completer"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type ExecuteQueryMsg struct {
	Query string
}

type Model struct {
	input     textinput.Model
	completer completer.Model
	focused   bool
	width     int
	history   []string
	histIdx   int
}

func New() Model {
	ti := textinput.New()
	ti.Placeholder = "SQL query..."
	ti.Prompt = ": "
	ti.CharLimit = 1000
	return Model{
		input:     ti,
		completer: completer.New(),
		histIdx:   -1,
	}
}

func (m *Model) SetWidth(w int) {
	m.width = w
	m.input.Width = w - 4
}

func (m *Model) SetFocused(f bool) {
	m.focused = f
	if f {
		m.input.Focus()
	} else {
		m.input.Blur()
		m.completer.Reset()
	}
}

func (m *Model) Focused() bool {
	return m.focused
}

func (m *Model) Completer() *completer.Model {
	return &m.completer
}

func (m *Model) AddToHistory(q string) {
	if q == "" {
		return
	}
	if len(m.history) > 0 && m.history[len(m.history)-1] == q {
		return
	}
	m.history = append(m.history, q)
	m.histIdx = len(m.history)
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if !m.focused {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			m.completer.Reset()
			query := m.input.Value()
			if query != "" {
				m.AddToHistory(query)
				m.input.SetValue("")
				m.histIdx = len(m.history)
				return m, func() tea.Msg {
					return ExecuteQueryMsg{Query: query}
				}
			}
		case "ctrl+n":
			if m.completer.Active() {
				m.completer.Next()
				return m, nil
			}
		case "ctrl+p":
			if m.completer.Active() {
				m.completer.Prev()
				return m, nil
			}
		case "up":
			m.completer.Reset()
			if len(m.history) > 0 && m.histIdx > 0 {
				m.histIdx--
				m.input.SetValue(m.history[m.histIdx])
				m.input.CursorEnd()
			}
			return m, nil
		case "down":
			m.completer.Reset()
			if m.histIdx < len(m.history)-1 {
				m.histIdx++
				m.input.SetValue(m.history[m.histIdx])
				m.input.CursorEnd()
			} else {
				m.histIdx = len(m.history)
				m.input.SetValue("")
			}
			return m, nil
		case "esc":
			if m.completer.Active() {
				m.completer.Reset()
				return m, nil
			}
			m.input.SetValue("")
			return m, nil
		case "tab":
			if m.completer.Active() {
				if full, ok := m.completer.Accept(); ok {
					val := m.input.Value()
					pos := m.input.Position()
					prefix := extractWordBackward(val, pos)
					before := val[:pos-len(prefix)]
					after := val[pos:]
					m.input.SetValue(before + full + after)
					m.input.SetCursor(len(before) + len(full))
					m.completer.Reset()
				}
				return m, nil
			}
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)

	// Update completer after each keystroke
	m.completer.Update(m.input.Value(), m.input.Position())

	return m, cmd
}

func (m Model) CompleterView() string {
	if m.completer.Active() {
		return m.completer.View()
	}
	return ""
}

func (m Model) View() string {
	style := lipgloss.NewStyle().
		Padding(0, 1)
	return style.Render(m.input.View())
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
