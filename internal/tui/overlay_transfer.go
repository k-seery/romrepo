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
	romNames []string

	currentIdx  int // index into romNames
	transferred atomic.Int64
	total       atomic.Int64
	done        bool
	err         error
}

func NewTransferModel(app *App, romNames []string) *TransferModel {
	p := progress.New(progress.WithDefaultGradient())
	return &TransferModel{
		app:      app,
		progress: p,
		romNames: romNames,
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
	romNames := m.romNames
	transferred := &m.transferred
	total := &m.total
	currentIdx := &m.currentIdx

	return func() tea.Msg {
		if app.selectedClient == nil || app.selectedConsole == nil {
			return TransferCompleteMsg{Err: fmt.Errorf("no client or console selected")}
		}

		client := app.resolvePassword(*app.selectedClient)
		console := app.selectedConsole

		sshConn, err := app.connMgr.Get(client)
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

		for i, romName := range romNames {
			*currentIdx = i
			transferred.Store(0)
			total.Store(0)

			progressFn := func(t, tot int64) {
				transferred.Store(t)
				total.Store(tot)
			}

			serverPath := filepath.Join(app.cfg.Server.ROMDir, console.Dir, romName)
			clientPath := filepath.Join(clientDir, romName)

			err = sftpClient.Push(serverPath, clientPath, progressFn)

			if err != nil {
				return TransferCompleteMsg{Err: fmt.Errorf("%s: %w", romName, err)}
			}
		}

		return TransferCompleteMsg{Err: nil}
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

	romCount := len(m.romNames)
	currentName := ""
	idx := m.currentIdx
	if idx < romCount {
		currentName = m.romNames[idx]
	}

	var header string
	if romCount == 1 {
		header = fmt.Sprintf("  Pushing %s...\n\n", currentName)
	} else {
		header = fmt.Sprintf("  Pushing (%d/%d) %s...\n\n", idx+1, romCount, currentName)
	}

	bar := "  " + m.progress.ViewAs(pct) + "\n"
	stats := fmt.Sprintf("  %s / %s", formatSize(cur), formatSize(tot))

	content := header
	if m.done {
		if m.err != nil {
			content += StyleError.Render(fmt.Sprintf("  Error: %v", m.err))
		} else {
			if romCount == 1 {
				content += "  Complete!"
			} else {
				content += fmt.Sprintf("  Complete! %d ROMs transferred.", romCount)
			}
		}
	} else {
		content += bar + stats
	}

	return lipgloss.NewStyle().Width(w).Height(h).MaxHeight(h).Render(content)
}
