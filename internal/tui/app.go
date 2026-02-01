package tui

import (
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"romrepo/internal/config"
	"romrepo/internal/remote"
)

type Screen interface {
	tea.Model
	Title() string
}

type App struct {
	stack    []Screen
	cfg      *config.Config
	cfgPath  string
	connMgr  *remote.ConnManager
	keys     KeyMap
	help     help.Model
	showHelp bool
	width    int
	height   int

	errMsg   string
	errTimer *time.Timer

	// Current context
	selectedClient  *config.Client
	selectedConsole *config.Console
}

func NewApp(cfg *config.Config, connMgr *remote.ConnManager) *App {
	h := help.New()
	h.ShowAll = false

	app := &App{
		cfg:     cfg,
		connMgr: connMgr,
		keys:    DefaultKeyMap(),
		help:    h,
		width:   80,
		height:  24,
	}

	clientScreen := NewClientScreen(cfg, app)
	app.stack = []Screen{clientScreen}
	return app
}

func (a *App) Init() tea.Cmd {
	if len(a.stack) > 0 {
		return a.stack[len(a.stack)-1].Init()
	}
	return nil
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.help.Width = msg.Width
		// Forward to current screen
		if len(a.stack) > 0 {
			top := a.stack[len(a.stack)-1]
			updated, cmd := top.Update(msg)
			a.stack[len(a.stack)-1] = updated.(Screen)
			return a, cmd
		}
		return a, nil

	case tea.KeyMsg:
		// Global keys
		switch {
		case msg.String() == "ctrl+c":
			return a, tea.Quit
		case msg.String() == "q" && !a.isFiltering():
			if len(a.stack) <= 1 {
				return a, tea.Quit
			}
		case msg.String() == "?":
			a.showHelp = !a.showHelp
			return a, nil
		}

	case GoBackMsg:
		return a, a.popScreen()

	case SelectClientMsg:
		a.selectedClient = &msg.Client
		screen := NewConsoleScreen(a.cfg, a)
		return a, a.pushScreen(screen)

	case SelectConsoleMsg:
		a.selectedConsole = &msg.Console
		screen := NewROMScreen(a.cfg, a)
		return a, a.pushScreen(screen)

	case OpenManageMsg:
		screen := NewManageScreen(a.cfg, a)
		return a, a.pushScreen(screen)

	case ConfigUpdatedMsg:
		a.cfg = msg.Config
		// Rebuild client screen
		if len(a.stack) > 0 {
			a.stack[0] = NewClientScreen(a.cfg, a)
		}
		return a, nil

	case TransferStartMsg:
		screen := NewTransferScreen(a, msg.ROMName, msg.Direction)
		return a, a.pushScreen(screen)

	case ErrorMsg:
		a.setError(msg.Err.Error())
		return a, a.clearErrorAfter(5 * time.Second)

	case ClearErrorMsg:
		a.errMsg = ""
		return a, nil
	}

	// Forward to current screen
	if len(a.stack) > 0 {
		top := a.stack[len(a.stack)-1]
		updated, cmd := top.Update(msg)
		a.stack[len(a.stack)-1] = updated.(Screen)
		return a, cmd
	}
	return a, nil
}

func (a *App) View() string {
	if a.width == 0 {
		return ""
	}

	var sections []string

	// Breadcrumb header
	breadcrumb := a.buildBreadcrumb()
	sections = append(sections, StyleTitle.Width(a.width).Render(breadcrumb))

	// Main content area
	contentHeight := a.height - 3 // title + status bar + help
	if a.showHelp {
		contentHeight -= 4
	}

	if len(a.stack) > 0 {
		content := a.stack[len(a.stack)-1].View()
		sections = append(sections, lipgloss.NewStyle().
			Height(contentHeight).
			MaxHeight(contentHeight).
			Width(a.width).
			Render(content))
	}

	// Help overlay
	if a.showHelp {
		a.help.ShowAll = true
		helpView := StyleHelp.Render(a.help.View(a.keys))
		sections = append(sections, helpView)
	}

	// Status bar
	status := a.buildStatusBar()
	sections = append(sections, status)

	return strings.Join(sections, "\n")
}

func (a *App) pushScreen(s Screen) tea.Cmd {
	a.stack = append(a.stack, s)
	return s.Init()
}

func (a *App) popScreen() tea.Cmd {
	if len(a.stack) <= 1 {
		return tea.Quit
	}
	a.stack = a.stack[:len(a.stack)-1]

	// Clear context based on stack depth
	switch len(a.stack) {
	case 1:
		a.selectedClient = nil
		a.selectedConsole = nil
	case 2:
		a.selectedConsole = nil
	}

	return nil
}

func (a *App) buildBreadcrumb() string {
	parts := []string{"romrepo"}
	for _, s := range a.stack {
		parts = append(parts, s.Title())
	}
	return strings.Join(parts, " > ")
}

func (a *App) buildStatusBar() string {
	if a.errMsg != "" {
		return StyleError.Width(a.width).Render(a.errMsg)
	}

	hints := []string{"?:help"}
	if len(a.stack) > 1 {
		hints = append(hints, "esc:back")
	}
	hints = append(hints, "q:quit")
	return StyleStatusBar.Width(a.width).Render(strings.Join(hints, "  "))
}

func (a *App) setError(msg string) {
	a.errMsg = msg
	if a.errTimer != nil {
		a.errTimer.Stop()
	}
}

func (a *App) clearErrorAfter(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg {
		return ClearErrorMsg{}
	})
}

func (a *App) isFiltering() bool {
	// Check if current screen has an active filter
	return false
}
