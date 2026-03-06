// internal/app/app.go
package app

import (
	"fmt"

	"github.com/bklimczak/dex/internal/config"
	"github.com/bklimczak/dex/internal/db"
	"github.com/bklimczak/dex/internal/keymap"
	"github.com/bklimczak/dex/internal/ui/querybar"
	"github.com/bklimczak/dex/internal/ui/results"
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

type Model struct {
	sidebar  sidebar.Model
	results  results.Model
	querybar querybar.Model

	registry *db.Registry
	cfg      *config.Config
	cfgPath  string

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

	return Model{
		sidebar:  sb,
		results:  results.New(),
		querybar: querybar.New(),
		registry: db.NewRegistry(),
		cfg:      cfg,
		cfgPath:  cfgPath,
		focus:    paneSidebar,
		keys:     keymap.Default,
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
		m.status = fmt.Sprintf("Loaded %d tables from %s", len(msg.tables), msg.connName)
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
		return m, m.executeQueryCmd(msg.Query)

	case tea.KeyMsg:
		// Global keys (when no modal is open and query bar is not focused)
		if m.modal == modalNone && m.focus != paneQueryBar {
			switch {
			case key.Matches(msg, m.keys.Quit):
				m.registry.CloseAll()
				return m, tea.Quit
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
			case key.Matches(msg, m.keys.QueryBar):
				m.setFocus(paneQueryBar)
				return m, nil
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
	queryBarHeight := 3
	resultsHeight := m.height - queryBarHeight - 4
	mainWidth := m.width - sidebarWidth - 5

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

	rightSide := lipgloss.JoinVertical(lipgloss.Left, resultsView, queryBarView)
	main := lipgloss.JoinHorizontal(lipgloss.Top, sidebarView, rightSide)
	status := styles.StatusBar.Width(m.width).Render(m.status)

	return lipgloss.JoinVertical(lipgloss.Left, main, status)
}
