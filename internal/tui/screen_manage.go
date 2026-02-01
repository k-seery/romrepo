package tui

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"romrepo/internal/config"
	"romrepo/internal/network"
)

type manageMode int

const (
	manageList manageMode = iota
	manageEdit
	manageScan
)

type ManageScreen struct {
	app        *App
	mode       manageMode
	list       list.Model
	inputs     []textinput.Model
	focusIdx   int
	editIdx    int // index into cfg.Clients, -1 for new
	scanList   list.Model
	scanning   bool
	scanCancel context.CancelFunc
	spinner    spinner.Model
}

// deviceItem implements list.Item for scan results.
type deviceItem struct {
	device network.Device
}

func (d deviceItem) Title() string {
	ip := d.device.IP.String()
	if d.device.Hostname != "" {
		return fmt.Sprintf("%s (%s)", ip, d.device.Hostname)
	}
	return ip
}

func (d deviceItem) Description() string {
	if d.device.SSHOpen {
		return StyleSSHOpen.Render("SSH open")
	}
	return "SSH closed"
}

func (d deviceItem) FilterValue() string {
	return d.device.IP.String()
}

const (
	inputName = iota
	inputHost
	inputPort
	inputUser
	inputAuthMethod
	inputKeyPath
	inputPassword
	inputROMDir
	inputCount
)

func NewManageScreen(cfg *config.Config, app *App) *ManageScreen {
	sp := spinner.New()
	sp.Spinner = spinner.Dot

	s := &ManageScreen{
		app:     app,
		mode:    manageList,
		spinner: sp,
	}
	s.rebuildList()
	return s
}

func (s *ManageScreen) rebuildList() {
	items := make([]list.Item, len(s.app.cfg.Clients))
	for i, c := range s.app.cfg.Clients {
		items[i] = clientItem{client: c}
	}

	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = StyleSelected
	delegate.Styles.SelectedDesc = StyleSelected.Copy().Faint(true)

	l := list.New(items, delegate, s.app.width, s.app.height-4)
	l.Title = "Manage Clients"
	l.SetShowTitle(false)
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)

	s.list = l
}

func (s *ManageScreen) Title() string { return "Manage" }

func (s *ManageScreen) Init() tea.Cmd { return nil }

func (s *ManageScreen) initInputs(c *config.Client) {
	s.inputs = make([]textinput.Model, inputCount)

	labels := []string{"Name", "Host", "Port", "User", "Auth Method (key/password)", "Key Path", "Password", "ROM Dir"}
	placeholders := []string{"my-device", "192.168.1.100", "22", "pi", "key", "~/.ssh/id_rsa", "", "/home/pi/roms"}

	for i := 0; i < inputCount; i++ {
		t := textinput.New()
		t.Placeholder = placeholders[i]
		t.Prompt = labels[i] + ": "
		t.Width = 40

		if c != nil {
			switch i {
			case inputName:
				t.SetValue(c.Name)
			case inputHost:
				t.SetValue(c.Host)
			case inputPort:
				t.SetValue(strconv.Itoa(c.Port))
			case inputUser:
				t.SetValue(c.User)
			case inputAuthMethod:
				t.SetValue(c.Auth.Method)
			case inputKeyPath:
				t.SetValue(c.Auth.KeyPath)
			case inputPassword:
				t.SetValue(c.Auth.Password)
				t.EchoMode = textinput.EchoPassword
			case inputROMDir:
				t.SetValue(c.ROMDir)
			}
		}

		s.inputs[i] = t
	}

	s.focusIdx = 0
	s.inputs[0].Focus()
}

func (s *ManageScreen) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		s.list.SetSize(msg.Width, msg.Height-4)
		if s.mode == manageScan {
			s.scanList.SetSize(msg.Width, msg.Height-4)
		}
		return s, nil

	case ScanResultMsg:
		s.scanning = false
		if msg.Err != nil {
			return s, func() tea.Msg { return ErrorMsg{Err: msg.Err} }
		}
		s.buildScanList(msg.Devices)
		return s, nil

	case spinner.TickMsg:
		if s.mode == manageScan && s.scanning {
			var cmd tea.Cmd
			s.spinner, cmd = s.spinner.Update(msg)
			return s, cmd
		}
		return s, nil

	case tea.KeyMsg:
		switch s.mode {
		case manageList:
			return s.updateList(msg)
		case manageEdit:
			return s.updateEdit(msg)
		case manageScan:
			return s.updateScan(msg)
		}
	}

	switch s.mode {
	case manageList:
		var cmd tea.Cmd
		s.list, cmd = s.list.Update(msg)
		return s, cmd
	case manageScan:
		if !s.scanning {
			var cmd tea.Cmd
			s.scanList, cmd = s.scanList.Update(msg)
			return s, cmd
		}
	}
	return s, nil
}

func (s *ManageScreen) updateList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, s.app.keys.Back):
		return s, func() tea.Msg { return GoBackMsg{} }

	case key.Matches(msg, s.app.keys.Add):
		s.mode = manageEdit
		s.editIdx = -1
		s.initInputs(nil)
		return s, nil

	case key.Matches(msg, s.app.keys.Edit):
		idx := s.list.Index()
		if idx >= 0 && idx < len(s.app.cfg.Clients) {
			s.mode = manageEdit
			s.editIdx = idx
			s.initInputs(&s.app.cfg.Clients[idx])
		}
		return s, nil

	case key.Matches(msg, s.app.keys.Delete):
		idx := s.list.Index()
		if idx >= 0 && idx < len(s.app.cfg.Clients) {
			s.app.cfg.Clients = append(s.app.cfg.Clients[:idx], s.app.cfg.Clients[idx+1:]...)
			if err := config.Save(s.app.cfg, s.app.cfgPath); err != nil {
				return s, func() tea.Msg { return ErrorMsg{Err: err} }
			}
			s.rebuildList()
			return s, func() tea.Msg { return ConfigUpdatedMsg{Config: s.app.cfg} }
		}
		return s, nil

	case key.Matches(msg, s.app.keys.Scan):
		s.mode = manageScan
		s.scanning = true
		return s, tea.Batch(s.spinner.Tick, s.startScan())
	}

	var cmd tea.Cmd
	s.list, cmd = s.list.Update(msg)
	return s, cmd
}

func (s *ManageScreen) updateEdit(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		s.mode = manageList
		return s, nil

	case "tab", "down":
		s.inputs[s.focusIdx].Blur()
		s.focusIdx = (s.focusIdx + 1) % inputCount
		s.inputs[s.focusIdx].Focus()
		return s, nil

	case "shift+tab", "up":
		s.inputs[s.focusIdx].Blur()
		s.focusIdx = (s.focusIdx - 1 + inputCount) % inputCount
		s.inputs[s.focusIdx].Focus()
		return s, nil

	case "enter":
		return s, s.saveClient()
	}

	var cmd tea.Cmd
	s.inputs[s.focusIdx], cmd = s.inputs[s.focusIdx].Update(msg)
	return s, cmd
}

func (s *ManageScreen) updateScan(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, s.app.keys.Back):
		if s.scanCancel != nil {
			s.scanCancel()
			s.scanCancel = nil
		}
		s.scanning = false
		s.mode = manageList
		return s, nil

	case key.Matches(msg, s.app.keys.Enter):
		if s.scanning {
			return s, nil
		}
		if item, ok := s.scanList.SelectedItem().(deviceItem); ok {
			s.mode = manageEdit
			s.editIdx = -1
			port := 22
			hostname := item.device.Hostname
			if hostname == "" {
				hostname = item.device.IP.String()
			}
			c := &config.Client{
				Name: hostname,
				Host: item.device.IP.String(),
				Port: port,
			}
			s.initInputs(c)
		}
		return s, nil
	}

	if !s.scanning {
		var cmd tea.Cmd
		s.scanList, cmd = s.scanList.Update(msg)
		return s, cmd
	}
	return s, nil
}

func (s *ManageScreen) startScan() tea.Cmd {
	return func() tea.Msg {
		subnet, err := network.LocalSubnet()
		if err != nil {
			return ScanResultMsg{Err: err}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		s.scanCancel = cancel

		devices, err := network.ScanSubnet(ctx, subnet)
		cancel()
		return ScanResultMsg{Devices: devices, Err: err}
	}
}

func (s *ManageScreen) buildScanList(devices []network.Device) {
	items := make([]list.Item, len(devices))
	for i, d := range devices {
		items[i] = deviceItem{device: d}
	}

	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = StyleSelected
	delegate.Styles.SelectedDesc = StyleSelected.Copy().Faint(true)

	l := list.New(items, delegate, s.app.width, s.app.height-4)
	l.Title = "Network Scan Results"
	l.SetShowTitle(true)
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)

	s.scanList = l
}

func (s *ManageScreen) saveClient() tea.Cmd {
	port, _ := strconv.Atoi(s.inputs[inputPort].Value())
	if port == 0 {
		port = 22
	}

	client := config.Client{
		Name:   strings.TrimSpace(s.inputs[inputName].Value()),
		Host:   strings.TrimSpace(s.inputs[inputHost].Value()),
		Port:   port,
		User:   strings.TrimSpace(s.inputs[inputUser].Value()),
		Auth: config.AuthConfig{
			Method:   strings.TrimSpace(s.inputs[inputAuthMethod].Value()),
			KeyPath:  strings.TrimSpace(s.inputs[inputKeyPath].Value()),
			Password: s.inputs[inputPassword].Value(),
		},
		ROMDir: strings.TrimSpace(s.inputs[inputROMDir].Value()),
	}

	if client.Name == "" || client.Host == "" || client.User == "" {
		return func() tea.Msg {
			return ErrorMsg{Err: fmt.Errorf("name, host, and user are required")}
		}
	}

	if s.editIdx >= 0 {
		s.app.cfg.Clients[s.editIdx] = client
	} else {
		s.app.cfg.Clients = append(s.app.cfg.Clients, client)
	}

	if err := config.Save(s.app.cfg, s.app.cfgPath); err != nil {
		return func() tea.Msg { return ErrorMsg{Err: err} }
	}

	s.mode = manageList
	s.rebuildList()

	return func() tea.Msg { return ConfigUpdatedMsg{Config: s.app.cfg} }
}

func (s *ManageScreen) View() string {
	switch s.mode {
	case manageEdit:
		return s.viewEdit()
	case manageScan:
		return s.viewScan()
	default:
		return s.viewList()
	}
}

func (s *ManageScreen) viewList() string {
	help := StyleHelp.Render("a:add  e:edit  d:delete  s:scan  esc:back")
	return s.list.View() + "\n" + help
}

func (s *ManageScreen) viewScan() string {
	if s.scanning {
		return fmt.Sprintf("\n  %s Scanning local network...\n\n  esc:cancel", s.spinner.View())
	}

	if len(s.scanList.Items()) == 0 {
		return "\n  No devices found.\n\n  esc:back"
	}

	help := StyleHelp.Render("enter:select  esc:back")
	return s.scanList.View() + "\n" + help
}

func (s *ManageScreen) viewEdit() string {
	var b strings.Builder
	title := "  Add Client"
	if s.editIdx >= 0 {
		title = "  Edit Client"
	}
	b.WriteString(title + "\n\n")

	for i, input := range s.inputs {
		if i == s.focusIdx {
			b.WriteString("  > ")
		} else {
			b.WriteString("    ")
		}
		b.WriteString(input.View())
		b.WriteString("\n")
	}

	b.WriteString("\n  enter:save  esc:cancel  tab:next field")
	return b.String()
}
