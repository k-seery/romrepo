package tui

import (
	"fmt"
	"strings"

	"romrepo/internal/rom"
)

type MetadataPanel struct {
	app    *App
	width  int
	height int
}

func NewMetadataPanel(app *App) MetadataPanel {
	return MetadataPanel{
		app: app,
	}
}

func (p *MetadataPanel) SetSize(w, h int) {
	p.width = w
	p.height = h
}

func (p *MetadataPanel) View() string {
	innerW := p.width - 2
	if innerW < 0 {
		innerW = 0
	}

	var b strings.Builder

	if p.app.selectedClient != nil {
		b.WriteString(" " + StyleInfoLabel.Render("Device") + "    ")
		b.WriteString(StyleInfoValue.Render(p.app.selectedClient.Name))
		b.WriteString("\n")
		b.WriteString("            ")
		b.WriteString(StyleInfoDim.Render(fmt.Sprintf("%s@%s:%d",
			p.app.selectedClient.User,
			p.app.selectedClient.Host,
			p.app.selectedClient.Port)))
		b.WriteString("\n")
	}

	if p.app.selectedConsole != nil {
		b.WriteString(" " + StyleInfoLabel.Render("Console") + "   ")
		b.WriteString(StyleInfoValue.Render(p.app.selectedConsole.Dir))
		b.WriteString("\n")
	}

	if r := p.app.romPanel.SelectedROM(); r != nil {
		b.WriteString(" " + StyleInfoLabel.Render("ROM") + "       ")
		b.WriteString(StyleInfoValue.Render(r.Name))
		b.WriteString("\n")
		b.WriteString("            ")
		b.WriteString(StyleInfoDim.Render(formatSize(r.ServerSize)))
		if r.Location == rom.OnBoth {
			b.WriteString("  " + StyleSyncBadge.Render("synced"))
		} else {
			b.WriteString("  " + StyleUnsyncBadge.Render("not synced"))
		}
	}

	if p.app.selectedClient == nil && p.app.selectedConsole == nil {
		b.WriteString("\n")
		b.WriteString(StyleHelp.Render(" Select a device and console"))
		b.WriteString("\n")
		b.WriteString(StyleHelp.Render(" to browse ROMs"))
	}

	return StylePanelUnfocused.
		Width(innerW).
		Height(p.height).
		Render(b.String())
}
