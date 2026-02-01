package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"romrepo/internal/config"
)

type SettingsScreen struct {
	input textinput.Model
	app   *App
}

func NewSettingsScreen(cfg *config.Config, app *App) *SettingsScreen {
	ti := textinput.New()
	ti.Prompt = "ROM Directory: "
	ti.Placeholder = "~/roms"
	ti.Width = 40
	ti.SetValue(cfg.Server.ROMDir)
	ti.Focus()

	return &SettingsScreen{
		input: ti,
		app:   app,
	}
}

func (s *SettingsScreen) Title() string { return "Settings" }

func (s *SettingsScreen) Init() tea.Cmd {
	return textinput.Blink
}

func (s *SettingsScreen) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return s, func() tea.Msg { return GoBackMsg{} }

		case "enter":
			val := strings.TrimSpace(s.input.Value())
			if val == "" {
				return s, func() tea.Msg {
					return ErrorMsg{Err: fmt.Errorf("ROM directory cannot be empty")}
				}
			}
			s.app.cfg.Server.ROMDir = val
			if err := config.Save(s.app.cfg, s.app.cfgPath); err != nil {
				return s, func() tea.Msg { return ErrorMsg{Err: err} }
			}
			return s, func() tea.Msg { return ConfigUpdatedMsg{Config: s.app.cfg} }
		}
	}

	var cmd tea.Cmd
	s.input, cmd = s.input.Update(msg)
	return s, cmd
}

func (s *SettingsScreen) View() string {
	help := StyleHelp.Render("enter:save  esc:back")
	return "\n" + s.input.View() + "\n\n" + help
}
