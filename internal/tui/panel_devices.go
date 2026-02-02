package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"romrepo/internal/config"
)

type DevicePanel struct {
	app      *App
	items    []config.Client
	cursor   int
	width    int
	height   int
}

func NewDevicePanel(app *App) DevicePanel {
	return DevicePanel{
		app:   app,
		items: app.cfg.Clients,
	}
}

func (p *DevicePanel) SetSize(w, h int) {
	p.width = w
	p.height = h
}

func (p *DevicePanel) Rebuild(cfg *config.Config) {
	p.items = cfg.Clients
	if p.cursor >= len(p.items) {
		p.cursor = max(0, len(p.items)-1)
	}
}

func (p *DevicePanel) Update(msg tea.KeyMsg) tea.Cmd {
	switch {
	case key.Matches(msg, p.app.keys.Enter):
		if p.cursor >= 0 && p.cursor < len(p.items) {
			client := p.items[p.cursor]
			return func() tea.Msg { return SelectClientMsg{Client: client} }
		}

	case msg.String() == "up", msg.String() == "k":
		if p.cursor > 0 {
			p.cursor--
		}

	case msg.String() == "down", msg.String() == "j":
		if p.cursor < len(p.items)-1 {
			p.cursor++
		}

	case key.Matches(msg, p.app.keys.Add):
		p.app.mode = ModeEditing
		p.app.overlay = NewEditFormModel(p.app, nil, -1)
		return p.app.overlay.Init()

	case key.Matches(msg, p.app.keys.Edit):
		if p.cursor >= 0 && p.cursor < len(p.items) {
			p.app.mode = ModeEditing
			p.app.overlay = NewEditFormModel(p.app, &p.items[p.cursor], p.cursor)
			return p.app.overlay.Init()
		}

	case key.Matches(msg, p.app.keys.Delete):
		if p.cursor >= 0 && p.cursor < len(p.items) {
			p.app.cfg.Clients = append(p.app.cfg.Clients[:p.cursor], p.app.cfg.Clients[p.cursor+1:]...)
			if err := config.Save(p.app.cfg, p.app.cfgPath); err != nil {
				return func() tea.Msg { return ErrorMsg{Err: err} }
			}
			p.Rebuild(p.app.cfg)
			return func() tea.Msg { return ConfigUpdatedMsg{Config: p.app.cfg} }
		}

	case key.Matches(msg, p.app.keys.Settings):
		p.app.mode = ModeSettings
		p.app.overlay = NewSettingsModel(p.app)
		return p.app.overlay.Init()
	}

	return nil
}

func (p *DevicePanel) View(focused bool) string {
	// Content area height = panel height minus title line
	contentH := p.height - 1
	if contentH < 0 {
		contentH = 0
	}

	var b strings.Builder

	title := "Devices"
	titleStyle := StylePanelTitle
	if focused {
		titleStyle = StylePanelTitleFocused
	}
	b.WriteString(titleStyle.Render(title))
	b.WriteString("\n")

	if len(p.items) == 0 {
		b.WriteString(StyleHelp.Render(" no devices — press a to add"))
	} else {
		// Scrolling window
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

		for i := start; i < end; i++ {
			c := p.items[i]
			cursor := "  "
			if i == p.cursor {
				cursor = StyleCursor.Render("▸") + " "
			}

			name := c.Name
			if p.app.selectedClient != nil && c.Name == p.app.selectedClient.Name {
				name = StyleSelected.Render(name)
			} else if i == p.cursor {
				name = StyleSelected.Render(name)
			} else {
				name = StyleInfoValue.Render(name)
			}

			b.WriteString(cursor + name)
			if i < end-1 {
				b.WriteString("\n")
			}
		}
	}

	content := b.String()
	innerW := p.width - 2 // account for border
	if innerW < 0 {
		innerW = 0
	}

	panelStyle := StylePanelUnfocused
	if focused {
		panelStyle = StylePanelFocused
	}

	return panelStyle.
		Width(innerW).
		Height(p.height).
		Render(content)
}

func (p *DevicePanel) SelectedClient() *config.Client {
	if p.cursor >= 0 && p.cursor < len(p.items) {
		c := p.items[p.cursor]
		return &c
	}
	return nil
}

// wrapWithIndent renders text constrained to width, with continuation lines
// indented by indent spaces to align with the first line's content after a cursor prefix.
func wrapWithIndent(text string, width, indent int) string {
	if width <= 0 {
		return text
	}
	wrapped := lipgloss.NewStyle().Width(width).Render(text)
	lines := strings.Split(wrapped, "\n")
	pad := strings.Repeat(" ", indent)
	for i := 1; i < len(lines); i++ {
		lines[i] = pad + lines[i]
	}
	return strings.Join(lines, "\n")
}

func renderPanel(content string, title string, focused bool, w, h int) string {
	titleStyle := StylePanelTitle
	if focused {
		titleStyle = StylePanelTitleFocused
	}

	header := titleStyle.Render(title)

	full := header + "\n" + content

	innerW := w - 2
	if innerW < 0 {
		innerW = 0
	}

	panelStyle := StylePanelUnfocused
	if focused {
		panelStyle = StylePanelFocused
	}

	return panelStyle.
		Width(innerW).
		Height(h).
		Render(full)
}

