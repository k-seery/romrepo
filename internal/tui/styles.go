package tui

import "github.com/charmbracelet/lipgloss"

var (
	StyleTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("15")). // bright white
			Background(lipgloss.Color("62")). // muted purple
			Padding(0, 1)

	StyleBreadcrumb = lipgloss.NewStyle().
			Foreground(lipgloss.Color("247")). // grey
			Padding(0, 1)

	StyleStatusBar = lipgloss.NewStyle().
			Foreground(lipgloss.Color("247")).
			Padding(0, 1)

	StyleError = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")). // red
			Padding(0, 1)

	StyleOnBoth = lipgloss.NewStyle().
			Foreground(lipgloss.Color("15")) // bright white

	StyleServerOnly = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")) // grey

	StyleHelp = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Padding(0, 1)

	StyleSelected = lipgloss.NewStyle().
			Foreground(lipgloss.Color("212")). // pink
			Bold(true)

	StyleSSHOpen = lipgloss.NewStyle().
			Foreground(lipgloss.Color("82")) // green
)
