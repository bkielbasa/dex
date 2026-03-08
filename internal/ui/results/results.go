// internal/ui/results/results.go
package results

import (
	"fmt"
	"strings"

	"github.com/bklimczak/dex/internal/db"
	"github.com/bklimczak/dex/internal/ui/styles"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type OpenDetailMsg struct {
	Columns []string
	Values  []string
}

type CellEditMsg struct {
	Table    string
	Column   string
	NewValue string
	IsNull   bool              // true if user wants to set NULL
	Row      map[string]string // all column values for WHERE clause
}

type SortMsg struct {
	Column string
	Desc   bool
}

type PageNextMsg struct{}
type PagePrevMsg struct{}

type Model struct {
	result      *db.QueryResult
	sourceTable string
	cursorRow   int
	cursorCol   int
	scrollX     int
	scrollY     int
	focused     bool
	editing     bool
	editInput   textinput.Model
	width       int
	height      int
	colWidths   []int
	err         string
	sortCol     int
	sortDesc    bool
	page        int
	totalRows   int
	pageSize    int
}

func New() Model {
	return Model{}
}

func (m *Model) SetSize(w, h int) {
	m.width = w
	m.height = h
}

func (m *Model) SetFocused(f bool) {
	m.focused = f
}

func (m *Model) Focused() bool {
	return m.focused
}

func (m *Model) SetSourceTable(table string) {
	m.sourceTable = table
	m.sortCol = -1
	m.sortDesc = false
	m.page = 0
	m.totalRows = 0
	m.pageSize = 0
}

func (m *Model) SourceTable() string {
	return m.sourceTable
}

func (m *Model) Result() *db.QueryResult {
	return m.result
}

func (m *Model) SetPagination(page, totalRows, pageSize int) {
	m.page = page
	m.totalRows = totalRows
	m.pageSize = pageSize
}

func (m *Model) SetResult(r *db.QueryResult) {
	m.result = r
	m.cursorRow = 0
	m.cursorCol = 0
	m.scrollX = 0
	m.scrollY = 0
	m.editing = false
	m.err = ""
	if r != nil && r.Error != "" {
		m.err = r.Error
		return
	}
	m.calculateColWidths()
}

func (m *Model) SetError(e string) {
	m.err = e
	m.result = nil
}

func (m *Model) calculateColWidths() {
	if m.result == nil {
		return
	}
	m.colWidths = make([]int, len(m.result.Columns))
	for i, col := range m.result.Columns {
		m.colWidths[i] = len(col)
	}
	for _, row := range m.result.Rows {
		for i, cell := range row {
			if len(cell) > m.colWidths[i] {
				m.colWidths[i] = len(cell)
			}
		}
	}
	for i := range m.colWidths {
		if m.colWidths[i] > 40 {
			m.colWidths[i] = 40
		}
		if m.colWidths[i] < 4 {
			m.colWidths[i] = 4
		}
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if m.editing {
		return m.updateEditing(msg)
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if !m.focused || m.result == nil {
			return m, nil
		}
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("j", "down"))):
			if m.cursorRow < len(m.result.Rows)-1 {
				m.cursorRow++
			}
			m.ensureVisible()
		case key.Matches(msg, key.NewBinding(key.WithKeys("k", "up"))):
			if m.cursorRow > 0 {
				m.cursorRow--
			}
			m.ensureVisible()
		case key.Matches(msg, key.NewBinding(key.WithKeys("h", "left"))):
			if m.cursorCol > 0 {
				m.cursorCol--
			}
			m.ensureVisibleX()
		case key.Matches(msg, key.NewBinding(key.WithKeys("l", "right"))):
			if m.cursorCol < len(m.result.Columns)-1 {
				m.cursorCol++
			}
			m.ensureVisibleX()
		case key.Matches(msg, key.NewBinding(key.WithKeys("g"))):
			m.cursorRow = 0
			m.ensureVisible()
		case key.Matches(msg, key.NewBinding(key.WithKeys("G"))):
			m.cursorRow = max(0, len(m.result.Rows)-1)
			m.ensureVisible()
		case key.Matches(msg, key.NewBinding(key.WithKeys("n"))):
			if m.sourceTable != "" && m.totalRows > 0 {
				return m, func() tea.Msg { return PageNextMsg{} }
			}
		case key.Matches(msg, key.NewBinding(key.WithKeys("p"))):
			if m.sourceTable != "" && m.totalRows > 0 {
				return m, func() tea.Msg { return PagePrevMsg{} }
			}
		case key.Matches(msg, key.NewBinding(key.WithKeys("s"))):
			if m.sourceTable != "" && m.result != nil && len(m.result.Columns) > 0 {
				if m.sortCol == m.cursorCol {
					m.sortDesc = !m.sortDesc
				} else {
					m.sortCol = m.cursorCol
					m.sortDesc = false
				}
				col := m.result.Columns[m.cursorCol]
				desc := m.sortDesc
				return m, func() tea.Msg {
					return SortMsg{Column: col, Desc: desc}
				}
			}
		case key.Matches(msg, key.NewBinding(key.WithKeys("i", "a"))):
			if m.sourceTable != "" && len(m.result.Rows) > 0 {
				m.startEditing()
				return m, m.editInput.Focus()
			}
		case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
			if len(m.result.Rows) > 0 {
				row := m.result.Rows[m.cursorRow]
				vals := make([]string, len(row))
				copy(vals, row)
				cols := make([]string, len(m.result.Columns))
				copy(cols, m.result.Columns)
				openMsg := OpenDetailMsg{Columns: cols, Values: vals}
				return m, func() tea.Msg { return openMsg }
			}
		}
	}
	return m, nil
}

func (m *Model) startEditing() {
	ti := textinput.New()
	ti.CharLimit = 1000
	currentVal := m.result.Rows[m.cursorRow][m.cursorCol]
	if currentVal == "NULL" {
		ti.SetValue("")
	} else {
		ti.SetValue(currentVal)
	}
	ti.CursorEnd()
	m.editInput = ti
	m.editing = true
}

func (m Model) updateEditing(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			newVal := m.editInput.Value()
			oldVal := m.result.Rows[m.cursorRow][m.cursorCol]
			m.editing = false
			isNull := strings.ToUpper(strings.TrimSpace(newVal)) == "NULL" || (newVal == "" && oldVal == "NULL")
			if !isNull && newVal == oldVal {
				return m, nil
			}
			// Build row map for WHERE clause
			row := make(map[string]string)
			for i, col := range m.result.Columns {
				row[col] = m.result.Rows[m.cursorRow][i]
			}
			// Update local data immediately
			if isNull {
				m.result.Rows[m.cursorRow][m.cursorCol] = "NULL"
			} else {
				m.result.Rows[m.cursorRow][m.cursorCol] = newVal
			}
			editMsg := CellEditMsg{
				Table:    m.sourceTable,
				Column:   m.result.Columns[m.cursorCol],
				NewValue: newVal,
				IsNull:   isNull,
				Row:      row,
			}
			return m, func() tea.Msg { return editMsg }
		case "esc":
			m.editing = false
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.editInput, cmd = m.editInput.Update(msg)
	return m, cmd
}

func (m *Model) ensureVisible() {
	visibleRows := m.height - 4
	if visibleRows < 1 {
		visibleRows = 1
	}
	if m.cursorRow < m.scrollY {
		m.scrollY = m.cursorRow
	}
	if m.cursorRow >= m.scrollY+visibleRows {
		m.scrollY = m.cursorRow - visibleRows + 1
	}
}

func (m *Model) ensureVisibleX() {
	if m.cursorCol < m.scrollX {
		m.scrollX = m.cursorCol
	}
	if m.cursorCol > m.scrollX+3 {
		m.scrollX = m.cursorCol - 3
	}
}

func (m Model) View() string {
	if m.err != "" {
		return styles.ErrorText.Render("Error: " + m.err)
	}
	if m.result == nil {
		return styles.NormalItem.Render("No data. Select a table or run a query.")
	}
	if len(m.result.Rows) == 0 {
		return styles.NormalItem.Render(fmt.Sprintf("Query returned 0 rows. Columns: %s",
			strings.Join(m.result.Columns, ", ")))
	}

	var b strings.Builder

	endCol := len(m.result.Columns)
	startCol := m.scrollX
	if startCol >= endCol {
		startCol = 0
	}

	// Header
	var headerParts []string
	for i := startCol; i < endCol; i++ {
		w := m.colWidths[i]
		colName := m.result.Columns[i]
		if i == m.sortCol {
			if m.sortDesc {
				colName += " ▼"
			} else {
				colName += " ▲"
			}
		}
		cell := truncPad(colName, w)
		headerParts = append(headerParts, styles.HeaderCell.Width(w+2).Render(cell))
	}
	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, headerParts...))
	b.WriteString("\n")

	// Separator
	for i := startCol; i < endCol; i++ {
		b.WriteString(strings.Repeat("─", m.colWidths[i]+2))
		if i < endCol-1 {
			b.WriteString("┼")
		}
	}
	b.WriteString("\n")

	// Rows
	visibleRows := m.height - 4
	if visibleRows < 1 {
		visibleRows = 10
	}
	endRow := min(m.scrollY+visibleRows, len(m.result.Rows))

	for r := m.scrollY; r < endRow; r++ {
		var rowParts []string
		for i := startCol; i < endCol; i++ {
			w := m.colWidths[i]
			cell := ""
			if i < len(m.result.Rows[r]) {
				cell = m.result.Rows[r][i]
			}
			cellStr := truncPad(cell, w)
			style := styles.DataCell.Width(w + 2)
			if cell == "NULL" {
				style = styles.NullCell.Width(w + 2)
			}
			if r == m.cursorRow && i == m.cursorCol && m.focused {
				if m.editing {
					style = style.Background(lipgloss.Color("22")).Foreground(lipgloss.Color("230"))
					editView := m.editInput.View()
					rowParts = append(rowParts, style.Render(truncPad(editView, w)))
					continue
				}
				style = style.Background(lipgloss.Color("237"))
			} else if r == m.cursorRow && m.focused {
				style = style.Background(lipgloss.Color("235"))
			}
			rowParts = append(rowParts, style.Render(cellStr))
		}
		b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, rowParts...))
		if r < endRow-1 {
			b.WriteString("\n")
		}
	}

	// Status line
	b.WriteString("\n")
	var status string
	if m.totalRows > 0 && m.pageSize > 0 {
		startRow := (m.page-1)*m.pageSize + 1
		endRow2 := startRow + len(m.result.Rows) - 1
		totalPages := (m.totalRows + m.pageSize - 1) / m.pageSize
		status = fmt.Sprintf(" page %d/%d | %d-%d of %d | row %d/%d ",
			m.page, totalPages, startRow, endRow2, m.totalRows,
			m.cursorRow+1, len(m.result.Rows))
	} else {
		status = fmt.Sprintf(" %d rows | row %d/%d ", m.result.RowCount, m.cursorRow+1, len(m.result.Rows))
	}
	if m.editing {
		status += "| EDITING (enter: save, esc: cancel) "
	}
	b.WriteString(styles.StatusBar.Render(status))

	return b.String()
}

func truncPad(s string, w int) string {
	if len(s) > w {
		return s[:w-1] + "…"
	}
	return s + strings.Repeat(" ", w-len(s))
}
