package tui

import (
	"fmt"
	"io"
	"path/filepath"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"romrepo/internal/config"
	"romrepo/internal/remote"
	"romrepo/internal/rom"
)

type romItem struct {
	status rom.ROMStatus
}

func (i romItem) Title() string       { return i.status.Name }
func (i romItem) FilterValue() string { return i.status.Name }
func (i romItem) Description() string {
	size := formatSize(i.status.ServerSize)
	switch i.status.Location {
	case rom.OnBoth:
		return fmt.Sprintf("%s  [synced]", size)
	default:
		return fmt.Sprintf("%s  [server only]", size)
	}
}

type romDelegate struct{}

func (d romDelegate) Height() int                             { return 2 }
func (d romDelegate) Spacing() int                            { return 0 }
func (d romDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d romDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	item, ok := listItem.(romItem)
	if !ok {
		return
	}

	isSelected := index == m.Index()

	var style lipgloss.Style
	switch {
	case isSelected:
		style = StyleSelected
	case item.status.Location == rom.OnBoth:
		style = StyleOnBoth
	default:
		style = StyleServerOnly
	}

	cursor := "  "
	if isSelected {
		cursor = "> "
	}

	title := style.Render(item.status.Name)
	desc := style.Copy().Faint(true).Render(item.Description())

	fmt.Fprintf(w, "%s%s\n%s%s", cursor, title, "  ", desc)
}

type ROMScreen struct {
	list    list.Model
	app     *App
	loading bool
	roms    []rom.ROMStatus
}

func NewROMScreen(cfg *config.Config, app *App) *ROMScreen {
	delegate := romDelegate{}
	l := list.New(nil, delegate, app.width, app.height-4)
	l.Title = "ROMs"
	l.SetShowTitle(false)
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.SetShowHelp(false)

	return &ROMScreen{
		list:    l,
		app:     app,
		loading: true,
	}
}

func (s *ROMScreen) Title() string {
	if s.app.selectedConsole != nil {
		return s.app.selectedConsole.Name
	}
	return "ROMs"
}

func (s *ROMScreen) Init() tea.Cmd {
	return s.loadROMs()
}

func (s *ROMScreen) loadROMs() tea.Cmd {
	app := s.app
	return func() tea.Msg {
		if app.selectedConsole == nil || app.selectedClient == nil {
			return ROMsLoadErrorMsg{Err: fmt.Errorf("no console or client selected")}
		}

		console := *app.selectedConsole
		client := *app.selectedClient

		// Load server ROMs from local filesystem
		serverROMs, err := rom.ListServerROMs(app.cfg, console)
		if err != nil {
			return ROMsLoadErrorMsg{Err: fmt.Errorf("listing server ROMs: %w", err)}
		}

		// Load client ROMs via SFTP
		clientFiles := make(map[string]bool)
		sshConn, err := app.connMgr.Get(client)
		if err != nil {
			// If we can't connect, show server-only list
			return ROMsLoadedMsg{ROMs: rom.Diff(serverROMs, clientFiles)}
		}

		sftpClient, err := remote.NewSFTPClient(sshConn)
		if err != nil {
			return ROMsLoadedMsg{ROMs: rom.Diff(serverROMs, clientFiles)}
		}
		defer sftpClient.Close()

		clientDir := client.ROMDir
		if override, ok := client.ConsoleDirs[console.Dir]; ok {
			clientDir = override
		} else {
			clientDir = filepath.Join(client.ROMDir, console.Dir)
		}

		files, err := sftpClient.ListFiles(clientDir)
		if err == nil {
			for _, f := range files {
				clientFiles[f.Name] = true
			}
		}

		return ROMsLoadedMsg{ROMs: rom.Diff(serverROMs, clientFiles)}
	}
}

func (s *ROMScreen) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		s.list.SetSize(msg.Width, msg.Height-4)
		return s, nil

	case ROMsLoadedMsg:
		s.loading = false
		s.roms = msg.ROMs
		items := make([]list.Item, len(msg.ROMs))
		for i, r := range msg.ROMs {
			items[i] = romItem{status: r}
		}
		s.list.SetItems(items)
		return s, nil

	case ROMsLoadErrorMsg:
		s.loading = false
		return s, func() tea.Msg { return ErrorMsg{Err: msg.Err} }

	case tea.KeyMsg:
		if s.list.FilterState() == list.Filtering {
			break
		}

		switch {
		case key.Matches(msg, s.app.keys.Back):
			return s, func() tea.Msg { return GoBackMsg{} }

		case key.Matches(msg, s.app.keys.Enter), key.Matches(msg, s.app.keys.Push):
			return s, s.startPush()

		case key.Matches(msg, s.app.keys.Pull):
			return s, s.startPull()
		}
	}

	var cmd tea.Cmd
	s.list, cmd = s.list.Update(msg)
	return s, cmd
}

func (s *ROMScreen) startPush() tea.Cmd {
	item, ok := s.list.SelectedItem().(romItem)
	if !ok {
		return nil
	}

	romStatus := item.status
	client := s.app.selectedClient
	console := s.app.selectedConsole
	connMgr := s.app.connMgr

	if client == nil || console == nil {
		return nil
	}

	_ = romStatus
	_ = connMgr

	return func() tea.Msg {
		return TransferStartMsg{
			ROMName:   romStatus.Name,
			Direction: "push",
		}
	}
}

func (s *ROMScreen) startPull() tea.Cmd {
	item, ok := s.list.SelectedItem().(romItem)
	if !ok || item.status.Location != rom.OnBoth {
		return nil
	}

	return func() tea.Msg {
		return TransferStartMsg{
			ROMName:   item.status.Name,
			Direction: "pull",
		}
	}
}

func (s *ROMScreen) View() string {
	if s.loading {
		return "  Loading ROMs..."
	}
	return s.list.View()
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
