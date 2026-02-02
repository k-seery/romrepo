package tui

import (
	"fmt"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"romrepo/internal/remote"
)

type transferTickMsg time.Time

type TransferModel struct {
	app       *App
	progress  progress.Model
	romName   string
	direction string // "push" or "pull"

	transferred atomic.Int64
	total       atomic.Int64
	done        bool
	err         error
}

func NewTransferModel(app *App, romName, direction string) *TransferModel {
	p := progress.New(progress.WithDefaultGradient())
	return &TransferModel{
		app:       app,
		progress:  p,
		romName:   romName,
		direction: direction,
	}
}

func (m *TransferModel) Init() tea.Cmd {
	return tea.Batch(
		m.doTransfer(),
		m.tickCmd(),
	)
}

func (m *TransferModel) tickCmd() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return transferTickMsg(t)
	})
}

func (m *TransferModel) doTransfer() tea.Cmd {
	app := m.app
	romName := m.romName
	direction := m.direction
	transferred := &m.transferred
	total := &m.total

	return func() tea.Msg {
		client := app.selectedClient
		console := app.selectedConsole
		if client == nil || console == nil {
			return TransferCompleteMsg{Err: fmt.Errorf("no client or console selected")}
		}

		sshConn, err := app.connMgr.Get(*client)
		if err != nil {
			return TransferCompleteMsg{Err: err}
		}

		sftpClient, err := remote.NewSFTPClient(sshConn)
		if err != nil {
			return TransferCompleteMsg{Err: err}
		}
		defer sftpClient.Close()

		clientDir := client.ROMDir
		if override, ok := client.ConsoleDirs[console.Dir]; ok {
			clientDir = override
		} else {
			clientDir = filepath.Join(client.ROMDir, console.Dir)
		}

		progressFn := func(t, tot int64) {
			transferred.Store(t)
			total.Store(tot)
		}

		serverPath := filepath.Join(app.cfg.Server.ROMDir, console.Dir, romName)
		clientPath := filepath.Join(clientDir, romName)

		switch direction {
		case "push":
			err = sftpClient.Push(serverPath, clientPath, progressFn)
		case "pull":
			err = sftpClient.Pull(clientPath, serverPath, progressFn)
		}

		return TransferCompleteMsg{Err: err}
	}
}

func (m *TransferModel) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case transferTickMsg:
		if m.done {
			return nil
		}
		return m.tickCmd()

	case TransferCompleteMsg:
		m.done = true
		if msg.Err != nil {
			m.err = msg.Err
			return tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
				return CancelOverlayMsg{}
			})
		}
		return tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg {
			return CancelOverlayMsg{}
		})
	}

	return nil
}

func (m *TransferModel) View(w, h int) string {
	tot := m.total.Load()
	cur := m.transferred.Load()

	var pct float64
	if tot > 0 {
		pct = float64(cur) / float64(tot)
	}

	direction := "Pushing"
	if m.direction == "pull" {
		direction = "Pulling"
	}

	header := fmt.Sprintf("  %s %s...\n\n", direction, m.romName)
	bar := "  " + m.progress.ViewAs(pct) + "\n"
	stats := fmt.Sprintf("  %s / %s", formatSize(cur), formatSize(tot))

	content := header
	if m.done {
		if m.err != nil {
			content += StyleError.Render(fmt.Sprintf("  Error: %v", m.err))
		} else {
			content += "  Complete!"
		}
	} else {
		content += bar + stats
	}

	return lipgloss.NewStyle().Width(w).Height(h).MaxHeight(h).Render(content)
}
