package tui

import "github.com/charmbracelet/bubbles/key"

type KeyMap struct {
	Enter     key.Binding
	Quit      key.Binding
	Push      key.Binding
	Pull      key.Binding
	Help      key.Binding
	Add       key.Binding
	Edit      key.Binding
	Delete    key.Binding
	Filter    key.Binding
	Scan      key.Binding
	Settings  key.Binding
	FocusNext key.Binding
	FocusPrev key.Binding
	Escape    key.Binding
}

func DefaultKeyMap() KeyMap {
	return KeyMap{
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "select"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Push: key.NewBinding(
			key.WithKeys("p"),
			key.WithHelp("p", "push to client"),
		),
		Pull: key.NewBinding(
			key.WithKeys("l"),
			key.WithHelp("l", "pull from client"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Add: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "add"),
		),
		Edit: key.NewBinding(
			key.WithKeys("e"),
			key.WithHelp("e", "edit"),
		),
		Delete: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "delete"),
		),
		Filter: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "filter"),
		),
		Scan: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "scan network"),
		),
		Settings: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "settings"),
		),
		FocusNext: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "next panel"),
		),
		FocusPrev: key.NewBinding(
			key.WithKeys("shift+tab"),
			key.WithHelp("shift+tab", "prev panel"),
		),
		Escape: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "cancel"),
		),
	}
}

func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.FocusNext, k.Enter, k.Quit, k.Help}
}

func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.FocusNext, k.FocusPrev, k.Escape},
		{k.Enter, k.Push, k.Pull, k.Filter},
		{k.Add, k.Edit, k.Delete, k.Scan},
		{k.Settings, k.Quit, k.Help},
	}
}
