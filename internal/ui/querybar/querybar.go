package querybar

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type ExecuteQueryMsg struct {
	Query string
}

type Model struct {
	input   textinput.Model
	focused bool
	width   int
	history []string
	histIdx int
}

func New() Model {
	ti := textinput.New()
	ti.Placeholder = "SQL query..."
	ti.Prompt = ": "
	ti.CharLimit = 1000
	return Model{
		input:   ti,
		histIdx: -1,
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
	}
}

func (m *Model) Focused() bool {
	return m.focused
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
			query := m.input.Value()
			if query != "" {
				m.AddToHistory(query)
				m.input.SetValue("")
				m.histIdx = len(m.history)
				return m, func() tea.Msg {
					return ExecuteQueryMsg{Query: query}
				}
			}
		case "ctrl+p":
			if len(m.history) > 0 && m.histIdx > 0 {
				m.histIdx--
				m.input.SetValue(m.history[m.histIdx])
				m.input.CursorEnd()
			}
			return m, nil
		case "ctrl+n":
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
			m.input.SetValue("")
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	style := lipgloss.NewStyle().
		Padding(0, 1)
	return style.Render(m.input.View())
}
