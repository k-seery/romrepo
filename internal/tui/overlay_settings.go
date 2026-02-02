package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"romrepo/internal/config"
)

type SettingsModel struct {
	app   *App
	input textinput.Model
}

func NewSettingsModel(app *App) *SettingsModel {
	ti := textinput.New()
	ti.Prompt = "ROM Directory: "
	ti.Placeholder = "~/roms"
	ti.Width = 40
	ti.SetValue(app.cfg.Server.ROMDir)
	ti.Focus()

	return &SettingsModel{
		app:   app,
		input: ti,
	}
}

func (m *SettingsModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m *SettingsModel) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return func() tea.Msg { return CancelOverlayMsg{} }

		case "enter":
			val := strings.TrimSpace(m.input.Value())
			if val == "" {
				return func() tea.Msg {
					return ErrorMsg{Err: fmt.Errorf("ROM directory cannot be empty")}
				}
			}
			m.app.cfg.Server.ROMDir = val
			if err := config.Save(m.app.cfg, m.app.cfgPath); err != nil {
				return func() tea.Msg { return ErrorMsg{Err: err} }
			}
			m.app.mode = ModeNormal
			m.app.overlay = nil
			return func() tea.Msg { return ConfigUpdatedMsg{Config: m.app.cfg} }
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return cmd
}

func (m *SettingsModel) View(w, h int) string {
	var b strings.Builder
	b.WriteString(StylePanelTitleFocused.Render("Settings"))
	b.WriteString("\n\n")
	b.WriteString("  " + m.input.View())
	b.WriteString("\n\n  enter:save  esc:cancel")

	return lipgloss.NewStyle().Width(w).Height(h).MaxHeight(h).Render(b.String())
}
