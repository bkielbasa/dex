package connpicker

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"

	tea "github.com/charmbracelet/bubbletea"
)

type SelectMsg struct {
	Name string
}

type CloseMsg struct{}

type Entry struct {
	Name     string
	Engine   string
	Database string
	Host     string
	Active   bool
}

type Model struct {
	entries  []Entry
	cursor   int
	width    int
	height   int
}

func New(entries []Entry) Model {
	// Pre-select the active entry
	cursor := 0
	for i, e := range entries {
		if e.Active {
			cursor = i
			break
		}
	}
	return Model{entries: entries, cursor: cursor}
}

func (m *Model) SetSize(w, h int) {
	m.width = w
	m.height = h
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return m, func() tea.Msg { return CloseMsg{} }
		case "enter":
			if len(m.entries) > 0 {
				name := m.entries[m.cursor].Name
				return m, func() tea.Msg { return SelectMsg{Name: name} }
			}
		case "j", "down":
			if m.cursor < len(m.entries)-1 {
				m.cursor++
			}
		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
			}
		case "g":
			m.cursor = 0
		case "G":
			if len(m.entries) > 0 {
				m.cursor = len(m.entries) - 1
			}
		}
	}
	return m, nil
}

func (m Model) View() string {
	if len(m.entries) == 0 {
		content := lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Render("No connections available")
		return modalStyle(m.width).Render(titleStyle() + "\n\n" + content + "\n\n" + helpLine())
	}

	var rows []string
	for i, e := range m.entries {
		marker := "  "
		if e.Active {
			marker = "* "
		}
		line := fmt.Sprintf("%s%-20s %s  %s@%s", marker, e.Name, e.Engine, e.Database, e.Host)

		style := lipgloss.NewStyle().Padding(0, 1)
		if i == m.cursor {
			style = style.
				Background(lipgloss.Color("62")).
				Foreground(lipgloss.Color("230")).
				Bold(true)
		} else if e.Active {
			style = style.Foreground(lipgloss.Color("42"))
		}
		rows = append(rows, style.Render(line))
	}

	content := lipgloss.JoinVertical(lipgloss.Left, rows...)
	return modalStyle(m.width).Render(titleStyle() + "\n\n" + content + "\n\n" + helpLine())
}

func titleStyle() string {
	return lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("230")).
		Background(lipgloss.Color("62")).
		Padding(0, 1).
		Render("Connections")
}

func helpLine() string {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Render("j/k: navigate | enter: select | esc: close")
}

func modalStyle(width int) lipgloss.Style {
	w := 60
	if width > 0 && width-6 < w {
		w = width - 6
	}
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(1, 2).
		Width(w)
}
