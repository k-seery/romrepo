package tui

import (
	"fmt"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"romrepo/internal/config"
	"romrepo/internal/remote"
)

type DirBrowser struct {
	app        *App
	entries    []remote.FileInfo
	cursor     int
	path       string
	loading    bool
	err        error
	sftpClient *remote.SFTPClient
	connected  bool
}

func NewDirBrowser(app *App) *DirBrowser {
	return &DirBrowser{app: app}
}

func (b *DirBrowser) Close() {
	if b.sftpClient != nil {
		b.sftpClient.Close()
		b.sftpClient = nil
	}
}

func (b *DirBrowser) Connect(c config.Client) tea.Cmd {
	b.loading = true
	b.err = nil
	b.connected = false
	connMgr := b.app.connMgr

	return func() tea.Msg {
		sshConn, err := connMgr.Get(c)
		if err != nil {
			return DirConnectErrorMsg{Err: err}
		}
		sftpClient, err := remote.NewSFTPClient(sshConn)
		if err != nil {
			return DirConnectErrorMsg{Err: err}
		}
		home, err := sftpClient.HomePath()
		if err != nil {
			home = "/"
		}
		return DirConnectedMsg{SFTPClient: sftpClient, HomePath: home}
	}
}

func (b *DirBrowser) NavigateTo(path string) tea.Cmd {
	b.loading = true
	sftpClient := b.sftpClient

	return func() tea.Msg {
		entries, err := sftpClient.ListDir(path)
		return DirListedMsg{Path: path, Entries: entries, Err: err}
	}
}

func (b *DirBrowser) HandleConnected(msg DirConnectedMsg) tea.Cmd {
	b.sftpClient = msg.SFTPClient
	b.connected = true
	b.loading = false
	b.err = nil

	startPath := msg.HomePath
	return b.NavigateTo(startPath)
}

func (b *DirBrowser) HandleConnectedWithPath(msg DirConnectedMsg, romDir string) tea.Cmd {
	b.sftpClient = msg.SFTPClient
	b.connected = true
	b.loading = false
	b.err = nil

	startPath := msg.HomePath
	if romDir != "" {
		startPath = romDir
	}
	return b.NavigateTo(startPath)
}

func (b *DirBrowser) HandleListed(msg DirListedMsg) {
	b.loading = false
	if msg.Err != nil {
		b.err = msg.Err
		return
	}
	b.path = msg.Path
	b.entries = msg.Entries
	b.cursor = 0
	b.err = nil
}

func (b *DirBrowser) HandleConnectError(msg DirConnectErrorMsg) {
	b.loading = false
	b.err = msg.Err
}

// Update handles key input when the browser has focus.
// Returns (cmd, dirWasSelected). If dirWasSelected is true,
// the selected path is in b.path.
func (b *DirBrowser) Update(msg tea.KeyMsg) (tea.Cmd, bool) {
	if !b.connected || b.loading {
		return nil, false
	}

	// Total visible entries: [use this dir], optionally [../], then entries
	totalItems := b.visibleCount()

	switch msg.String() {
	case "up", "k":
		if b.cursor > 0 {
			b.cursor--
		}
		return nil, false

	case "down", "j":
		if b.cursor < totalItems-1 {
			b.cursor++
		}
		return nil, false

	case "enter":
		// cursor 0 = "use this dir"
		if b.cursor == 0 {
			return nil, true
		}
		idx := b.cursor - 1
		// If not at root, cursor 1 = "../"
		if b.path != "/" {
			if idx == 0 {
				parent := filepath.Dir(b.path)
				return b.NavigateTo(parent), false
			}
			idx-- // adjust for ../ entry
		}
		if idx >= 0 && idx < len(b.entries) {
			entry := b.entries[idx]
			if entry.IsDir {
				target := filepath.Join(b.path, entry.Name)
				return b.NavigateTo(target), false
			}
		}
		return nil, false

	case "backspace", "left":
		if b.path != "/" {
			parent := filepath.Dir(b.path)
			return b.NavigateTo(parent), false
		}
		return nil, false
	}

	return nil, false
}

func (b *DirBrowser) visibleCount() int {
	count := 1 // "use this dir"
	if b.path != "/" {
		count++ // "../"
	}
	count += len(b.entries)
	return count
}

func (b *DirBrowser) SelectedPath() string {
	return b.path
}

func (b *DirBrowser) View(w, h int) string {
	var sb strings.Builder

	titleStyle := StylePanelTitleFocused
	sb.WriteString(titleStyle.Render(truncatePath(b.path, w-2)))
	sb.WriteString("\n")

	if !b.connected && !b.loading && b.err == nil {
		msg := lipgloss.NewStyle().Foreground(colorDimGrey).Render("ctrl+t to connect")
		sb.WriteString("\n " + msg)
		return lipgloss.NewStyle().Width(w).Height(h).MaxHeight(h).Render(sb.String())
	}

	if b.loading {
		msg := lipgloss.NewStyle().Foreground(colorCyan).Render("Connecting...")
		sb.WriteString("\n " + msg)
		return lipgloss.NewStyle().Width(w).Height(h).MaxHeight(h).Render(sb.String())
	}

	if b.err != nil {
		errStr := fmt.Sprintf("Error: %s", b.err)
		if len(errStr) > w-2 {
			errStr = errStr[:w-2]
		}
		msg := lipgloss.NewStyle().Foreground(colorRed).Render(errStr)
		sb.WriteString("\n " + msg)
		sb.WriteString("\n\n " + lipgloss.NewStyle().Foreground(colorDimGrey).Render("ctrl+t to retry"))
		return lipgloss.NewStyle().Width(w).Height(h).MaxHeight(h).Render(sb.String())
	}

	// Scrollable list area (h minus title line)
	listH := h - 2
	if listH < 1 {
		listH = 1
	}

	type entry struct {
		label string
		dim   bool
	}

	var items []entry
	items = append(items, entry{label: "[ use this dir ]", dim: false})
	if b.path != "/" {
		items = append(items, entry{label: "../", dim: false})
	}
	for _, e := range b.entries {
		if e.IsDir {
			items = append(items, entry{label: e.Name + "/", dim: false})
		} else {
			items = append(items, entry{label: e.Name, dim: true})
		}
	}

	// Calculate scroll offset
	start := 0
	if b.cursor >= listH {
		start = b.cursor - listH + 1
	}
	end := start + listH
	if end > len(items) {
		end = len(items)
	}

	for i := start; i < end; i++ {
		item := items[i]
		prefix := "  "
		if i == b.cursor {
			prefix = StyleCursor.Render("▸ ")
		}
		label := item.label
		maxLabelW := w - 3
		if len(label) > maxLabelW {
			label = label[:maxLabelW]
		}
		if item.dim {
			label = lipgloss.NewStyle().Foreground(colorDimGrey).Render(label)
		}
		sb.WriteString(prefix + label + "\n")
	}

	return lipgloss.NewStyle().Width(w).Height(h).MaxHeight(h).Render(sb.String())
}

func truncatePath(p string, maxW int) string {
	if p == "" {
		p = "/"
	}
	if len(p) <= maxW {
		return p
	}
	// Show as much of the end as possible
	return "..." + p[len(p)-maxW+3:]
}
