package cmdbar

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type ExecuteCommandMsg struct {
	Command string
}

type CancelMsg struct{}

type Model struct {
	input   textinput.Model
	focused bool
	width   int
}

func New() Model {
	ti := textinput.New()
	ti.Prompt = ":"
	ti.Placeholder = ""
	ti.CharLimit = 256
	ti.Focus()
	return Model{input: ti, focused: true}
}

func (m *Model) SetWidth(w int) {
	m.width = w
	m.input.Width = w - 4
}

func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if !m.focused {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			cmd := m.input.Value()
			m.input.SetValue("")
			m.focused = false
			return m, func() tea.Msg {
				return ExecuteCommandMsg{Command: cmd}
			}
		case "esc":
			m.input.SetValue("")
			m.focused = false
			return m, func() tea.Msg {
				return CancelMsg{}
			}
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	style := lipgloss.NewStyle().Padding(0, 1)
	return style.Render(m.input.View())
}
