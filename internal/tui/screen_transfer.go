package tui

import (
	"fmt"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"

	"romrepo/internal/remote"
)

type tickMsg time.Time

type TransferScreen struct {
	app       *App
	progress  progress.Model
	romName   string
	direction string // "push" or "pull"

	transferred atomic.Int64
	total       atomic.Int64
	done        bool
	err         error
}

func NewTransferScreen(app *App, romName, direction string) *TransferScreen {
	p := progress.New(progress.WithDefaultGradient())
	return &TransferScreen{
		app:       app,
		progress:  p,
		romName:   romName,
		direction: direction,
	}
}

func (s *TransferScreen) Title() string {
	if s.direction == "push" {
		return fmt.Sprintf("Push: %s", s.romName)
	}
	return fmt.Sprintf("Pull: %s", s.romName)
}

func (s *TransferScreen) Init() tea.Cmd {
	return tea.Batch(
		s.doTransfer(),
		s.tickCmd(),
	)
}

func (s *TransferScreen) tickCmd() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (s *TransferScreen) doTransfer() tea.Cmd {
	app := s.app
	romName := s.romName
	direction := s.direction
	transferred := &s.transferred
	total := &s.total

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

func (s *TransferScreen) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case tickMsg:
		if s.done {
			return s, nil
		}
		return s, s.tickCmd()

	case TransferCompleteMsg:
		s.done = true
		m := msg.(TransferCompleteMsg)
		if m.Err != nil {
			s.err = m.Err
			return s, tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
				return GoBackMsg{}
			})
		}
		// Auto-dismiss after a short delay
		return s, tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg {
			return GoBackMsg{}
		})
	}

	return s, nil
}

func (s *TransferScreen) View() string {
	tot := s.total.Load()
	cur := s.transferred.Load()

	var pct float64
	if tot > 0 {
		pct = float64(cur) / float64(tot)
	}

	direction := "Pushing"
	if s.direction == "pull" {
		direction = "Pulling"
	}

	header := fmt.Sprintf("  %s %s...\n\n", direction, s.romName)
	bar := "  " + s.progress.ViewAs(pct) + "\n"
	stats := fmt.Sprintf("  %s / %s", formatSize(cur), formatSize(tot))

	if s.done {
		if s.err != nil {
			return header + StyleError.Render(fmt.Sprintf("  Error: %v", s.err))
		}
		return header + "  Complete!"
	}

	return header + bar + stats
}
