// internal/ui/connform/connform.go
package connform

import (
	"strconv"

	"github.com/bklimczak/dex/internal/config"
	"github.com/bklimczak/dex/internal/ui/styles"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type field int

const (
	fieldName field = iota
	fieldEngine
	fieldHost
	fieldPort
	fieldDatabase
	fieldUser
	fieldPassword
	fieldCount
)

type SaveConnectionMsg struct {
	Connection config.Connection
}

type CancelMsg struct{}

type TestConnectionMsg struct {
	Connection config.Connection
}

type Model struct {
	inputs     []textinput.Model
	focusIndex field
	width      int
	height     int
	editing    *config.Connection
	testResult string
}

func New() Model {
	inputs := make([]textinput.Model, fieldCount)

	names := []string{"Name", "Engine (postgres/mysql)", "Host", "Port", "Database", "User", "Password"}
	placeholders := []string{"my-database", "postgres", "localhost", "5432", "mydb", "user", ""}
	for i := range inputs {
		inputs[i] = textinput.New()
		inputs[i].Prompt = names[i] + ": "
		inputs[i].Placeholder = placeholders[i]
		inputs[i].CharLimit = 256
		if field(i) == fieldPassword {
			inputs[i].EchoMode = textinput.EchoPassword
		}
	}
	inputs[0].Focus()

	return Model{inputs: inputs}
}

func NewEditing(conn *config.Connection) Model {
	m := New()
	m.editing = conn
	m.inputs[fieldName].SetValue(conn.Name)
	m.inputs[fieldEngine].SetValue(conn.Engine)
	m.inputs[fieldHost].SetValue(conn.Host)
	m.inputs[fieldPort].SetValue(strconv.Itoa(conn.Port))
	m.inputs[fieldDatabase].SetValue(conn.Database)
	m.inputs[fieldUser].SetValue(conn.User)
	m.inputs[fieldPassword].SetValue(conn.Password)
	return m
}

func (m *Model) SetSize(w, h int) {
	m.width = w
	m.height = h
	for i := range m.inputs {
		m.inputs[i].Width = w - 30
	}
}

func (m *Model) SetTestResult(s string) {
	m.testResult = s
}

func (m Model) toConnection() config.Connection {
	port, _ := strconv.Atoi(m.inputs[fieldPort].Value())
	return config.Connection{
		Name:     m.inputs[fieldName].Value(),
		Engine:   m.inputs[fieldEngine].Value(),
		Host:     m.inputs[fieldHost].Value(),
		Port:     port,
		Database: m.inputs[fieldDatabase].Value(),
		User:     m.inputs[fieldUser].Value(),
		Password: m.inputs[fieldPassword].Value(),
	}
}

func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return m, func() tea.Msg { return CancelMsg{} }
		case "tab", "down":
			m.inputs[m.focusIndex].Blur()
			m.focusIndex = (m.focusIndex + 1) % fieldCount
			m.inputs[m.focusIndex].Focus()
			return m, nil
		case "shift+tab", "up":
			m.inputs[m.focusIndex].Blur()
			m.focusIndex = (m.focusIndex - 1 + fieldCount) % fieldCount
			m.inputs[m.focusIndex].Focus()
			return m, nil
		case "ctrl+t":
			conn := m.toConnection()
			return m, func() tea.Msg { return TestConnectionMsg{Connection: conn} }
		case "ctrl+s":
			conn := m.toConnection()
			if conn.Name == "" || conn.Engine == "" || conn.Host == "" {
				m.testResult = "Name, engine, and host are required"
				return m, nil
			}
			return m, func() tea.Msg { return SaveConnectionMsg{Connection: conn} }
		}
	}

	var cmd tea.Cmd
	m.inputs[m.focusIndex], cmd = m.inputs[m.focusIndex].Update(msg)
	return m, cmd
}

func (m Model) View() string {
	var content string

	title := styles.ModalTitle.Render("New Connection")
	if m.editing != nil {
		title = styles.ModalTitle.Render("Edit Connection")
	}
	content += title + "\n\n"

	for i := range m.inputs {
		content += m.inputs[i].View() + "\n"
	}

	content += "\n"
	if m.testResult != "" {
		content += m.testResult + "\n"
	}
	content += lipgloss.NewStyle().Foreground(lipgloss.Color("240")).
		Render("Ctrl+t: test | Ctrl+s: save | Esc: cancel")

	w := 60
	if m.width > 0 && m.width-10 < w {
		w = m.width - 10
	}

	return styles.ModalOverlay.
		Width(w).
		Render(content)
}
