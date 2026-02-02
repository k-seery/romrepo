package tui

import "github.com/charmbracelet/lipgloss"

// Color palette
var (
	colorMagenta   = lipgloss.Color("165")
	colorPink      = lipgloss.Color("212")
	colorCyan      = lipgloss.Color("81")
	colorGreen     = lipgloss.Color("78")
	colorRed       = lipgloss.Color("196")
	colorWhite     = lipgloss.Color("15")
	colorLightGrey = lipgloss.Color("252")
	colorGrey      = lipgloss.Color("245")
	colorDimGrey   = lipgloss.Color("240")
	colorDarkGrey  = lipgloss.Color("236")
	colorFaintGrey = lipgloss.Color("241")
	colorPurple    = lipgloss.Color("62")
)

var (
	StyleTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorWhite).
			Background(colorPurple).
			Padding(0, 1)

	StyleBreadcrumb = lipgloss.NewStyle().
			Foreground(colorGrey).
			Padding(0, 1)

	StyleStatusBar = lipgloss.NewStyle().
			Background(colorDarkGrey).
			Foreground(colorLightGrey).
			Padding(0, 1)

	StyleError = lipgloss.NewStyle().
			Foreground(colorRed).
			Bold(true).
			Background(colorDarkGrey).
			Padding(0, 1)

	StyleOnBoth = lipgloss.NewStyle().
			Foreground(colorGreen)

	StyleServerOnly = lipgloss.NewStyle().
			Foreground(colorDimGrey)

	StyleHelp = lipgloss.NewStyle().
			Foreground(colorFaintGrey).
			Italic(true).
			Padding(0, 1)

	StyleSelected = lipgloss.NewStyle().
			Foreground(colorPink).
			Bold(true)

	StyleSSHOpen = lipgloss.NewStyle().
			Foreground(colorGreen).
			Bold(true)

	StylePanelFocused = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colorMagenta)

	StylePanelUnfocused = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colorDimGrey)

	StylePanelTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorGrey).
			Padding(0, 1)

	StylePanelTitleFocused = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorWhite).
				Background(colorMagenta).
				Padding(0, 1)

	StyleSeparator = lipgloss.NewStyle().
			Foreground(colorDimGrey)

	StyleFilterActive = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorPink).
				Underline(true)

	StyleFilterDim = lipgloss.NewStyle().
			Foreground(colorFaintGrey)

	StyleBanner = lipgloss.NewStyle().
			Foreground(colorMagenta)

	StyleInfoLabel = lipgloss.NewStyle().
			Foreground(colorMagenta).
			Bold(true)

	StyleInfoValue = lipgloss.NewStyle().
			Foreground(colorLightGrey)

	StyleInfoDim = lipgloss.NewStyle().
			Foreground(colorDimGrey).
			Italic(true)

	StyleHintKey = lipgloss.NewStyle().
			Foreground(colorPink).
			Bold(true)

	StyleHintSep = lipgloss.NewStyle().
			Foreground(colorDimGrey)

	StyleCursor = lipgloss.NewStyle().
			Foreground(colorPink).
			Bold(true)

	StyleSyncBadge = lipgloss.NewStyle().
			Foreground(colorGreen)

	StyleUnsyncBadge = lipgloss.NewStyle().
				Foreground(colorDimGrey)
)
