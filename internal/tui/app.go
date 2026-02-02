package tui

import (
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"romrepo/internal/config"
	"romrepo/internal/remote"
)

// PanelID identifies which panel has focus.
type PanelID int

const (
	PanelDevices PanelID = iota
	PanelScan
	PanelConsoles
	PanelROMs
	panelCount // sentinel for cycling
)

// AppMode tracks the current interaction mode.
type AppMode int

const (
	ModeNormal AppMode = iota
	ModeEditing
	ModeTransfer
	ModeSettings
	ModePassword
)

const (
	pendingNone = iota
	pendingLoadROMs
	pendingTransfer
)

const banner = "" +
	"╔══════════════════════════════════════════════════════════════════╗\n" +
	"║                                                                  ║\n" +
	"║  ██████╗  ██████╗ ███╗   ███╗   ██████╗ ███████╗██████╗  ██████╗ ║\n" +
	"║  ██╔══██╗██╔═══██╗████╗ ████║   ██╔══██╗██╔════╝██╔══██╗██╔═══██╗║\n" +
	"║  ██████╔╝██║   ██║██╔████╔██║   ██████╔╝█████╗  ██████╔╝██║   ██║║\n" +
	"║  ██╔══██╗██║   ██║██║╚██╔╝██║   ██╔══██╗██╔══╝  ██╔═══╝ ██║   ██║║\n" +
	"║  ██║  ██║╚██████╔╝██║ ╚═╝ ██║   ██║  ██║███████╗██║     ╚██████╔╝║\n" +
	"║  ╚═╝  ╚═╝ ╚═════╝ ╚═╝     ╚═╝   ╚═╝  ╚═╝╚══════╝╚═╝      ╚═════╝ ║\n" +
	"║                                                                  ║\n" +
	"║                 ░▒▓█  R  O  M   R  E  P  O  █▓▒░                 ║\n" +
	"║                                                                  ║\n" +
	"╚══════════════════════════════════════════════════════════════════╝"

const bannerH = 12
const bannerW = 68

// Overlay is any model that replaces the ROM panel area.
type Overlay interface {
	Init() tea.Cmd
	Update(tea.Msg) tea.Cmd
	View(w, h int) string
}

type App struct {
	cfg     *config.Config
	cfgPath string
	connMgr *remote.ConnManager
	keys    KeyMap
	help    help.Model

	focus    PanelID
	mode     AppMode
	overlay  Overlay
	showHelp bool
	width    int
	height   int

	devicePanel   DevicePanel
	scanPanel     ScanPanel
	consolePanel  ConsolePanel
	romPanel      ROMPanel
	metadataPanel MetadataPanel

	errMsg   string
	errTimer *time.Timer

	selectedClient  *config.Client
	selectedConsole *config.Console

	passwords     map[string]string
	pendingAction struct {
		kind     int
		romNames []string
	}
}

func NewApp(cfg *config.Config, connMgr *remote.ConnManager, cfgPath string) *App {
	h := help.New()
	h.ShowAll = false

	app := &App{
		cfg:       cfg,
		cfgPath:   cfgPath,
		connMgr:   connMgr,
		keys:      DefaultKeyMap(),
		help:      h,
		width:     80,
		height:    24,
		focus:     PanelDevices,
		mode:      ModeNormal,
		passwords: make(map[string]string),
	}

	app.devicePanel = NewDevicePanel(app)
	app.scanPanel = NewScanPanel(app)
	app.consolePanel = NewConsolePanel(app)
	app.romPanel = NewROMPanel(app)
	app.metadataPanel = NewMetadataPanel(app)

	return app
}

func (a *App) Init() tea.Cmd {
	return nil
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.help.Width = msg.Width
		return a, nil

	case tea.KeyMsg:
		// ctrl+c always quits
		if msg.String() == "ctrl+c" {
			return a, tea.Quit
		}

		// In overlay mode, route all keys to overlay
		if a.mode != ModeNormal {
			if a.overlay != nil {
				cmd := a.overlay.Update(msg)
				return a, cmd
			}
			return a, nil
		}

		// Global keys in normal mode
		switch {
		case msg.String() == "?":
			a.showHelp = !a.showHelp
			return a, nil

		case msg.String() == "q":
			return a, tea.Quit

		case msg.String() == "tab":
			a.focus = (a.focus + 1) % panelCount
			return a, nil

		case msg.String() == "shift+tab":
			a.focus = (a.focus - 1 + panelCount) % panelCount
			return a, nil

		case msg.String() == "s":
			a.scanPanel.StartScan()
			return a, a.scanPanel.Init()
		}

		// Route to focused panel
		var cmd tea.Cmd
		switch a.focus {
		case PanelDevices:
			cmd = a.devicePanel.Update(msg)
		case PanelScan:
			cmd = a.scanPanel.Update(msg)
		case PanelConsoles:
			cmd = a.consolePanel.Update(msg)
		case PanelROMs:
			cmd = a.romPanel.Update(msg)
		}
		return a, cmd

	case SelectClientMsg:
		a.selectedClient = &msg.Client
		a.selectedConsole = nil
		a.romPanel.Clear()
		a.focus = PanelConsoles
		return a, nil

	case SelectConsoleMsg:
		a.selectedConsole = &msg.Console
		a.focus = PanelROMs
		if a.selectedClient != nil && a.needsPassword(a.selectedClient) {
			a.pendingAction.kind = pendingLoadROMs
			a.mode = ModePassword
			a.overlay = NewPasswordModel(a, a.selectedClient.Name, a.selectedClient.Host, a.selectedClient.User)
			return a, a.overlay.Init()
		}
		return a, a.romPanel.LoadROMs()

	case ROMsLoadedMsg:
		cmd := a.romPanel.HandleLoaded(msg)
		return a, cmd

	case ROMsLoadErrorMsg:
		cmd := a.romPanel.HandleLoadError(msg)
		return a, cmd

	case TransferStartMsg:
		if a.selectedClient != nil && a.needsPassword(a.selectedClient) {
			a.pendingAction.kind = pendingTransfer
			a.pendingAction.romNames = msg.ROMNames
			a.mode = ModePassword
			a.overlay = NewPasswordModel(a, a.selectedClient.Name, a.selectedClient.Host, a.selectedClient.User)
			return a, a.overlay.Init()
		}
		a.mode = ModeTransfer
		a.overlay = NewTransferModel(a, msg.ROMNames)
		return a, a.overlay.Init()

	case TransferCompleteMsg:
		if a.overlay != nil {
			cmd := a.overlay.Update(msg)
			return a, cmd
		}
		return a, nil

	case transferTickMsg:
		if a.overlay != nil {
			cmd := a.overlay.Update(msg)
			return a, cmd
		}
		return a, nil

	case PasswordEnteredMsg:
		a.passwords[msg.ClientName] = msg.Password
		a.overlay = nil
		pending := a.pendingAction
		a.pendingAction.kind = pendingNone
		a.pendingAction.romNames = nil
		a.mode = ModeNormal
		switch pending.kind {
		case pendingLoadROMs:
			return a, a.romPanel.LoadROMs()
		case pendingTransfer:
			romNames := pending.romNames
			return a, func() tea.Msg {
				return TransferStartMsg{ROMNames: romNames}
			}
		}
		return a, nil

	case CancelOverlayMsg:
		wasTransfer := a.mode == ModeTransfer
		wasPassword := a.mode == ModePassword
		if c, ok := a.overlay.(interface{ Close() }); ok {
			c.Close()
		}
		a.mode = ModeNormal
		a.overlay = nil
		if wasPassword {
			a.pendingAction.kind = pendingNone
			a.pendingAction.romNames = nil
			return a, nil
		}
		// After transfer completes, clear selection and reload ROMs
		if wasTransfer && a.selectedClient != nil && a.selectedConsole != nil {
			a.romPanel.selected = make(map[string]bool)
			return a, a.romPanel.LoadROMs()
		}
		return a, nil

	case DirConnectedMsg, DirListedMsg, DirConnectErrorMsg:
		if a.overlay != nil {
			cmd := a.overlay.Update(msg)
			return a, cmd
		}
		return a, nil

	case ScanResultMsg:
		cmd := a.scanPanel.HandleScanResult(msg)
		return a, cmd

	case spinner.TickMsg:
		cmd := a.scanPanel.UpdateSpinner(msg)
		return a, cmd

	case ConfigUpdatedMsg:
		a.cfg = msg.Config
		a.devicePanel.Rebuild(a.cfg)
		a.consolePanel.Rebuild(a.cfg)
		return a, nil

	case ErrorMsg:
		a.setError(msg.Err.Error())
		return a, a.clearErrorAfter(5 * time.Second)

	case ClearErrorMsg:
		a.errMsg = ""
		return a, nil
	}

	// Forward blink and other messages to overlay if active
	if a.mode != ModeNormal && a.overlay != nil {
		cmd := a.overlay.Update(msg)
		return a, cmd
	}

	return a, nil
}

func (a *App) View() string {
	if a.width == 0 {
		return ""
	}

	var sections []string

	statusH := 1
	helpH := 0
	if a.showHelp {
		helpH = 5
	}

	// ── Top row: banner (left) + info panel (right) ──
	bannerView := StyleBanner.Render(banner)
	infoW := a.width - bannerW
	if infoW < 12 {
		infoW = 12
	}
	infoInnerH := bannerH - 2 // match banner height minus border
	a.metadataPanel.SetSize(infoW, infoInnerH)
	infoView := a.metadataPanel.View()
	topRow := lipgloss.JoinHorizontal(lipgloss.Top, bannerView, infoView)
	sections = append(sections, topRow)

	// ── Panel area: everything below banner ──
	fullH := a.height - bannerH - statusH - helpH
	if fullH < 6 {
		fullH = 6
	}

	// Column widths
	leftW := a.width * 25 / 100
	rightW := a.width - leftW
	if leftW < 10 {
		leftW = 10
	}
	if rightW < 20 {
		rightW = 20
	}

	// Left column: two bordered panels stacking to fullH
	leftInnerTotal := fullH - 4 // 2 borders x 2 panels
	if leftInnerTotal < 2 {
		leftInnerTotal = 2
	}
	deviceInnerH := leftInnerTotal / 2
	scanInnerH := leftInnerTotal - deviceInnerH

	a.devicePanel.SetSize(leftW, deviceInnerH)
	a.scanPanel.SetSize(leftW, scanInnerH)

	deviceView := a.devicePanel.View(a.focus == PanelDevices)
	scanView := a.scanPanel.View(a.focus == PanelScan)
	leftCol := lipgloss.JoinVertical(lipgloss.Left, deviceView, scanView)

	// ── Combined browser panel: console tabs │ separator │ ROMs ──
	browserInnerH := fullH - 2 // single border
	if browserInnerH < 2 {
		browserInnerH = 2
	}

	rightInnerW := rightW - 2 // subtract combined border
	tabW := rightW * 25 / 100
	if tabW < 8 {
		tabW = 8
	}
	consoleBlockW := tabW + 1 // includes integrated separator column
	romContentW := rightInnerW - consoleBlockW
	if romContentW < 10 {
		romContentW = 10
	}

	consoleFocused := a.focus == PanelConsoles
	romFocused := a.focus == PanelROMs
	browserFocused := consoleFocused || romFocused

	// Console panel renders tabs with integrated separator (breaks on active tab)
	consoleBlock := a.consolePanel.ViewBlock(consoleFocused, consoleBlockW, browserInnerH)

	if a.mode == ModePassword && a.overlay != nil {
		// Password dialog floats over the full panel area
		panelArea := a.overlay.View(a.width, fullH)
		sections = append(sections, panelArea)
	} else {
		var romBlock string
		if a.mode != ModeNormal && a.overlay != nil {
			romBlock = a.overlay.View(romContentW, browserInnerH)
		} else {
			romBlock = a.romPanel.ViewBlock(romFocused, romContentW, browserInnerH)
		}

		innerContent := lipgloss.JoinHorizontal(lipgloss.Top, consoleBlock, romBlock)

		browserStyle := StylePanelUnfocused
		if browserFocused {
			browserStyle = StylePanelFocused
		}
		browserPanel := browserStyle.
			Width(rightInnerW).
			Height(browserInnerH).
			Render(innerContent)

		mainRow := lipgloss.JoinHorizontal(lipgloss.Top, leftCol, browserPanel)
		sections = append(sections, mainRow)
	}

	// ── Help overlay ──
	if a.showHelp {
		a.help.ShowAll = true
		helpView := StyleHelp.Render(a.help.View(a.keys))
		sections = append(sections, helpView)
	}

	// ── Status bar ──
	status := a.buildStatusBar()
	sections = append(sections, status)

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

func styledHint(key, action string) string {
	return StyleHintKey.Render(key) + " " + lipgloss.NewStyle().Foreground(colorLightGrey).Render(action)
}

func (a *App) buildStatusBar() string {
	if a.errMsg != "" {
		return StyleError.Width(a.width).Render(" " + a.errMsg)
	}

	var parts []string

	switch a.mode {
	case ModeNormal:
		parts = append(parts, styledHint("tab", "panel"))
		switch a.focus {
		case PanelDevices:
			parts = append(parts, styledHint("enter", "select"), styledHint("a", "add"), styledHint("e", "edit"), styledHint("d", "del"), styledHint("c", "settings"))
		case PanelScan:
			parts = append(parts, styledHint("enter", "add device"))
		case PanelConsoles:
			parts = append(parts, styledHint("enter", "select"))
		case PanelROMs:
			parts = append(parts, styledHint("enter", "select"), styledHint("p", "push"), styledHint("←/→", "filter"))
		}
		parts = append(parts, styledHint("s", "scan"), styledHint("?", "help"), styledHint("q", "quit"))
	case ModeEditing:
		parts = append(parts, styledHint("enter", "save"), styledHint("esc", "cancel"), styledHint("tab", "field"), styledHint("ctrl+t", "connect"), styledHint("ctrl+b", "browse"))
	case ModeTransfer:
		parts = append(parts, StyleHintKey.Render("transferring..."))
	case ModeSettings:
		parts = append(parts, styledHint("enter", "save"), styledHint("esc", "cancel"))
	case ModePassword:
		parts = append(parts, styledHint("enter", "submit"), styledHint("esc", "cancel"))
	}

	joined := strings.Join(parts, StyleHintSep.Render(" │ "))
	return StyleStatusBar.Width(a.width).Render(joined)
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

func (a *App) needsPassword(c *config.Client) bool {
	if c.Auth.Method != "password" {
		return false
	}
	_, ok := a.passwords[c.Name]
	return !ok
}

func (a *App) resolvePassword(c config.Client) config.Client {
	if c.Auth.Method == "password" {
		if pw, ok := a.passwords[c.Name]; ok {
			c.Auth.Password = pw
		}
	}
	return c
}
