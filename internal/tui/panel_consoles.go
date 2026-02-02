package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"romrepo/internal/config"
	"romrepo/internal/rom"
)

type ConsolePanel struct {
	app    *App
	items  []config.Console
	cursor int
	width  int
	height int
}

func NewConsolePanel(app *App) ConsolePanel {
	return ConsolePanel{
		app:   app,
		items: rom.DiscoverConsoles(app.cfg),
	}
}

func (p *ConsolePanel) SetSize(w, h int) {
	p.width = w
	p.height = h
}

func (p *ConsolePanel) Rebuild(cfg *config.Config) {
	p.items = rom.DiscoverConsoles(cfg)
	if p.cursor >= len(p.items) {
		p.cursor = max(0, len(p.items)-1)
	}
}

func (p *ConsolePanel) Update(msg tea.KeyMsg) tea.Cmd {
	switch {
	case key.Matches(msg, p.app.keys.Enter):
		if p.cursor >= 0 && p.cursor < len(p.items) {
			console := p.items[p.cursor]
			return func() tea.Msg { return SelectConsoleMsg{Console: console} }
		}

	case msg.String() == "up", msg.String() == "k":
		if p.cursor > 0 {
			p.cursor--
		}

	case msg.String() == "down", msg.String() == "j":
		if p.cursor < len(p.items)-1 {
			p.cursor++
		}
	}

	return nil
}

// tabLine pads content to contentW visual width, then appends a separator or space.
func tabLine(content string, contentW int, sep string) string {
	padded := lipgloss.NewStyle().Width(contentW).MaxWidth(contentW).Render(content)
	return padded + sep
}

// ViewBlock renders the console list as vertical tabs with an integrated separator.
// w includes the separator column (contentW + 1). The active tab breaks the
// separator to visually connect to the ROM panel.
func (p *ConsolePanel) ViewBlock(focused bool, w, h int) string {
	contentW := w - 1 // reserve 1 col for separator
	if contentW < 1 {
		contentW = 1
	}

	sepChar := StyleSeparator.Render("│")
	var lines []string

	// Title line
	titleStyle := StylePanelTitle
	if focused {
		titleStyle = StylePanelTitleFocused
	}
	lines = append(lines, tabLine(titleStyle.Render("Consoles"), contentW, sepChar))

	contentH := h - 1
	if contentH < 0 {
		contentH = 0
	}

	if len(p.items) == 0 {
		lines = append(lines, tabLine(StyleHelp.Render(" no consoles"), contentW, sepChar))
	} else {
		visibleLines := contentH
		if visibleLines < 1 {
			visibleLines = 1
		}

		start := 0
		if p.cursor >= visibleLines {
			start = p.cursor - visibleLines + 1
		}
		end := start + visibleLines
		if end > len(p.items) {
			end = len(p.items)
		}

		activeTab := lipgloss.NewStyle().
			Width(contentW).
			MaxWidth(contentW).
			Background(colorDarkGrey)

		for i := start; i < end; i++ {
			c := p.items[i]
			isActive := p.app.selectedConsole != nil && c.Dir == p.app.selectedConsole.Dir
			isCursor := i == p.cursor

			prefix := "  "
			if isCursor {
				prefix = StyleCursor.Render("▸") + " "
			}

			name := c.Dir
			if isActive || isCursor {
				name = StyleSelected.Render(name)
			} else {
				name = StyleInfoValue.Render(name)
			}

			content := prefix + name

			if isActive {
				// Active tab: background highlight, separator breaks open
				lines = append(lines, activeTab.Render(content)+" ")
			} else {
				lines = append(lines, tabLine(content, contentW, sepChar))
			}
		}
	}

	// Fill remaining rows with separator
	for len(lines) < h {
		lines = append(lines, tabLine("", contentW, sepChar))
	}
	if len(lines) > h {
		lines = lines[:h]
	}

	return strings.Join(lines, "\n")
}

func (p *ConsolePanel) SelectedConsole() *config.Console {
	if p.cursor >= 0 && p.cursor < len(p.items) {
		c := p.items[p.cursor]
		return &c
	}
	return nil
}
