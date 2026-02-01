package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	"romrepo/internal/config"
)

type consoleItem struct {
	console config.Console
}

func (i consoleItem) Title() string       { return i.console.Name }
func (i consoleItem) Description() string { return fmt.Sprintf("dir: %s  ext: %v", i.console.Dir, i.console.Extensions) }
func (i consoleItem) FilterValue() string { return i.console.Name }

type ConsoleScreen struct {
	list list.Model
	app  *App
}

func NewConsoleScreen(cfg *config.Config, app *App) *ConsoleScreen {
	items := make([]list.Item, len(cfg.Server.Consoles))
	for i, c := range cfg.Server.Consoles {
		items[i] = consoleItem{console: c}
	}

	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = StyleSelected
	delegate.Styles.SelectedDesc = StyleSelected.Copy().Faint(true)

	l := list.New(items, delegate, app.width, app.height-4)
	l.Title = "Consoles"
	l.SetShowTitle(false)
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.SetShowHelp(false)

	return &ConsoleScreen{list: l, app: app}
}

func (s *ConsoleScreen) Title() string { return "Consoles" }

func (s *ConsoleScreen) Init() tea.Cmd { return nil }

func (s *ConsoleScreen) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			if item, ok := s.list.SelectedItem().(consoleItem); ok {
				return s, func() tea.Msg { return SelectConsoleMsg{Console: item.console} }
			}

		case key.Matches(msg, s.app.keys.Back):
			return s, func() tea.Msg { return GoBackMsg{} }
		}
	}

	var cmd tea.Cmd
	s.list, cmd = s.list.Update(msg)
	return s, cmd
}

func (s *ConsoleScreen) View() string {
	return s.list.View()
}
