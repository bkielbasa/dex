package completer

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/charmbracelet/lipgloss"
)

type Model struct {
	tables     []string
	columns    map[string][]string // table -> columns
	matches    []string
	selected   int
	active     bool
	prefix     string
}

func New() Model {
	return Model{
		columns: make(map[string][]string),
	}
}

func (m *Model) SetTables(tables []string) {
	m.tables = tables
}

func (m *Model) SetColumns(table string, cols []string) {
	m.columns[table] = cols
}

func (m *Model) CopyFrom(other *Model) {
	m.tables = make([]string, len(other.tables))
	copy(m.tables, other.tables)
	m.columns = make(map[string][]string)
	for k, v := range other.columns {
		cols := make([]string, len(v))
		copy(cols, v)
		m.columns[k] = cols
	}
}

func (m *Model) AllCompletions() []string {
	seen := make(map[string]bool)
	var all []string
	for _, t := range m.tables {
		if !seen[t] {
			all = append(all, t)
			seen[t] = true
		}
	}
	for _, cols := range m.columns {
		for _, c := range cols {
			if !seen[c] {
				all = append(all, c)
				seen[c] = true
			}
		}
	}
	return all
}

// Update recalculates matches based on the current input text and cursor position.
// Returns true if there are matches to show.
func (m *Model) Update(text string, cursorPos int) bool {
	m.prefix = extractWord(text, cursorPos)
	if len(m.prefix) < 1 {
		m.active = false
		m.matches = nil
		m.selected = 0
		return false
	}

	lower := strings.ToLower(m.prefix)
	m.matches = nil
	for _, item := range m.AllCompletions() {
		if strings.HasPrefix(strings.ToLower(item), lower) && strings.ToLower(item) != lower {
			m.matches = append(m.matches, item)
		}
	}

	m.active = len(m.matches) > 0
	if m.selected >= len(m.matches) {
		m.selected = 0
	}
	return m.active
}

// Next moves selection to the next match.
func (m *Model) Next() {
	if !m.active || len(m.matches) == 0 {
		return
	}
	m.selected = (m.selected + 1) % len(m.matches)
}

// Prev moves selection to the previous match.
func (m *Model) Prev() {
	if !m.active || len(m.matches) == 0 {
		return
	}
	m.selected = (m.selected - 1 + len(m.matches)) % len(m.matches)
}

// Accept returns the currently selected completion.
func (m *Model) Accept() (completion string, ok bool) {
	if !m.active || len(m.matches) == 0 {
		return "", false
	}
	return m.matches[m.selected], true
}

func (m *Model) Active() bool {
	return m.active
}

func (m *Model) Reset() {
	m.active = false
	m.matches = nil
	m.selected = 0
}

const maxVisible = 10

func (m Model) View() string {
	if !m.active || len(m.matches) == 0 {
		return ""
	}

	// Scrollable window around selected item
	start := 0
	if m.selected >= maxVisible {
		start = m.selected - maxVisible + 1
	}
	end := start + maxVisible
	if end > len(m.matches) {
		end = len(m.matches)
	}

	var lines []string
	for i := start; i < end; i++ {
		style := lipgloss.NewStyle().
			Padding(0, 1).
			Background(lipgloss.Color("237")).
			Foreground(lipgloss.Color("252"))
		if i == m.selected {
			style = style.
				Background(lipgloss.Color("62")).
				Foreground(lipgloss.Color("230")).
				Bold(true)
		}
		lines = append(lines, style.Render(m.matches[i]))
	}

	header := fmt.Sprintf(" %d/%d ", m.selected+1, len(m.matches))
	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Italic(true)

	border := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240"))

	return border.Render(headerStyle.Render(header) + "\n" + strings.Join(lines, "\n"))
}

func extractWord(text string, pos int) string {
	if pos <= 0 || pos > len(text) {
		return ""
	}
	// Walk backwards from cursor to find word start
	start := pos
	for start > 0 {
		r := rune(text[start-1])
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			start--
		} else {
			break
		}
	}
	return text[start:pos]
}
