package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	"romrepo/internal/config"
)

type clientItem struct {
	client config.Client
}

func (i clientItem) Title() string       { return i.client.Name }
func (i clientItem) Description() string { return fmt.Sprintf("%s@%s:%d", i.client.User, i.client.Host, i.client.Port) }
func (i clientItem) FilterValue() string { return i.client.Name }

type ClientScreen struct {
	list list.Model
	app  *App
}

func NewClientScreen(cfg *config.Config, app *App) *ClientScreen {
	items := make([]list.Item, len(cfg.Clients))
	for i, c := range cfg.Clients {
		items[i] = clientItem{client: c}
	}

	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = StyleSelected
	delegate.Styles.SelectedDesc = StyleSelected.Copy().Faint(true)

	l := list.New(items, delegate, app.width, app.height-4)
	l.Title = "Clients"
	l.SetShowTitle(false)
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.SetShowHelp(false)
	l.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{app.keys.Manage}
	}

	return &ClientScreen{list: l, app: app}
}

func (s *ClientScreen) Title() string { return "Clients" }

func (s *ClientScreen) Init() tea.Cmd { return nil }

func (s *ClientScreen) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		s.list.SetSize(msg.Width, msg.Height-4)
		return s, nil

	case tea.KeyMsg:
		if s.list.FilterState() == list.Filtering {
			break
		}

		switch {
		case key.Matches(msg, s.app.keys.Enter):
			if item, ok := s.list.SelectedItem().(clientItem); ok {
				return s, func() tea.Msg { return SelectClientMsg{Client: item.client} }
			}

		case key.Matches(msg, s.app.keys.Back):
			return s, tea.Quit

		case key.Matches(msg, s.app.keys.Manage):
			return s, func() tea.Msg { return OpenManageMsg{} }
		}
	}

	var cmd tea.Cmd
	s.list, cmd = s.list.Update(msg)
	return s, cmd
}

func (s *ClientScreen) View() string {
	help := StyleHelp.Render("m:manage  enter:select")
	return s.list.View() + "\n" + help
}
