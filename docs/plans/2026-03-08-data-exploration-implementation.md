# Data Exploration Features Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add five data exploration features: row detail view, column sorting, export, server-side pagination, and column hiding.

**Architecture:** All features modify the results grid component (`internal/ui/results/results.go`) and its integration in `internal/app/app.go`. Pagination and sorting require new messages and async commands in app.go. Export is handled via the command bar. A new `rowdetail` modal component is created for the detail view.

**Tech Stack:** Go, bubbletea, lipgloss, encoding/csv, encoding/json

---

### Task 1: Change edit keybinding from Enter to i/a

**Files:**
- Modify: `internal/ui/results/results.go:147-151` (change enter to i/a)
- Modify: `internal/ui/results/results.go:316-317` (update help text)

**Step 1: Change the key binding in Update**

In `internal/ui/results/results.go`, replace the `enter` case (line 147) with `i` and `a`:

```go
case key.Matches(msg, key.NewBinding(key.WithKeys("i", "a"))):
    if m.sourceTable != "" && len(m.result.Rows) > 0 {
        m.startEditing()
        return m, m.editInput.Focus()
    }
```

**Step 2: Update the editing help text**

In the status line (line 316-317), change the editing hint:

```go
if m.editing {
    status += "| EDITING (enter: save, esc: cancel) "
}
```

This stays the same — editing behavior unchanged, only the trigger key changes.

**Step 3: Build and verify**

Run: `go build -o dex .`
Expected: Builds without errors.

**Step 4: Commit**

```bash
git add internal/ui/results/results.go
git commit -m "refactor: change edit trigger from Enter to i/a (vim-compatible)"
```

---

### Task 2: Row Detail View Modal

**Files:**
- Create: `internal/ui/rowdetail/rowdetail.go`
- Modify: `internal/app/app.go` (add modal enum, message handlers, view overlay)
- Modify: `internal/ui/results/results.go` (emit OpenDetailMsg on Enter)

**Step 1: Create the rowdetail component**

Create `internal/ui/rowdetail/rowdetail.go`:

```go
package rowdetail

import (
	"fmt"
	"strings"

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

func New(columns, values []string, width, height int) Model {
	vp := viewport.New(width-8, height-8)

	nameStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("75")).Width(30)
	valStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	nullStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Italic(true)

	var lines []string
	for i, col := range columns {
		val := ""
		if i < len(values) {
			val = values[i]
		}
		vs := valStyle
		if val == "NULL" {
			vs = nullStyle
		}
		lines = append(lines, nameStyle.Render(col)+"  "+vs.Render(val))
	}
	vp.SetContent(strings.Join(lines, "\n"))

	return Model{
		columns:  columns,
		values:   values,
		viewport: vp,
		width:    width,
		height:   height,
	}
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "enter", "q":
			return m, func() tea.Msg { return CloseMsg{} }
		}
	}
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("230")).
		Background(lipgloss.Color("62")).
		Padding(0, 1).
		Render("Row Detail")

	help := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Render(fmt.Sprintf("j/k: scroll | esc: close | row has %d columns", len(m.columns)))

	w := 80
	if m.width > 0 && m.width-6 < w {
		w = m.width - 6
	}

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(1, 2).
		Width(w).
		Render(title + "\n\n" + m.viewport.View() + "\n\n" + help)
}
```

**Step 2: Add OpenDetailMsg to results**

In `internal/ui/results/results.go`, add a new message type after `CellEditMsg`:

```go
type OpenDetailMsg struct {
	Columns []string
	Values  []string
}
```

Replace the `enter` key case (previously removed in Task 1, now re-add as detail view):

```go
case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
    if m.result != nil && len(m.result.Rows) > 0 {
        return m, func() tea.Msg {
            return OpenDetailMsg{
                Columns: m.result.Columns,
                Values:  m.result.Rows[m.cursorRow],
            }
        }
    }
```

**Step 3: Wire into app.go**

In `internal/app/app.go`:

1. Add import: `"github.com/bklimczak/dex/internal/ui/rowdetail"`
2. Add `modalRowDetail` to modal enum
3. Add `rowDetail rowdetail.Model` field to Model struct
4. Add message handlers:

```go
case results.OpenDetailMsg:
    m.rowDetail = rowdetail.New(msg.Columns, msg.Values, m.width, m.height)
    m.modal = modalRowDetail
    return m, nil

case rowdetail.CloseMsg:
    m.modal = modalNone
    return m, nil
```

5. Add modal delegation in Update switch:

```go
case modalRowDetail:
    var cmd tea.Cmd
    m.rowDetail, cmd = m.rowDetail.Update(msg)
    if cmd != nil {
        cmds = append(cmds, cmd)
    }
    return m, tea.Batch(cmds...)
```

6. Add view overlay:

```go
case modalRowDetail:
    return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center,
        m.rowDetail.View(), lipgloss.WithWhitespaceChars(" "), lipgloss.WithWhitespaceForeground(lipgloss.Color("236")))
```

**Step 4: Build and verify**

Run: `go build -o dex .`

**Step 5: Commit**

```bash
git add internal/ui/rowdetail/rowdetail.go internal/ui/results/results.go internal/app/app.go
git commit -m "feat: row detail view modal on Enter"
```

---

### Task 3: Column Sorting

**Files:**
- Modify: `internal/ui/results/results.go` (add sort state, `s` key, sort indicator in header)
- Modify: `internal/app/app.go` (handle SortMsg, rebuild query with ORDER BY)

**Step 1: Add sort state and message to results**

In `internal/ui/results/results.go`, add fields to Model:

```go
type SortMsg struct {
	Column string
	Desc   bool
}

// In Model struct, add:
sortCol  int  // -1 = no sort
sortDesc bool
```

Initialize `sortCol: -1` in `SetResult`.

**Step 2: Add `s` key handler**

In the Update key switch, add:

```go
case key.Matches(msg, key.NewBinding(key.WithKeys("s"))):
    if m.result != nil && len(m.result.Columns) > 0 {
        if m.sortCol == m.cursorCol {
            m.sortDesc = !m.sortDesc
        } else {
            m.sortCol = m.cursorCol
            m.sortDesc = false
        }
        col := m.result.Columns[m.cursorCol]
        return m, func() tea.Msg {
            return SortMsg{Column: col, Desc: m.sortDesc}
        }
    }
```

**Step 3: Add sort indicator in header**

In the View header rendering loop, append arrow to column name:

```go
colName := m.result.Columns[i]
if i == m.sortCol {
    if m.sortDesc {
        colName += " ▼"
    } else {
        colName += " ▲"
    }
}
cell := truncPad(colName, w)
```

**Step 4: Handle SortMsg in app.go**

In `internal/app/app.go`, add handler:

```go
case results.SortMsg:
    if m.results.SourceTable() != "" {
        order := "ASC"
        if msg.Desc {
            order = "DESC"
        }
        query := fmt.Sprintf("SELECT * FROM %s ORDER BY %s %s LIMIT 100",
            m.results.SourceTable(), msg.Column, order)
        m.status = fmt.Sprintf("Sorting by %s %s...", msg.Column, order)
        return m, m.executeQueryCmd(query)
    }
```

Add a `SourceTable()` accessor to results model:

```go
func (m *Model) SourceTable() string {
    return m.sourceTable
}
```

Note: `SetResult` must preserve `sortCol` and `sortDesc` when results come back from a sort query. Add a `keepSort` flag or just don't reset them in `SetResult`. Simplest: only reset sort state in `SetSourceTable`.

**Step 5: Build and verify**

Run: `go build -o dex .`

**Step 6: Commit**

```bash
git add internal/ui/results/results.go internal/app/app.go
git commit -m "feat: column sorting with s key (server-side ORDER BY)"
```

---

### Task 4: Export Results

**Files:**
- Modify: `internal/app/app.go` (add export command handler)

**Step 1: Add export to handleCommand**

In `internal/app/app.go`, add cases in `handleCommand`:

```go
case "export":
    return m.exportResults(arg)
```

**Step 2: Implement exportResults method**

```go
func (m Model) exportResults(arg string) (tea.Model, tea.Cmd) {
    if m.results.Result() == nil || len(m.results.Result().Rows) == 0 {
        m.status = "No results to export"
        return m, nil
    }

    // Parse format and path: "csv [path]" or "json [path]"
    parts := strings.SplitN(arg, " ", 2)
    format := strings.ToLower(parts[0])
    if format != "csv" && format != "json" {
        m.status = "Usage: :export csv [path] | :export json [path]"
        return m, nil
    }

    path := ""
    if len(parts) > 1 {
        path = strings.TrimSpace(parts[1])
    }
    if path == "" {
        table := m.results.SourceTable()
        if table == "" {
            table = "results"
        }
        path = fmt.Sprintf("./%s.%s", table, format)
    }

    result := m.results.Result()
    var err error
    switch format {
    case "csv":
        err = exportCSV(path, result)
    case "json":
        err = exportJSON(path, result)
    }

    if err != nil {
        m.status = fmt.Sprintf("Export failed: %v", err)
    } else {
        m.status = fmt.Sprintf("Exported %d rows to %s", len(result.Rows), path)
    }
    return m, nil
}
```

**Step 3: Implement exportCSV and exportJSON**

Add to `internal/app/app.go` (or a new file `internal/app/export.go`):

```go
// internal/app/export.go
package app

import (
    "encoding/csv"
    "encoding/json"
    "os"

    "github.com/bklimczak/dex/internal/db"
)

func exportCSV(path string, result *db.QueryResult) error {
    f, err := os.Create(path)
    if err != nil {
        return err
    }
    defer f.Close()
    w := csv.NewWriter(f)
    w.Write(result.Columns)
    for _, row := range result.Rows {
        w.Write(row)
    }
    w.Flush()
    return w.Error()
}

func exportJSON(path string, result *db.QueryResult) error {
    var records []map[string]string
    for _, row := range result.Rows {
        record := make(map[string]string)
        for i, col := range result.Columns {
            if i < len(row) {
                record[col] = row[i]
            }
        }
        records = append(records, record)
    }
    data, err := json.MarshalIndent(records, "", "  ")
    if err != nil {
        return err
    }
    return os.WriteFile(path, data, 0644)
}
```

**Step 4: Add Result() accessor to results model**

In `internal/ui/results/results.go`:

```go
func (m *Model) Result() *db.QueryResult {
    return m.result
}
```

**Step 5: Build and verify**

Run: `go build -o dex .`

**Step 6: Commit**

```bash
git add internal/app/app.go internal/app/export.go internal/ui/results/results.go
git commit -m "feat: export results to CSV or JSON via :export command"
```

---

### Task 5: Server-Side Pagination

**Files:**
- Modify: `internal/app/app.go` (add pagination state, count query, page navigation)
- Modify: `internal/ui/results/results.go` (add n/p keys, page status display)

**Step 1: Add pagination state to app Model**

In `internal/app/app.go`, add fields to Model:

```go
pageOffset   int
pageSize     int
totalRows    int
currentTable string
sortColumn   string
sortDesc     bool
```

Initialize `pageSize: 100` in `New()`.

**Step 2: Add count message type**

```go
type tableCountMsg struct {
    table string
    count int
    err   error
}
```

**Step 3: Add count command**

```go
func (m Model) countTableCmd(table string) tea.Cmd {
    return func() tea.Msg {
        engine := m.registry.Active()
        if engine == nil {
            return tableCountMsg{err: fmt.Errorf("no active connection")}
        }
        var count int
        err := engine.DB().QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", table)).Scan(&count)
        return tableCountMsg{table: table, count: count, err: err}
    }
}
```

**Step 4: Modify TableSelectedMsg handler**

When a table is selected from sidebar, reset pagination and fire both count and data queries:

```go
case sidebar.TableSelectedMsg:
    m.registry.SetActive(msg.Connection)
    m.results.SetSourceTable(msg.Table)
    m.currentTable = msg.Table
    m.pageOffset = 0
    m.sortColumn = ""
    m.sortDesc = false
    m.status = fmt.Sprintf("Loading %s.%s...", msg.Connection, msg.Table)
    return m, tea.Batch(
        m.loadPageCmd(),
        m.countTableCmd(msg.Table),
    )
```

**Step 5: Add loadPageCmd helper**

```go
func (m Model) loadPageCmd() tea.Cmd {
    query := fmt.Sprintf("SELECT * FROM %s", m.currentTable)
    if m.sortColumn != "" {
        order := "ASC"
        if m.sortDesc {
            order = "DESC"
        }
        query += fmt.Sprintf(" ORDER BY %s %s", m.sortColumn, order)
    }
    query += fmt.Sprintf(" LIMIT %d OFFSET %d", m.pageSize, m.pageOffset)
    return m.executeQueryCmd(query)
}
```

**Step 6: Handle tableCountMsg**

```go
case tableCountMsg:
    if msg.err == nil {
        m.totalRows = msg.count
    }
    return m, nil
```

**Step 7: Add PageNextMsg / PagePrevMsg to results**

In `internal/ui/results/results.go`, add:

```go
type PageNextMsg struct{}
type PagePrevMsg struct{}
```

Add key handlers:

```go
case key.Matches(msg, key.NewBinding(key.WithKeys("n"))):
    return m, func() tea.Msg { return PageNextMsg{} }
case key.Matches(msg, key.NewBinding(key.WithKeys("p"))):
    return m, func() tea.Msg { return PagePrevMsg{} }
```

**Step 8: Handle page messages in app.go**

```go
case results.PageNextMsg:
    if m.currentTable != "" && m.pageOffset+m.pageSize < m.totalRows {
        m.pageOffset += m.pageSize
        m.status = "Loading next page..."
        return m, m.loadPageCmd()
    }
    return m, nil

case results.PagePrevMsg:
    if m.currentTable != "" && m.pageOffset > 0 {
        m.pageOffset -= m.pageSize
        if m.pageOffset < 0 {
            m.pageOffset = 0
        }
        m.status = "Loading previous page..."
        return m, m.loadPageCmd()
    }
    return m, nil
```

**Step 9: Update sort handler to use pagination**

Replace the `SortMsg` handler to use `loadPageCmd`:

```go
case results.SortMsg:
    if m.currentTable != "" {
        m.sortColumn = msg.Column
        m.sortDesc = msg.Desc
        m.pageOffset = 0
        m.status = fmt.Sprintf("Sorting by %s...", msg.Column)
        return m, m.loadPageCmd()
    }
```

**Step 10: Update results status line for pagination**

In `internal/ui/results/results.go`, add pagination fields:

```go
// Add to Model:
page      int
totalRows int
pageSize  int
```

Add setters:

```go
func (m *Model) SetPagination(page, totalRows, pageSize int) {
    m.page = page
    m.totalRows = totalRows
    m.pageSize = pageSize
}
```

Update status line in View:

```go
status := fmt.Sprintf(" %d rows | row %d/%d ", m.result.RowCount, m.cursorRow+1, len(m.result.Rows))
if m.totalRows > 0 {
    startRow := (m.page-1)*m.pageSize + 1
    endRow := startRow + len(m.result.Rows) - 1
    totalPages := (m.totalRows + m.pageSize - 1) / m.pageSize
    status = fmt.Sprintf(" page %d/%d | %d-%d of %d | row %d/%d ",
        m.page, totalPages, startRow, endRow, m.totalRows,
        m.cursorRow+1, len(m.result.Rows))
}
```

Call `SetPagination` from app.go in `queryResultMsg` handler when `currentTable` is set:

```go
case queryResultMsg:
    // ... existing error handling ...
    m.results.SetResult(msg.result)
    if m.currentTable != "" {
        page := (m.pageOffset / m.pageSize) + 1
        m.results.SetPagination(page, m.totalRows, m.pageSize)
    }
    m.setFocus(paneResults)
    return m, nil
```

**Step 11: Build and verify**

Run: `go build -o dex .`

**Step 12: Commit**

```bash
git add internal/ui/results/results.go internal/app/app.go
git commit -m "feat: server-side pagination with n/p keys"
```

---

### Task 6: Column Hide/Show

**Files:**
- Modify: `internal/ui/results/results.go` (add hidden columns set, `c`/`C` keys, filter in View)

**Step 1: Add hidden columns state**

In `internal/ui/results/results.go`, add to Model:

```go
hiddenCols map[int]bool
```

Initialize in `SetResult`:

```go
m.hiddenCols = make(map[int]bool)
```

**Step 2: Add `c` and `C` key handlers**

```go
case key.Matches(msg, key.NewBinding(key.WithKeys("c"))):
    if m.result != nil && len(m.result.Columns) > 1 {
        m.hiddenCols[m.cursorCol] = true
        // Move cursor if current column is now hidden
        for m.hiddenCols[m.cursorCol] && m.cursorCol < len(m.result.Columns)-1 {
            m.cursorCol++
        }
        if m.hiddenCols[m.cursorCol] && m.cursorCol > 0 {
            m.cursorCol--
            for m.hiddenCols[m.cursorCol] && m.cursorCol > 0 {
                m.cursorCol--
            }
        }
    }
case key.Matches(msg, key.NewBinding(key.WithKeys("C"))):
    m.hiddenCols = make(map[int]bool)
```

**Step 3: Update View to skip hidden columns**

In the View method, wherever columns are iterated (header, separator, rows), add a skip check:

```go
for i := startCol; i < endCol; i++ {
    if m.hiddenCols[i] {
        continue
    }
    // ... existing rendering ...
}
```

Apply this to all three loops: header, separator, and row rendering.

**Step 4: Update status line**

Add hidden count to status:

```go
if len(m.hiddenCols) > 0 {
    status += fmt.Sprintf("| %d cols hidden (C to show all) ", len(m.hiddenCols))
}
```

**Step 5: Build and verify**

Run: `go build -o dex .`

**Step 6: Commit**

```bash
git add internal/ui/results/results.go
git commit -m "feat: hide/show columns with c/C keys"
```
