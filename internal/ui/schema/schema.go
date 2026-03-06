// internal/ui/schema/schema.go
package schema

import (
	"fmt"
	"strings"

	"github.com/bklimczak/dex/internal/db"
	"github.com/bklimczak/dex/internal/ui/styles"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type CloseMsg struct{}

type Model struct {
	schema   *db.TableSchema
	viewport viewport.Model
	width    int
	height   int
}

func New(s *db.TableSchema, w, h int) Model {
	vp := viewport.New(w-8, h-8)
	vp.SetContent(renderSchema(s))
	return Model{
		schema:   s,
		viewport: vp,
		width:    w,
		height:   h,
	}
}

func renderSchema(s *db.TableSchema) string {
	var b strings.Builder

	b.WriteString(styles.ModalTitle.Render(fmt.Sprintf("Table: %s", s.Name)))
	b.WriteString("\n\n")

	b.WriteString(lipgloss.NewStyle().Bold(true).Render("Columns"))
	b.WriteString("\n")
	for _, col := range s.Columns {
		nullable := ""
		if col.Nullable {
			nullable = " (nullable)"
		}
		b.WriteString(fmt.Sprintf("  %-30s %-20s%s\n", col.Name, col.Type, nullable))
	}

	if len(s.Indexes) > 0 {
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().Bold(true).Render("Indexes"))
		b.WriteString("\n")
		for _, idx := range s.Indexes {
			unique := ""
			if idx.Unique {
				unique = " UNIQUE"
			}
			b.WriteString(fmt.Sprintf("  %-30s (%s)%s\n", idx.Name, strings.Join(idx.Columns, ", "), unique))
		}
	}

	if len(s.ForeignKeys) > 0 {
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().Bold(true).Render("Foreign Keys"))
		b.WriteString("\n")
		for _, fk := range s.ForeignKeys {
			b.WriteString(fmt.Sprintf("  %s -> %s.%s (%s)\n", fk.Column, fk.RefTable, fk.RefColumn, fk.ConstraintName))
		}
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
		case "esc", "q":
			return m, func() tea.Msg { return CloseMsg{} }
		}
	}
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	help := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).
		Render("j/k: scroll | Esc: close")

	content := m.viewport.View() + "\n" + help

	w := 80
	if m.width > 0 && m.width-6 < w {
		w = m.width - 6
	}

	return styles.ModalOverlay.
		Width(w).
		Render(content)
}
