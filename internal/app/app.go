// internal/app/app.go
package app

import (
	"fmt"
	"strings"

	"github.com/bklimczak/dex/internal/config"
	"github.com/bklimczak/dex/internal/db"
	"github.com/bklimczak/dex/internal/keymap"
	"github.com/bklimczak/dex/internal/ui/cmdbar"
	"github.com/bklimczak/dex/internal/ui/connform"
	"github.com/bklimczak/dex/internal/ui/connpicker"
	"github.com/bklimczak/dex/internal/ui/editor"
	"github.com/bklimczak/dex/internal/ui/querypicker"
	"github.com/bklimczak/dex/internal/ui/querybar"
	"github.com/bklimczak/dex/internal/ui/results"
	"github.com/bklimczak/dex/internal/ui/schema"
	"github.com/bklimczak/dex/internal/ui/sidebar"
	"github.com/bklimczak/dex/internal/ui/styles"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type pane int

const (
	paneSidebar pane = iota
	paneResults
	paneQueryBar
	paneCount
)

type modal int

const (
	modalNone modal = iota
	modalConnForm
	modalEditor
	modalSchema
	modalCommand
	modalConnPicker
	modalQueryPicker
)

// Messages
type tablesLoadedMsg struct {
	connName string
	tables   []string
	err      error
}

type queryResultMsg struct {
	result *db.QueryResult
	err    error
}

type connectResultMsg struct {
	connName string
	err      error
}

type schemaLoadedMsg struct {
	schema *db.TableSchema
	err    error
}

type testConnResultMsg struct {
	err error
}

type columnsLoadedMsg struct {
	connName string
	table    string
	columns  []string
}

type Model struct {
	sidebar  sidebar.Model
	results  results.Model
	querybar querybar.Model

	registry *db.Registry
	cfg      *config.Config
	cfgPath  string

	connForm     connform.Model
	editorModal  editor.Model
	schemaModal  schema.Model
	cmdBar       cmdbar.Model
	connPicker   connpicker.Model
	queryPicker  querypicker.Model
	savedQueries *config.SavedQueries
	queriesPath  string
	lastQuery    string
	queryHistory []string
	historyPath  string

	focus  pane
	modal  modal
	width  int
	height int
	keys   keymap.KeyMap
	status string
}

func New(cfg *config.Config, cfgPath string) Model {
	sb := sidebar.New()
	sb.SetFocused(true)

	histPath := config.HistoryPath()
	history := config.LoadHistory(histPath)
	qPath := config.QueriesPath()
	sq := config.LoadQueries(qPath)

	qb := querybar.New()
	// Pre-load history into querybar
	for _, q := range history {
		qb.AddToHistory(q)
	}

	return Model{
		sidebar:      sb,
		results:      results.New(),
		querybar:     qb,
		registry:     db.NewRegistry(),
		cfg:          cfg,
		cfgPath:      cfgPath,
		savedQueries: sq,
		queriesPath:  qPath,
		queryHistory: history,
		historyPath:  histPath,
		focus:        paneSidebar,
		keys:         keymap.Default,
	}
}

func (m Model) Init() tea.Cmd {
	var cmds []tea.Cmd
	for _, conn := range m.cfg.Connections {
		conn := conn
		cmds = append(cmds, m.connectCmd(conn))
	}
	return tea.Batch(cmds...)
}

func (m Model) connectCmd(conn config.Connection) tea.Cmd {
	return func() tea.Msg {
		engine, err := db.NewEngine(conn.Engine)
		if err != nil {
			return connectResultMsg{connName: conn.Name, err: err}
		}
		cfg := db.ConnectionConfig{
			Name:     conn.Name,
			Engine:   conn.Engine,
			Host:     conn.Host,
			Port:     conn.Port,
			Database: conn.Database,
			User:     conn.User,
			Password: conn.Password,
			SSL:      conn.SSL,
		}
		if err := engine.Connect(cfg); err != nil {
			return connectResultMsg{connName: conn.Name, err: err}
		}
		return connectResultMsg{connName: conn.Name, err: nil}
	}
}

func (m Model) loadTablesCmd(connName string) tea.Cmd {
	return func() tea.Msg {
		engine := m.registry.Get(connName)
		if engine == nil {
			return tablesLoadedMsg{connName: connName, err: fmt.Errorf("not connected")}
		}
		tables, err := engine.Tables()
		return tablesLoadedMsg{connName: connName, tables: tables, err: err}
	}
}

func (m Model) executeQueryCmd(query string) tea.Cmd {
	return func() tea.Msg {
		engine := m.registry.Active()
		if engine == nil {
			return queryResultMsg{err: fmt.Errorf("no active connection")}
		}
		result, err := engine.Execute(query)
		return queryResultMsg{result: result, err: err}
	}
}

func (m Model) loadSchemaCmd(table string) tea.Cmd {
	return func() tea.Msg {
		engine := m.registry.Active()
		if engine == nil {
			return schemaLoadedMsg{err: fmt.Errorf("no active connection")}
		}
		s, err := engine.Schema(table)
		return schemaLoadedMsg{schema: s, err: err}
	}
}

func (m Model) handleCommand(input string) (tea.Model, tea.Cmd) {
	cmd := input
	var arg string
	if idx := strings.IndexByte(input, ' '); idx != -1 {
		cmd = input[:idx]
		arg = strings.TrimSpace(input[idx+1:])
	}

	switch cmd {
	case "q", "quit":
		m.registry.CloseAll()
		return m, tea.Quit
	case "conn", "connections":
		return m.openConnPicker()
	case "reload":
		return m.reloadConfig()
	case "save":
		return m.saveQuery(arg)
	case "queries", "q!":
		return m.openQueryPicker()
	default:
		// Treat as SQL query
		if input != "" {
			m.queryHistory = append(m.queryHistory, input)
			config.SaveHistory(m.historyPath, m.queryHistory)
			m.querybar.AddToHistory(input)
			m.lastQuery = input
			m.status = "Executing query..."
			return m, m.executeQueryCmd(input)
		}
	}
	return m, nil
}

func (m Model) saveQuery(name string) (tea.Model, tea.Cmd) {
	if name == "" {
		m.status = "Usage: :save <name>"
		return m, nil
	}
	if m.lastQuery == "" {
		m.status = "No query to save"
		return m, nil
	}
	connName := m.registry.ActiveName()
	if connName == "" {
		m.status = "No active connection"
		return m, nil
	}
	sq := config.SavedQuery{Name: name, Query: m.lastQuery}
	m.savedQueries.Connections[connName] = append(m.savedQueries.Connections[connName], sq)
	config.SaveQueries(m.queriesPath, m.savedQueries)
	m.status = fmt.Sprintf("Saved query '%s' for %s", name, connName)
	return m, nil
}

func (m Model) openQueryPicker() (tea.Model, tea.Cmd) {
	connName := m.registry.ActiveName()
	var queries []config.SavedQuery
	if connName != "" {
		queries = m.savedQueries.Connections[connName]
	}
	m.queryPicker = querypicker.New(queries)
	m.queryPicker.SetSize(m.width, m.height)
	m.modal = modalQueryPicker
	return m, m.queryPicker.Init()
}

func (m Model) reloadConfig() (tea.Model, tea.Cmd) {
	cfg, err := config.Load(m.cfgPath)
	if err != nil {
		m.status = fmt.Sprintf("Reload failed: %v", err)
		return m, nil
	}
	m.cfg = cfg
	// Connect any new connections not already in the registry
	existing := make(map[string]bool)
	for _, name := range m.registry.Names() {
		existing[name] = true
	}
	var cmds []tea.Cmd
	for _, conn := range cfg.Connections {
		if !existing[conn.Name] {
			cmds = append(cmds, m.connectCmd(conn))
		}
	}
	if len(cmds) > 0 {
		m.status = fmt.Sprintf("Reloaded config, connecting %d new connection(s)...", len(cmds))
	} else {
		m.status = "Config reloaded, no new connections"
	}
	return m, tea.Batch(cmds...)
}

func (m Model) openConnPicker() (tea.Model, tea.Cmd) {
	activeName := m.registry.ActiveName()
	var entries []connpicker.Entry
	for _, name := range m.registry.Names() {
		cfg := m.registry.GetConfig(name)
		entries = append(entries, connpicker.Entry{
			Name:     name,
			Engine:   cfg.Engine,
			Database: cfg.Database,
			Host:     cfg.Host,
			Active:   name == activeName,
		})
	}
	m.connPicker = connpicker.New(entries)
	m.connPicker.SetSize(m.width, m.height)
	m.modal = modalConnPicker
	return m, nil
}

func (m *Model) setFocus(p pane) {
	m.focus = p
	m.sidebar.SetFocused(p == paneSidebar)
	m.results.SetFocused(p == paneResults)
	m.querybar.SetFocused(p == paneQueryBar)
}

func (m *Model) cycleFocus(forward bool) {
	if forward {
		m.setFocus((m.focus + 1) % paneCount)
	} else {
		m.setFocus((m.focus - 1 + paneCount) % paneCount)
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.updateSizes()
		return m, nil

	case connectResultMsg:
		if msg.err != nil {
			m.status = fmt.Sprintf("Failed to connect %s: %v", msg.connName, msg.err)
			return m, nil
		}
		// Find the config and register
		for _, conn := range m.cfg.Connections {
			if conn.Name == msg.connName {
				engine, _ := db.NewEngine(conn.Engine)
				cfg := db.ConnectionConfig{
					Name: conn.Name, Engine: conn.Engine,
					Host: conn.Host, Port: conn.Port,
					Database: conn.Database, User: conn.User,
					Password: conn.Password, SSL: conn.SSL,
				}
				engine.Connect(cfg)
				m.registry.Add(conn.Name, engine, cfg)
				break
			}
		}
		m.status = fmt.Sprintf("Connected to %s", msg.connName)
		m.updateSidebar()
		return m, m.loadTablesCmd(msg.connName)

	case tablesLoadedMsg:
		if msg.err != nil {
			m.status = fmt.Sprintf("Error loading tables for %s: %v", msg.connName, msg.err)
			return m, nil
		}
		m.sidebar.UpdateTables(msg.connName, msg.tables)
		// Update completer with table names
		m.querybar.Completer().SetTables(msg.tables)
		// Load columns for each table in background
		var cmds []tea.Cmd
		for _, table := range msg.tables {
			table := table
			connName := msg.connName
			cmds = append(cmds, func() tea.Cmd {
				return func() tea.Msg {
					engine := m.registry.Get(connName)
					if engine == nil {
						return nil
					}
					s, err := engine.Schema(table)
					if err != nil {
						return nil
					}
					var cols []string
					for _, c := range s.Columns {
						cols = append(cols, c.Name)
					}
					return columnsLoadedMsg{connName: connName, table: table, columns: cols}
				}
			}())
		}
		m.status = fmt.Sprintf("Loaded %d tables from %s", len(msg.tables), msg.connName)
		return m, tea.Batch(cmds...)

	case columnsLoadedMsg:
		m.querybar.Completer().SetColumns(msg.table, msg.columns)
		return m, nil

	case queryResultMsg:
		if msg.err != nil {
			m.results.SetError(msg.err.Error())
			return m, nil
		}
		m.results.SetResult(msg.result)
		m.setFocus(paneResults)
		return m, nil

	case sidebar.TableSelectedMsg:
		m.registry.SetActive(msg.Connection)
		query := fmt.Sprintf("SELECT * FROM %s LIMIT 100", msg.Table)
		m.status = fmt.Sprintf("Loading %s.%s...", msg.Connection, msg.Table)
		return m, m.executeQueryCmd(query)

	case sidebar.ConnectionSelectedMsg:
		m.registry.SetActive(msg.Connection)
		m.status = fmt.Sprintf("Active: %s", msg.Connection)
		return m, nil

	case querybar.ExecuteQueryMsg:
		m.status = "Executing query..."
		m.lastQuery = msg.Query
		m.queryHistory = append(m.queryHistory, msg.Query)
		config.SaveHistory(m.historyPath, m.queryHistory)
		return m, m.executeQueryCmd(msg.Query)

	case connform.SaveConnectionMsg:
		conn := msg.Connection
		m.cfg.Connections = append(m.cfg.Connections, conn)
		config.Save(m.cfgPath, m.cfg)
		m.modal = modalNone
		m.status = fmt.Sprintf("Saved connection %s, connecting...", conn.Name)
		return m, m.connectCmd(conn)

	case connform.CancelMsg:
		m.modal = modalNone
		return m, nil

	case connform.TestConnectionMsg:
		conn := msg.Connection
		return m, func() tea.Msg {
			engine, err := db.NewEngine(conn.Engine)
			if err != nil {
				return testConnResultMsg{err: err}
			}
			cfg := db.ConnectionConfig{
				Name: conn.Name, Engine: conn.Engine,
				Host: conn.Host, Port: conn.Port,
				Database: conn.Database, User: conn.User,
				Password: conn.Password, SSL: conn.SSL,
			}
			if err := engine.Connect(cfg); err != nil {
				return testConnResultMsg{err: err}
			}
			engine.Close()
			return testConnResultMsg{err: nil}
		}

	case testConnResultMsg:
		if msg.err != nil {
			m.connForm.SetTestResult(styles.ErrorText.Render("Connection failed: " + msg.err.Error()))
		} else {
			m.connForm.SetTestResult(lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Render("Connection successful!"))
		}
		return m, nil

	case editor.ExecuteMsg:
		m.modal = modalNone
		m.status = "Executing query..."
		m.lastQuery = msg.Query
		m.queryHistory = append(m.queryHistory, msg.Query)
		config.SaveHistory(m.historyPath, m.queryHistory)
		return m, m.executeQueryCmd(msg.Query)

	case editor.CloseMsg:
		m.modal = modalNone
		return m, nil

	case schemaLoadedMsg:
		if msg.err != nil {
			m.status = fmt.Sprintf("Schema error: %v", msg.err)
			return m, nil
		}
		m.schemaModal = schema.New(msg.schema, m.width, m.height)
		m.modal = modalSchema
		return m, nil

	case schema.CloseMsg:
		m.modal = modalNone
		return m, nil

	case cmdbar.ExecuteCommandMsg:
		m.modal = modalNone
		return m.handleCommand(msg.Command)

	case cmdbar.CancelMsg:
		m.modal = modalNone
		return m, nil

	case connpicker.SelectMsg:
		m.modal = modalNone
		m.registry.SetActive(msg.Name)
		m.status = fmt.Sprintf("Switched to %s", msg.Name)
		m.updateSidebar()
		return m, m.loadTablesCmd(msg.Name)

	case connpicker.CloseMsg:
		m.modal = modalNone
		return m, nil

	case querypicker.SelectMsg:
		m.modal = modalNone
		m.lastQuery = msg.Query
		m.queryHistory = append(m.queryHistory, msg.Query)
		config.SaveHistory(m.historyPath, m.queryHistory)
		m.querybar.AddToHistory(msg.Query)
		m.status = "Executing saved query..."
		return m, m.executeQueryCmd(msg.Query)

	case querypicker.CloseMsg:
		m.modal = modalNone
		return m, nil

	case querypicker.DeleteMsg:
		connName := m.registry.ActiveName()
		if connName != "" {
			queries := m.savedQueries.Connections[connName]
			for i, sq := range queries {
				if sq.Name == msg.Name {
					m.savedQueries.Connections[connName] = append(queries[:i], queries[i+1:]...)
					break
				}
			}
			config.SaveQueries(m.queriesPath, m.savedQueries)
			// Reopen picker with updated list
			return m.openQueryPicker()
		}
		return m, nil

	case tea.KeyMsg:
		// Global keys (when no modal is open and query bar is not focused)
		if m.modal == modalNone && m.focus != paneQueryBar && !m.sidebar.Filtering() {
			switch {
			case key.Matches(msg, m.keys.QueryBar):
				m.cmdBar = cmdbar.New()
				m.cmdBar.SetWidth(m.width)
				m.modal = modalCommand
				return m, m.cmdBar.Init()
			case key.Matches(msg, m.keys.FocusNext):
				m.cycleFocus(true)
				return m, nil
			case key.Matches(msg, m.keys.FocusPrev):
				m.cycleFocus(false)
				return m, nil
			case key.Matches(msg, m.keys.FocusLeft):
				m.setFocus(paneSidebar)
				return m, nil
			case key.Matches(msg, m.keys.FocusRight):
				m.setFocus(paneResults)
				return m, nil
			case key.Matches(msg, m.keys.NewConn):
				m.connForm = connform.New()
				m.connForm.SetSize(m.width, m.height)
				m.modal = modalConnForm
				return m, m.connForm.Init()
			case key.Matches(msg, m.keys.Editor):
				m.editorModal = editor.New()
				m.editorModal.SetSize(m.width, m.height)
				m.editorModal.SetHistory(m.queryHistory)
				// Copy completions to editor
				m.editorModal.Completer().CopyFrom(m.querybar.Completer())
				m.modal = modalEditor
				return m, m.editorModal.Init()
			case key.Matches(msg, m.keys.SchemaView):
				if m.focus == paneSidebar {
					if _, table, ok := m.sidebar.SelectedTable(); ok {
						m.status = fmt.Sprintf("Loading schema for %s...", table)
						return m, m.loadSchemaCmd(table)
					}
				}
			case key.Matches(msg, m.keys.Describe):
				if m.focus == paneSidebar {
					if _, table, ok := m.sidebar.SelectedTable(); ok {
						m.status = fmt.Sprintf("Describing %s...", table)
						query := fmt.Sprintf("SELECT column_name, data_type, is_nullable, column_default FROM information_schema.columns WHERE table_name = '%s' ORDER BY ordinal_position", table)
						return m, m.executeQueryCmd(query)
					}
				}
			}

			// Quick-switch connections 1-9
			for i := 1; i <= 9; i++ {
				if key.Matches(msg, keymap.ConnBinding(i)) {
					names := m.registry.Names()
					if i <= len(names) {
						m.registry.SetActive(names[i-1])
						m.status = fmt.Sprintf("Switched to %s", names[i-1])
					}
					return m, nil
				}
			}
		}

		// Escape from query bar
		if m.focus == paneQueryBar && msg.String() == "esc" {
			m.setFocus(paneSidebar)
			return m, nil
		}

		// ctrl+c always quits
		if msg.String() == "ctrl+c" {
			m.registry.CloseAll()
			return m, tea.Quit
		}
	}

	// Delegate to active modal first
	switch m.modal {
	case modalConnForm:
		var cmd tea.Cmd
		m.connForm, cmd = m.connForm.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		return m, tea.Batch(cmds...)
	case modalEditor:
		var cmd tea.Cmd
		m.editorModal, cmd = m.editorModal.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		return m, tea.Batch(cmds...)
	case modalSchema:
		var cmd tea.Cmd
		m.schemaModal, cmd = m.schemaModal.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		return m, tea.Batch(cmds...)
	case modalCommand:
		var cmd tea.Cmd
		m.cmdBar, cmd = m.cmdBar.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		return m, tea.Batch(cmds...)
	case modalConnPicker:
		var cmd tea.Cmd
		m.connPicker, cmd = m.connPicker.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		return m, tea.Batch(cmds...)
	case modalQueryPicker:
		var cmd tea.Cmd
		m.queryPicker, cmd = m.queryPicker.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		return m, tea.Batch(cmds...)
	}

	// Delegate to focused pane
	switch m.focus {
	case paneSidebar:
		var cmd tea.Cmd
		m.sidebar, cmd = m.sidebar.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	case paneResults:
		var cmd tea.Cmd
		m.results, cmd = m.results.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	case paneQueryBar:
		var cmd tea.Cmd
		m.querybar, cmd = m.querybar.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

func (m *Model) updateSizes() {
	sidebarWidth := m.width / 4
	if sidebarWidth < 20 {
		sidebarWidth = 20
	}
	if sidebarWidth > 40 {
		sidebarWidth = 40
	}
	mainWidth := m.width - sidebarWidth - 3
	queryBarHeight := 3
	resultsHeight := m.height - queryBarHeight - 2

	m.sidebar.SetSize(sidebarWidth, m.height-2)
	m.results.SetSize(mainWidth, resultsHeight)
	m.querybar.SetWidth(mainWidth)
}

func (m *Model) updateSidebar() {
	names := m.registry.Names()
	tables := make(map[string][]string)
	m.sidebar.SetConnections(names, tables)
}

func (m Model) View() string {
	sidebarWidth := m.width / 4
	if sidebarWidth < 20 {
		sidebarWidth = 20
	}
	if sidebarWidth > 40 {
		sidebarWidth = 40
	}

	sidebarStyle := styles.InactiveBorder
	if m.focus == paneSidebar {
		sidebarStyle = styles.ActiveBorder
	}
	sidebarView := sidebarStyle.
		Width(sidebarWidth).
		Height(m.height - 2).
		Render(m.sidebar.View())

	resultsStyle := styles.InactiveBorder
	if m.focus == paneResults {
		resultsStyle = styles.ActiveBorder
	}
	mainWidth := m.width - sidebarWidth - 5

	completerView := m.querybar.CompleterView()
	completerHeight := 0
	if completerView != "" {
		completerHeight = lipgloss.Height(completerView)
	}

	queryBarHeight := 3
	resultsHeight := m.height - queryBarHeight - completerHeight - 4

	resultsView := resultsStyle.
		Width(mainWidth).
		Height(resultsHeight).
		Render(m.results.View())

	qbStyle := styles.InactiveBorder
	if m.focus == paneQueryBar {
		qbStyle = styles.ActiveBorder
	}
	queryBarView := qbStyle.
		Width(mainWidth).
		Render(m.querybar.View())

	var rightParts []string
	rightParts = append(rightParts, resultsView)
	if completerView != "" {
		rightParts = append(rightParts, completerView)
	}
	rightParts = append(rightParts, queryBarView)

	rightSide := lipgloss.JoinVertical(lipgloss.Left, rightParts...)
	main := lipgloss.JoinHorizontal(lipgloss.Top, sidebarView, rightSide)
	var statusLine string
	if m.modal == modalCommand {
		statusLine = m.cmdBar.View()
	} else {
		statusLine = styles.StatusBar.Width(m.width).Render(m.status)
	}

	base := lipgloss.JoinVertical(lipgloss.Left, main, statusLine)

	// Overlay modals
	switch m.modal {
	case modalConnForm:
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center,
			m.connForm.View(), lipgloss.WithWhitespaceChars(" "), lipgloss.WithWhitespaceForeground(lipgloss.Color("236")))
	case modalEditor:
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center,
			m.editorModal.View(), lipgloss.WithWhitespaceChars(" "), lipgloss.WithWhitespaceForeground(lipgloss.Color("236")))
	case modalSchema:
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center,
			m.schemaModal.View(), lipgloss.WithWhitespaceChars(" "), lipgloss.WithWhitespaceForeground(lipgloss.Color("236")))
	case modalConnPicker:
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center,
			m.connPicker.View(), lipgloss.WithWhitespaceChars(" "), lipgloss.WithWhitespaceForeground(lipgloss.Color("236")))
	case modalQueryPicker:
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center,
			m.queryPicker.View(), lipgloss.WithWhitespaceChars(" "), lipgloss.WithWhitespaceForeground(lipgloss.Color("236")))
	}

	return base
}
