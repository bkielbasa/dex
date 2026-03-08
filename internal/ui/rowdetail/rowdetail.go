// internal/ui/rowdetail/rowdetail.go
package rowdetail

import (
	"fmt"
	"strings"

	"github.com/bklimczak/dex/internal/ui/styles"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type CloseMsg struct{}

type Model struct {
	columns  []string
	values   []string
	viewport viewport.Model
	width    int
	height   int
}

var (
	columnNameStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("62"))

	nullValueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Italic(true)
)

func New(columns, values []string, width, height int) Model {
	vp := viewport.New(width-8, height-8)
	vp.SetContent(renderDetail(columns, values))
	return Model{
		columns:  columns,
		values:   values,
		viewport: vp,
		width:    width,
		height:   height,
	}
}

func renderDetail(columns, values []string) string {
	var b strings.Builder

	b.WriteString(styles.ModalTitle.Render("Row Detail"))
	b.WriteString("\n\n")

	// Find max column name width for alignment
	maxColWidth := 0
	for _, col := range columns {
		if len(col) > maxColWidth {
			maxColWidth = len(col)
		}
	}

	for i, col := range columns {
		val := ""
		if i < len(values) {
			val = values[i]
		}

		name := columnNameStyle.Render(fmt.Sprintf("%-*s", maxColWidth, col))

		var valRendered string
		if val == "NULL" {
			valRendered = nullValueStyle.Render("NULL")
		} else {
			valRendered = val
		}

		b.WriteString(fmt.Sprintf("  %s  %s\n", name, valRendered))
	}

	return b.String()
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q", "enter":
			return m, func() tea.Msg { return CloseMsg{} }
		}
	}
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	help := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).
		Render("j/k: scroll | Esc/Enter/q: close")

	content := m.viewport.View() + "\n" + help

	w := 80
	if m.width > 0 && m.width-6 < w {
		w = m.width - 6
	}

	return styles.ModalOverlay.
		Width(w).
		Render(content)
}
