package tui

import "github.com/charmbracelet/bubbles/key"

type KeyMap struct {
	Enter    key.Binding
	Back     key.Binding
	Quit     key.Binding
	Push     key.Binding
	Pull     key.Binding
	Help     key.Binding
	Manage   key.Binding
	Add      key.Binding
	Edit     key.Binding
	Delete   key.Binding
	Filter   key.Binding
	Scan     key.Binding
	Settings key.Binding
}

func DefaultKeyMap() KeyMap {
	return KeyMap{
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "select"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "back"),
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
		Manage: key.NewBinding(
			key.WithKeys("m"),
			key.WithHelp("m", "manage clients"),
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
	}
}

func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Enter, k.Back, k.Quit, k.Help}
}

func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Enter, k.Back, k.Quit},
		{k.Push, k.Pull, k.Filter},
		{k.Manage, k.Scan, k.Settings, k.Help},
	}
}
