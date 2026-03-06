// internal/keymap/keymap.go
package keymap

import "github.com/charmbracelet/bubbles/key"

type KeyMap struct {
	Quit          key.Binding
	Up            key.Binding
	Down          key.Binding
	Top           key.Binding
	Bottom        key.Binding
	Enter         key.Binding
	Escape        key.Binding
	QueryBar      key.Binding
	Editor        key.Binding
	NewConn       key.Binding
	ManageConns   key.Binding
	SchemaView    key.Binding
	Describe      key.Binding
	Search        key.Binding
	NextPage      key.Binding
	PrevPage      key.Binding
	HistoryPrev   key.Binding
	HistoryNext   key.Binding
	ExecuteEditor key.Binding
	Conn1         key.Binding
	Conn2         key.Binding
	Conn3         key.Binding
	Conn4         key.Binding
	Conn5         key.Binding
	Conn6         key.Binding
	Conn7         key.Binding
	Conn8         key.Binding
	Conn9         key.Binding
}

var Default = KeyMap{
	Quit:          key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit")),
	Up:            key.NewBinding(key.WithKeys("k", "up"), key.WithHelp("k", "up")),
	Down:          key.NewBinding(key.WithKeys("j", "down"), key.WithHelp("j", "down")),
	Top:           key.NewBinding(key.WithKeys("g"), key.WithHelp("g", "top")),
	Bottom:        key.NewBinding(key.WithKeys("G"), key.WithHelp("G", "bottom")),
	Enter:         key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select/execute")),
	Escape:        key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back/close")),
	QueryBar:      key.NewBinding(key.WithKeys(":"), key.WithHelp(":", "query bar")),
	Editor:        key.NewBinding(key.WithKeys("ctrl+e"), key.WithHelp("ctrl+e", "sql editor")),
	NewConn:       key.NewBinding(key.WithKeys("ctrl+n"), key.WithHelp("ctrl+n", "new connection")),
	ManageConns:   key.NewBinding(key.WithKeys("ctrl+d"), key.WithHelp("ctrl+d", "manage connections")),
	SchemaView:    key.NewBinding(key.WithKeys("S"), key.WithHelp("S", "schema")),
	Describe:      key.NewBinding(key.WithKeys("D"), key.WithHelp("D", "describe")),
	Search:        key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "search")),
	NextPage:      key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "next page")),
	PrevPage:      key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "prev page")),
	HistoryPrev:   key.NewBinding(key.WithKeys("ctrl+p"), key.WithHelp("ctrl+p", "prev query")),
	HistoryNext:   key.NewBinding(key.WithKeys("ctrl+n"), key.WithHelp("ctrl+n", "next query")),
	ExecuteEditor: key.NewBinding(key.WithKeys("ctrl+enter"), key.WithHelp("ctrl+enter", "run query")),
	Conn1:         key.NewBinding(key.WithKeys("1"), key.WithHelp("1", "connection 1")),
	Conn2:         key.NewBinding(key.WithKeys("2"), key.WithHelp("2", "connection 2")),
	Conn3:         key.NewBinding(key.WithKeys("3"), key.WithHelp("3", "connection 3")),
	Conn4:         key.NewBinding(key.WithKeys("4"), key.WithHelp("4", "connection 4")),
	Conn5:         key.NewBinding(key.WithKeys("5"), key.WithHelp("5", "connection 5")),
	Conn6:         key.NewBinding(key.WithKeys("6"), key.WithHelp("6", "connection 6")),
	Conn7:         key.NewBinding(key.WithKeys("7"), key.WithHelp("7", "connection 7")),
	Conn8:         key.NewBinding(key.WithKeys("8"), key.WithHelp("8", "connection 8")),
	Conn9:         key.NewBinding(key.WithKeys("9"), key.WithHelp("9", "connection 9")),
}

func ConnBinding(n int) key.Binding {
	bindings := []key.Binding{
		Default.Conn1, Default.Conn2, Default.Conn3,
		Default.Conn4, Default.Conn5, Default.Conn6,
		Default.Conn7, Default.Conn8, Default.Conn9,
	}
	if n >= 1 && n <= 9 {
		return bindings[n-1]
	}
	return key.Binding{}
}
