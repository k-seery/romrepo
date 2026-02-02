package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"

	"romrepo/internal/config"
	"romrepo/internal/network"
)

type ScanPanel struct {
	app      *App
	devices  []network.Device
	cursor   int
	scanning bool
	cancel   context.CancelFunc
	spinner  spinner.Model
	width    int
	height   int
}

func NewScanPanel(app *App) ScanPanel {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	return ScanPanel{
		app:     app,
		spinner: sp,
	}
}

func (p *ScanPanel) SetSize(w, h int) {
	p.width = w
	p.height = h
}

func (p *ScanPanel) StartScan() {
	p.scanning = true
	p.devices = nil
	p.cursor = 0
}

func (p *ScanPanel) Init() tea.Cmd {
	return tea.Batch(p.spinner.Tick, p.doScan())
}

func (p *ScanPanel) doScan() tea.Cmd {
	return func() tea.Msg {
		subnet, err := network.LocalSubnet()
		if err != nil {
			return ScanResultMsg{Err: err}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		p.cancel = cancel

		devices, err := network.ScanSubnet(ctx, subnet)
		cancel()
		return ScanResultMsg{Devices: devices, Err: err}
	}
}

func (p *ScanPanel) HandleScanResult(msg ScanResultMsg) tea.Cmd {
	p.scanning = false
	if msg.Err != nil {
		return func() tea.Msg { return ErrorMsg{Err: msg.Err} }
	}
	p.devices = msg.Devices
	p.cursor = 0
	return nil
}

func (p *ScanPanel) Update(msg tea.KeyMsg) tea.Cmd {
	if p.scanning {
		if key.Matches(msg, p.app.keys.Escape) {
			if p.cancel != nil {
				p.cancel()
				p.cancel = nil
			}
			p.scanning = false
		}
		return nil
	}

	switch {
	case msg.String() == "up", msg.String() == "k":
		if p.cursor > 0 {
			p.cursor--
		}

	case msg.String() == "down", msg.String() == "j":
		if p.cursor < len(p.devices)-1 {
			p.cursor++
		}

	case key.Matches(msg, p.app.keys.Enter):
		if p.cursor >= 0 && p.cursor < len(p.devices) {
			dev := p.devices[p.cursor]
			hostname := dev.Hostname
			if hostname == "" {
				hostname = dev.IP.String()
			}
			c := &config.Client{
				Name: hostname,
				Host: dev.IP.String(),
				Port: 22,
			}
			p.app.mode = ModeEditing
			p.app.overlay = NewEditFormModel(p.app, c, -1)
			return p.app.overlay.Init()
		}

	}

	return nil
}

func (p *ScanPanel) UpdateSpinner(msg spinner.TickMsg) tea.Cmd {
	if !p.scanning {
		return nil
	}
	var cmd tea.Cmd
	p.spinner, cmd = p.spinner.Update(msg)
	return cmd
}

func (p *ScanPanel) View(focused bool) string {
	contentH := p.height - 1
	if contentH < 0 {
		contentH = 0
	}

	var b strings.Builder

	if p.scanning {
		b.WriteString(fmt.Sprintf(" %s Scanning network...", p.spinner.View()))
	} else if len(p.devices) == 0 {
		b.WriteString(StyleHelp.Render(" Press s to scan network"))
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
		if end > len(p.devices) {
			end = len(p.devices)
		}

		for i := start; i < end; i++ {
			dev := p.devices[i]
			cursor := "  "
			if i == p.cursor {
				cursor = StyleCursor.Render("▸") + " "
			}

			label := dev.IP.String()
			if dev.Hostname != "" {
				label = dev.Hostname
			}

			if i == p.cursor {
				label = StyleSelected.Render(label)
			} else if dev.SSHOpen {
				label = StyleSSHOpen.Render(label)
			} else {
				label = StyleInfoValue.Render(label)
			}

			b.WriteString(cursor + label)
			if i < end-1 {
				b.WriteString("\n")
			}
		}
	}

	return renderPanel(b.String(), "Scan", focused, p.width, p.height)
}
