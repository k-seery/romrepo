package tui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"romrepo/internal/config"
)

const (
	editInputName = iota
	editInputHost
	editInputPort
	editInputUser
	editInputAuthMethod
	editInputKeyPath
	editInputPassword
	editInputROMDir
	editInputCount
)

const (
	focusForm    = 0
	focusBrowser = 1
)

type EditFormModel struct {
	app        *App
	inputs     []textinput.Model
	focusIdx   int
	editIdx    int // index into cfg.Clients, -1 for new
	browser    *DirBrowser
	focusPanel int
}

func NewEditFormModel(app *App, c *config.Client, editIdx int) *EditFormModel {
	m := &EditFormModel{
		app:        app,
		editIdx:    editIdx,
		browser:    NewDirBrowser(app),
		focusPanel: focusForm,
	}
	m.initInputs(c)
	return m
}

func (m *EditFormModel) initInputs(c *config.Client) {
	m.inputs = make([]textinput.Model, editInputCount)

	labels := []string{"Name", "Host", "Port", "User", "Auth Method (key/password)", "Key Path", "Password", "ROM Dir"}
	placeholders := []string{"my-device", "192.168.1.100", "22", "pi", "key", "~/.ssh/id_rsa", "", "/home/pi/roms"}

	for i := 0; i < editInputCount; i++ {
		t := textinput.New()
		t.Placeholder = placeholders[i]
		t.Prompt = labels[i] + ": "
		t.Width = 40

		if c != nil {
			switch i {
			case editInputName:
				t.SetValue(c.Name)
			case editInputHost:
				t.SetValue(c.Host)
			case editInputPort:
				t.SetValue(strconv.Itoa(c.Port))
			case editInputUser:
				t.SetValue(c.User)
			case editInputAuthMethod:
				t.SetValue(c.Auth.Method)
			case editInputKeyPath:
				t.SetValue(c.Auth.KeyPath)
			case editInputPassword:
				t.SetValue(c.Auth.Password)
				t.EchoMode = textinput.EchoPassword
			case editInputROMDir:
				t.SetValue(c.ROMDir)
			}
		}

		m.inputs[i] = t
	}

	m.focusIdx = 0
	m.inputs[0].Focus()
}

func (m *EditFormModel) Init() tea.Cmd {
	cmds := []tea.Cmd{textinput.Blink}

	// Auto-connect for existing clients with credentials filled
	if m.editIdx >= 0 {
		c := m.buildClientFromForm()
		if c.Host != "" && c.User != "" && c.Auth.Method != "" {
			cmds = append(cmds, m.browser.Connect(c))
		}
	}

	return tea.Batch(cmds...)
}

func (m *EditFormModel) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case DirConnectedMsg:
		romDir := strings.TrimSpace(m.inputs[editInputROMDir].Value())
		return m.browser.HandleConnectedWithPath(msg, romDir)

	case DirListedMsg:
		m.browser.HandleListed(msg)
		return nil

	case DirConnectErrorMsg:
		m.browser.HandleConnectError(msg)
		return nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return tea.Quit

		case "ctrl+t":
			c := m.buildClientFromForm()
			if c.Host == "" || c.User == "" || c.Auth.Method == "" {
				return func() tea.Msg {
					return ErrorMsg{Err: fmt.Errorf("host, user, and auth method required to connect")}
				}
			}
			return m.browser.Connect(c)

		case "ctrl+b":
			if m.focusPanel == focusForm && m.browser.connected {
				m.focusPanel = focusBrowser
				m.inputs[m.focusIdx].Blur()
			} else {
				m.focusPanel = focusForm
				m.inputs[m.focusIdx].Focus()
			}
			return nil
		}

		if m.focusPanel == focusBrowser {
			switch msg.String() {
			case "esc":
				m.focusPanel = focusForm
				m.inputs[m.focusIdx].Focus()
				return nil
			}

			cmd, selected := m.browser.Update(msg)
			if selected {
				m.inputs[editInputROMDir].SetValue(m.browser.SelectedPath())
				m.focusPanel = focusForm
				m.inputs[m.focusIdx].Focus()
				return cmd
			}
			return cmd
		}

		// Form focus
		switch msg.String() {
		case "esc":
			return func() tea.Msg { return CancelOverlayMsg{} }

		case "tab", "down":
			m.inputs[m.focusIdx].Blur()
			m.focusIdx = (m.focusIdx + 1) % editInputCount
			m.inputs[m.focusIdx].Focus()
			return nil

		case "shift+tab", "up":
			m.inputs[m.focusIdx].Blur()
			m.focusIdx = (m.focusIdx - 1 + editInputCount) % editInputCount
			m.inputs[m.focusIdx].Focus()
			return nil

		case "enter":
			return m.save()
		}

		var cmd tea.Cmd
		m.inputs[m.focusIdx], cmd = m.inputs[m.focusIdx].Update(msg)
		return cmd
	}

	// Forward non-key messages to focused input (e.g. blink)
	var cmd tea.Cmd
	m.inputs[m.focusIdx], cmd = m.inputs[m.focusIdx].Update(msg)
	return cmd
}

func (m *EditFormModel) buildClientFromForm() config.Client {
	port, _ := strconv.Atoi(m.inputs[editInputPort].Value())
	if port == 0 {
		port = 22
	}
	return config.Client{
		Name: strings.TrimSpace(m.inputs[editInputName].Value()),
		Host: strings.TrimSpace(m.inputs[editInputHost].Value()),
		Port: port,
		User: strings.TrimSpace(m.inputs[editInputUser].Value()),
		Auth: config.AuthConfig{
			Method:   strings.TrimSpace(m.inputs[editInputAuthMethod].Value()),
			KeyPath:  strings.TrimSpace(m.inputs[editInputKeyPath].Value()),
			Password: m.inputs[editInputPassword].Value(),
		},
		ROMDir: strings.TrimSpace(m.inputs[editInputROMDir].Value()),
	}
}

func (m *EditFormModel) save() tea.Cmd {
	client := m.buildClientFromForm()

	if client.Name == "" || client.Host == "" || client.User == "" {
		return func() tea.Msg {
			return ErrorMsg{Err: fmt.Errorf("name, host, and user are required")}
		}
	}

	if m.editIdx >= 0 {
		m.app.cfg.Clients[m.editIdx] = client
	} else {
		m.app.cfg.Clients = append(m.app.cfg.Clients, client)
	}

	if err := config.Save(m.app.cfg, m.app.cfgPath); err != nil {
		return func() tea.Msg { return ErrorMsg{Err: err} }
	}

	m.app.mode = ModeNormal
	m.app.overlay = nil

	return func() tea.Msg { return ConfigUpdatedMsg{Config: m.app.cfg} }
}

func (m *EditFormModel) View(w, h int) string {
	// Split into form (65%) and browser (35%)
	browserW := w * 35 / 100
	if browserW < 12 {
		browserW = 12
	}
	sepW := 1
	formW := w - browserW - sepW
	if formW < 20 {
		formW = 20
	}

	formView := m.renderForm(formW, h)
	sep := m.renderSeparator(h)
	browserView := m.browser.View(browserW, h)

	return lipgloss.JoinHorizontal(lipgloss.Top, formView, sep, browserView)
}

func (m *EditFormModel) renderForm(w, h int) string {
	var b strings.Builder

	title := "Add Client"
	if m.editIdx >= 0 {
		title = "Edit Client"
	}
	titleStyle := StylePanelTitleFocused
	if m.focusPanel != focusForm {
		titleStyle = StylePanelTitle
	}
	b.WriteString(titleStyle.Render(title))
	b.WriteString("\n\n")

	for i, input := range m.inputs {
		if i == m.focusIdx && m.focusPanel == focusForm {
			b.WriteString("  > ")
		} else {
			b.WriteString("    ")
		}
		b.WriteString(input.View())
		b.WriteString("\n")
	}

	b.WriteString("\n  enter:save  esc:cancel  tab:next")
	if m.browser.connected {
		b.WriteString("\n  ctrl+b:browse")
	} else {
		b.WriteString("\n  ctrl+t:connect")
	}

	return lipgloss.NewStyle().Width(w).Height(h).MaxHeight(h).Render(b.String())
}

func (m *EditFormModel) renderSeparator(h int) string {
	sep := strings.Repeat("│\n", h)
	if len(sep) > 0 {
		sep = sep[:len(sep)-1] // trim trailing newline
	}
	return StyleSeparator.Render(sep)
}
