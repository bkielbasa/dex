// internal/ui/styles/styles.go
package styles

import "github.com/charmbracelet/lipgloss"

var (
	ActiveBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62"))

	InactiveBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240"))

	SelectedItem = lipgloss.NewStyle().
			Foreground(lipgloss.Color("170")).
			Bold(true)

	NormalItem = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	TreeIndent = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))

	HeaderCell = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("62")).
			Padding(0, 1)

	DataCell = lipgloss.NewStyle().
			Padding(0, 1)

	NullCell = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Italic(true).
			Padding(0, 1)

	ErrorText = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196"))

	StatusBar = lipgloss.NewStyle().
			Background(lipgloss.Color("236")).
			Foreground(lipgloss.Color("252")).
			Padding(0, 1)

	ModalOverlay = lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(1, 2)

	ModalTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("62"))
)
