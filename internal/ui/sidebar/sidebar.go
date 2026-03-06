// internal/ui/sidebar/sidebar.go
package sidebar

import (
	"fmt"
	"strings"

	"github.com/bklimczak/dex/internal/ui/styles"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Messages
type TableSelectedMsg struct {
	Connection string
	Table      string
}

type ConnectionSelectedMsg struct {
	Connection string
}

// Tree node
type Node struct {
	Name     string
	Children []Node
	Expanded bool
}

type Model struct {
	nodes      []Node
	cursor     int
	focused    bool
	width      int
	height     int
	filter     string
	filtered   []flatNode
	filtering  bool
	filterInput textinput.Model
}

type flatNode struct {
	name     string
	depth    int
	nodeIdx  int
	childIdx int
	isConn   bool
	connName string
}

func New() Model {
	ti := textinput.New()
	ti.Placeholder = "filter tables..."
	ti.CharLimit = 64
	return Model{
		filterInput: ti,
	}
}

func (m *Model) SetSize(w, h int) {
	m.width = w
	m.height = h
}

func (m *Model) SelectedTable() (connName, tableName string, ok bool) {
	if m.cursor >= len(m.filtered) {
		return "", "", false
	}
	node := m.filtered[m.cursor]
	if node.isConn {
		return node.connName, "", false
	}
	return node.connName, node.name, true
}

func (m *Model) SetFocused(f bool) {
	m.focused = f
}

func (m *Model) Focused() bool {
	return m.focused
}

func (m *Model) Filtering() bool {
	return m.filtering
}

func (m *Model) SetConnections(names []string, tables map[string][]string) {
	m.nodes = make([]Node, len(names))
	for i, name := range names {
		children := make([]Node, len(tables[name]))
		for j, t := range tables[name] {
			children[j] = Node{Name: t}
		}
		m.nodes[i] = Node{
			Name:     name,
			Children: children,
			Expanded: i == 0,
		}
	}
	m.rebuildFlat()
}

func (m *Model) UpdateTables(connName string, tables []string) {
	for i := range m.nodes {
		if m.nodes[i].Name == connName {
			children := make([]Node, len(tables))
			for j, t := range tables {
				children[j] = Node{Name: t}
			}
			m.nodes[i].Children = children
			break
		}
	}
	m.rebuildFlat()
}

func (m *Model) rebuildFlat() {
	m.filtered = nil
	for i, node := range m.nodes {
		m.filtered = append(m.filtered, flatNode{
			name:     node.Name,
			depth:    0,
			nodeIdx:  i,
			childIdx: -1,
			isConn:   true,
			connName: node.Name,
		})
		if node.Expanded {
			for j, child := range node.Children {
				if m.filter == "" || strings.Contains(strings.ToLower(child.Name), strings.ToLower(m.filter)) {
					m.filtered = append(m.filtered, flatNode{
						name:     child.Name,
						depth:    1,
						nodeIdx:  i,
						childIdx: j,
						isConn:   false,
						connName: node.Name,
					})
				}
			}
		}
	}
	if m.cursor >= len(m.filtered) {
		m.cursor = max(0, len(m.filtered)-1)
	}
}

func (m *Model) SetFilter(f string) {
	m.filter = f
	m.rebuildFlat()
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if !m.focused {
			return m, nil
		}

		// Filter mode: delegate to text input
		if m.filtering {
			switch msg.String() {
			case "esc":
				m.filtering = false
				m.filterInput.Blur()
				m.filter = ""
				m.filterInput.SetValue("")
				m.rebuildFlat()
				return m, nil
			case "enter":
				m.filtering = false
				m.filterInput.Blur()
				return m, nil
			default:
				var cmd tea.Cmd
				m.filterInput, cmd = m.filterInput.Update(msg)
				m.filter = m.filterInput.Value()
				m.rebuildFlat()
				return m, cmd
			}
		}

		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("/"))):
			m.filtering = true
			m.filterInput.SetValue(m.filter)
			m.filterInput.Focus()
			return m, textinput.Blink
		case key.Matches(msg, key.NewBinding(key.WithKeys("j", "down"))):
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
			}
		case key.Matches(msg, key.NewBinding(key.WithKeys("k", "up"))):
			if m.cursor > 0 {
				m.cursor--
			}
		case key.Matches(msg, key.NewBinding(key.WithKeys("g"))):
			m.cursor = 0
		case key.Matches(msg, key.NewBinding(key.WithKeys("G"))):
			m.cursor = max(0, len(m.filtered)-1)
		case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
			if m.cursor < len(m.filtered) {
				node := m.filtered[m.cursor]
				if node.isConn {
					m.nodes[node.nodeIdx].Expanded = !m.nodes[node.nodeIdx].Expanded
					m.rebuildFlat()
					return m, func() tea.Msg {
						return ConnectionSelectedMsg{Connection: node.connName}
					}
				}
				return m, func() tea.Msg {
					return TableSelectedMsg{
						Connection: node.connName,
						Table:      node.name,
					}
				}
			}
		}
	}
	return m, nil
}

func (m Model) View() string {
	if len(m.filtered) == 0 && !m.filtering {
		return styles.NormalItem.Render("No connections.\nPress Ctrl+n to add one.")
	}

	var b strings.Builder

	if m.filtering || m.filter != "" {
		b.WriteString(m.filterInput.View())
		b.WriteString("\n")
	}

	visibleHeight := m.height - 2
	if m.filtering || m.filter != "" {
		visibleHeight -= 1
	}
	if visibleHeight < 1 {
		visibleHeight = 10
	}

	start := 0
	if m.cursor >= visibleHeight {
		start = m.cursor - visibleHeight + 1
	}

	for i := start; i < len(m.filtered) && i < start+visibleHeight; i++ {
		node := m.filtered[i]
		prefix := ""
		if node.isConn {
			if m.nodes[node.nodeIdx].Expanded {
				prefix = "▼ "
			} else {
				prefix = "▶ "
			}
		} else {
			prefix = "  └─ "
		}

		line := fmt.Sprintf("%s%s", prefix, node.name)

		if i == m.cursor && m.focused {
			line = styles.SelectedItem.Render(line)
		} else if node.isConn {
			line = styles.NormalItem.Bold(true).Render(line)
		} else {
			line = styles.NormalItem.Render(line)
		}

		if m.width > 0 && lipgloss.Width(line) > m.width-2 {
			line = line[:m.width-2]
		}

		b.WriteString(line)
		if i < start+visibleHeight-1 {
			b.WriteString("\n")
		}
	}

	return b.String()
}
