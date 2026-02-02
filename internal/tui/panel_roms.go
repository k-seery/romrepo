package tui

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"romrepo/internal/remote"
	"romrepo/internal/rom"
)

type ROMPanel struct {
	app       *App
	roms      []rom.ROMStatus
	filtered  []rom.ROMStatus
	cursor    int
	filterIdx int // 0=ALL, 1=A, ..., 26=Z
	loading   bool
	width     int
	height    int
}

func NewROMPanel(app *App) ROMPanel {
	return ROMPanel{
		app: app,
	}
}

func (p *ROMPanel) SetSize(w, h int) {
	p.width = w
	p.height = h
}

func (p *ROMPanel) Clear() {
	p.roms = nil
	p.filtered = nil
	p.cursor = 0
	p.filterIdx = 0
	p.loading = false
}

func (p *ROMPanel) applyFilter() {
	if p.filterIdx == 0 {
		p.filtered = p.roms
		return
	}
	letter := rune('A' + p.filterIdx - 1)
	var result []rom.ROMStatus
	for _, r := range p.roms {
		if len(r.Name) > 0 && unicode.ToUpper(rune(r.Name[0])) == letter {
			result = append(result, r)
		}
	}
	p.filtered = result
}

func (p *ROMPanel) LoadROMs() tea.Cmd {
	app := p.app
	p.loading = true
	p.roms = nil
	p.filtered = nil
	p.cursor = 0

	return func() tea.Msg {
		if app.selectedConsole == nil || app.selectedClient == nil {
			return ROMsLoadErrorMsg{Err: fmt.Errorf("no console or client selected")}
		}

		console := *app.selectedConsole
		client := *app.selectedClient

		serverROMs, err := rom.ListServerROMs(app.cfg, console)
		if err != nil {
			return ROMsLoadErrorMsg{Err: fmt.Errorf("listing server ROMs: %w", err)}
		}

		clientFiles := make(map[string]bool)
		var clientErr error

		sshConn, err := app.connMgr.Get(client)
		if err != nil {
			clientErr = fmt.Errorf("SSH: %w", err)
		} else {
			sftpClient, err := remote.NewSFTPClient(sshConn)
			if err != nil {
				clientErr = fmt.Errorf("SFTP: %w", err)
			} else {
				defer sftpClient.Close()

				clientDir := client.ROMDir
				if override, ok := client.ConsoleDirs[console.Dir]; ok {
					clientDir = override
				} else {
					clientDir = filepath.Join(client.ROMDir, console.Dir)
				}

				files, err := sftpClient.ListFiles(clientDir)
				if err != nil {
					clientErr = fmt.Errorf("listing %s: %w", clientDir, err)
				} else {
					for _, f := range files {
						clientFiles[f.Name] = true
					}
				}
			}
		}

		return ROMsLoadedMsg{
			ROMs:      rom.Diff(serverROMs, clientFiles),
			ClientErr: clientErr,
		}
	}
}

func (p *ROMPanel) HandleLoaded(msg ROMsLoadedMsg) tea.Cmd {
	p.loading = false
	sort.Slice(msg.ROMs, func(i, j int) bool {
		if msg.ROMs[i].Location != msg.ROMs[j].Location {
			return msg.ROMs[i].Location == rom.OnBoth
		}
		return msg.ROMs[i].Name < msg.ROMs[j].Name
	})
	p.roms = msg.ROMs
	p.cursor = 0
	p.applyFilter()
	if msg.ClientErr != nil {
		return func() tea.Msg {
			return ErrorMsg{Err: fmt.Errorf("client: %w", msg.ClientErr)}
		}
	}
	return nil
}

func (p *ROMPanel) HandleLoadError(msg ROMsLoadErrorMsg) tea.Cmd {
	p.loading = false
	return func() tea.Msg { return ErrorMsg{Err: msg.Err} }
}

func (p *ROMPanel) Update(msg tea.KeyMsg) tea.Cmd {
	switch {
	case msg.String() == "up", msg.String() == "k":
		if p.cursor > 0 {
			p.cursor--
		}

	case msg.String() == "down", msg.String() == "j":
		if p.cursor < len(p.filtered)-1 {
			p.cursor++
		}

	case msg.String() == "left":
		if p.filterIdx > 0 {
			p.filterIdx--
			p.applyFilter()
			p.cursor = 0
		}

	case msg.String() == "right":
		if p.filterIdx < 26 {
			p.filterIdx++
			p.applyFilter()
			p.cursor = 0
		}

	case key.Matches(msg, p.app.keys.Enter), key.Matches(msg, p.app.keys.Push):
		return p.startPush()

	case key.Matches(msg, p.app.keys.Pull):
		return p.startPull()
	}

	return nil
}

func (p *ROMPanel) startPush() tea.Cmd {
	if p.cursor < 0 || p.cursor >= len(p.filtered) {
		return nil
	}
	romStatus := p.filtered[p.cursor]
	if p.app.selectedClient == nil || p.app.selectedConsole == nil {
		return nil
	}
	return func() tea.Msg {
		return TransferStartMsg{
			ROMName:   romStatus.Name,
			Direction: "push",
		}
	}
}

func (p *ROMPanel) startPull() tea.Cmd {
	if p.cursor < 0 || p.cursor >= len(p.filtered) {
		return nil
	}
	romStatus := p.filtered[p.cursor]
	if romStatus.Location != rom.OnBoth {
		return nil
	}
	return func() tea.Msg {
		return TransferStartMsg{
			ROMName:   romStatus.Name,
			Direction: "pull",
		}
	}
}

func (p *ROMPanel) renderFilterBar(w int) string {
	filters := []string{"ALL"}
	for c := 'A'; c <= 'Z'; c++ {
		filters = append(filters, string(c))
	}

	var parts []string
	for i, f := range filters {
		if i == p.filterIdx {
			parts = append(parts, StyleFilterActive.Render(f))
		} else {
			parts = append(parts, StyleFilterDim.Render(f))
		}
	}

	return lipgloss.NewStyle().Width(w).Render(" " + strings.Join(parts, " "))
}

// ViewBlock renders the ROM list as a fixed-size block without its own border,
// for embedding inside the combined browser panel.
func (p *ROMPanel) ViewBlock(focused bool, w, h int) string {
	var b strings.Builder

	// Filter bar takes first line
	b.WriteString(p.renderFilterBar(w))
	b.WriteString("\n")

	contentH := h - 1
	if contentH < 0 {
		contentH = 0
	}

	if p.loading {
		b.WriteString(StyleHelp.Render(" Loading ROMs..."))
	} else if p.app.selectedClient == nil {
		b.WriteString(StyleHelp.Render(" Select a device"))
	} else if p.app.selectedConsole == nil {
		b.WriteString(StyleHelp.Render(" Select a console"))
	} else if len(p.filtered) == 0 {
		if p.filterIdx > 0 {
			b.WriteString(StyleHelp.Render(" No ROMs for this letter"))
		} else {
			b.WriteString(StyleHelp.Render(" No ROMs found"))
		}
	} else {
		linesPerItem := 2
		visibleItems := contentH / linesPerItem
		if visibleItems < 1 {
			visibleItems = 1
		}

		// Center-pinned scrolling
		half := visibleItems / 2
		start := 0
		if p.cursor > half {
			start = p.cursor - half
		}
		if start+visibleItems > len(p.filtered) {
			start = len(p.filtered) - visibleItems
			if start < 0 {
				start = 0
			}
		}
		end := start + visibleItems
		if end > len(p.filtered) {
			end = len(p.filtered)
		}

		nameW := w - 2
		if nameW < 1 {
			nameW = 1
		}

		for i := start; i < end; i++ {
			r := p.filtered[i]
			isSelected := i == p.cursor

			var style = StyleServerOnly
			switch {
			case isSelected:
				style = StyleSelected
			case r.Location == rom.OnBoth:
				style = StyleOnBoth
			}

			cursor := "  "
			if isSelected {
				cursor = StyleCursor.Render("▸") + " "
			}

			title := style.Render(r.Name)
			size := formatSize(r.ServerSize)
			var status string
			if r.Location == rom.OnBoth {
				status = StyleSyncBadge.Render("● synced")
			} else {
				status = StyleUnsyncBadge.Render("○ server")
			}
			desc := style.Faint(true).Render(size) + "  " + status

			b.WriteString(cursor + wrapWithIndent(title, nameW, 2) + "\n")
			b.WriteString("  " + wrapWithIndent(desc, nameW, 2))
			if i < end-1 {
				b.WriteString("\n")
			}
		}
	}

	return lipgloss.NewStyle().Width(w).Height(h).MaxHeight(h).Render(b.String())
}

// SelectedROM returns the currently highlighted ROM from the filtered list.
func (p *ROMPanel) SelectedROM() *rom.ROMStatus {
	if p.cursor >= 0 && p.cursor < len(p.filtered) {
		r := p.filtered[p.cursor]
		return &r
	}
	return nil
}

func formatSize(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)
	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.1f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.1f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}
