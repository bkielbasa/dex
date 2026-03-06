// internal/ui/editor/editor.go
package editor

import (
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
	textarea textarea.Model
	width    int
	height   int
	history  []string
	histIdx  int
}

func New() Model {
	ta := textarea.New()
	ta.Placeholder = "Write your SQL query here..."
	ta.ShowLineNumbers = true
	ta.Focus()
	return Model{
		textarea: ta,
		histIdx:  -1,
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

func (m Model) Init() tea.Cmd {
	return textarea.Blink
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return m, func() tea.Msg { return CloseMsg{} }
		case "ctrl+enter":
			query := m.textarea.Value()
			if query != "" {
				return m, func() tea.Msg { return ExecuteMsg{Query: query} }
			}
			return m, nil
		case "ctrl+p":
			if len(m.history) > 0 && m.histIdx > 0 {
				m.histIdx--
				m.textarea.SetValue(m.history[m.histIdx])
			}
			return m, nil
		case "ctrl+n":
			if m.histIdx < len(m.history)-1 {
				m.histIdx++
				m.textarea.SetValue(m.history[m.histIdx])
			} else {
				m.histIdx = len(m.history)
				m.textarea.SetValue("")
			}
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.textarea, cmd = m.textarea.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	title := styles.ModalTitle.Render("SQL Editor")
	help := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).
		Render("Ctrl+Enter: execute | Ctrl+p/n: history | Esc: close")

	content := title + "\n\n" + m.textarea.View() + "\n\n" + help

	w := 100
	if m.width > 0 && m.width-6 < w {
		w = m.width - 6
	}

	return styles.ModalOverlay.
		Width(w).
		Render(content)
}
