package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type PasswordModel struct {
	app        *App
	clientName string
	host       string
	user       string
	input      textinput.Model
}

func NewPasswordModel(app *App, clientName, host, user string) *PasswordModel {
	ti := textinput.New()
	ti.Placeholder = ""
	ti.Prompt = "Password: "
	ti.Width = 30
	ti.EchoMode = textinput.EchoPassword
	ti.Focus()

	return &PasswordModel{
		app:        app,
		clientName: clientName,
		host:       host,
		user:       user,
		input:      ti,
	}
}

func (m *PasswordModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m *PasswordModel) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			pw := m.input.Value()
			clientName := m.clientName
			return func() tea.Msg {
				return PasswordEnteredMsg{ClientName: clientName, Password: pw}
			}
		case "esc":
			return func() tea.Msg { return CancelOverlayMsg{} }
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return cmd
}

func (m *PasswordModel) View(w, h int) string {
	const dialogW = 40

	var b strings.Builder
	b.WriteString(StylePanelTitleFocused.Render("Password Required"))
	b.WriteString("\n\n")
	b.WriteString(fmt.Sprintf("  Host: %s\n", m.host))
	b.WriteString(fmt.Sprintf("  User: %s\n", m.user))
	b.WriteString("\n")
	b.WriteString("  " + m.input.View())
	b.WriteString("\n\n")
	b.WriteString("  enter:submit  esc:cancel")

	inner := b.String()

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorMagenta).
		Padding(1, 2).
		Width(dialogW).
		Render(inner)

	return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, box)
}
