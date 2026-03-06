package querypicker

import (
	"strings"

	"github.com/bklimczak/dex/internal/config"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type SelectMsg struct {
	Query string
}

type CloseMsg struct{}

type DeleteMsg struct {
	Name string
}

type Model struct {
	queries  []config.SavedQuery
	filtered []config.SavedQuery
	input    textinput.Model
	cursor   int
	width    int
	height   int
}

func New(queries []config.SavedQuery) Model {
	ti := textinput.New()
	ti.Prompt = "/ "
	ti.Placeholder = "search queries..."
	ti.Focus()
	m := Model{
		queries: queries,
		input:   ti,
	}
	m.filtered = queries
	return m
}

func (m *Model) SetSize(w, h int) {
	m.width = w
	m.height = h
}

func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return m, func() tea.Msg { return CloseMsg{} }
		case "enter":
			if len(m.filtered) > 0 && m.cursor < len(m.filtered) {
				q := m.filtered[m.cursor].Query
				return m, func() tea.Msg { return SelectMsg{Query: q} }
			}
		case "ctrl+d":
			if len(m.filtered) > 0 && m.cursor < len(m.filtered) {
				name := m.filtered[m.cursor].Name
				return m, func() tea.Msg { return DeleteMsg{Name: name} }
			}
		case "ctrl+n", "down":
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
			}
			return m, nil
		case "ctrl+p", "up":
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	m.applyFilter()
	return m, cmd
}

func (m *Model) applyFilter() {
	query := strings.ToLower(m.input.Value())
	if query == "" {
		m.filtered = m.queries
	} else {
		m.filtered = nil
		for _, sq := range m.queries {
			if fuzzyMatch(strings.ToLower(sq.Name), query) {
				m.filtered = append(m.filtered, sq)
			}
		}
	}
	if m.cursor >= len(m.filtered) {
		m.cursor = 0
	}
}

// fuzzyMatch checks if all characters of pattern appear in s in order.
func fuzzyMatch(s, pattern string) bool {
	pi := 0
	for i := 0; i < len(s) && pi < len(pattern); i++ {
		if s[i] == pattern[pi] {
			pi++
		}
	}
	return pi == len(pattern)
}

const maxVisible = 12

func (m Model) View() string {
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("230")).
		Background(lipgloss.Color("62")).
		Padding(0, 1).
		Render("Saved Queries")

	searchLine := m.input.View()

	if len(m.filtered) == 0 {
		empty := lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Render("No matching queries")
		content := title + "\n\n" + searchLine + "\n\n" + empty + "\n\n" + helpLine()
		return modalStyle(m.width).Render(content)
	}

	start := 0
	if m.cursor >= maxVisible {
		start = m.cursor - maxVisible + 1
	}
	end := start + maxVisible
	if end > len(m.filtered) {
		end = len(m.filtered)
	}

	nameStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("75"))
	queryStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	selectedNameStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("230"))
	selectedQueryStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("223"))

	var rows []string
	for i := start; i < end; i++ {
		sq := m.filtered[i]
		preview := sq.Query
		if len(preview) > 60 {
			preview = preview[:57] + "..."
		}
		preview = strings.ReplaceAll(preview, "\n", " ")

		rowStyle := lipgloss.NewStyle().Padding(0, 1)
		var line string
		if i == m.cursor {
			rowStyle = rowStyle.
				Background(lipgloss.Color("62"))
			line = selectedNameStyle.Render(sq.Name) + "  " + selectedQueryStyle.Render(preview)
		} else {
			line = nameStyle.Render(sq.Name) + "  " + queryStyle.Render(preview)
		}
		rows = append(rows, rowStyle.Render(line))
	}

	list := lipgloss.JoinVertical(lipgloss.Left, rows...)
	content := title + "\n\n" + searchLine + "\n\n" + list + "\n\n" + helpLine()
	return modalStyle(m.width).Render(content)
}

func helpLine() string {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Render("↑/↓: navigate | enter: execute | ctrl+d: delete | esc: close")
}

func modalStyle(width int) lipgloss.Style {
	w := 80
	if width > 0 && width-6 < w {
		w = width - 6
	}
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(1, 2).
		Width(w)
}
