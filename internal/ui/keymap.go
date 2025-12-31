package ui

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	Quit         key.Binding
	Help         key.Binding
	Pause        key.Binding
	Refresh      key.Binding
	WindowShort  key.Binding
	WindowMid    key.Binding
	WindowLong   key.Binding
	FocusNext    key.Binding
	Filter       key.Binding
	ToggleEvents key.Binding
	ToggleTheme  key.Binding
	Snapshot     key.Binding
	Drilldown    key.Binding
	Auth         key.Binding
}

func newKeyMap() keyMap {
	return keyMap{
		Quit:         key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
		Help:         key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
		Pause:        key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "pause")),
		Refresh:      key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
		WindowShort:  key.NewBinding(key.WithKeys("1"), key.WithHelp("1", "1m window")),
		WindowMid:    key.NewBinding(key.WithKeys("2"), key.WithHelp("2", "5m window")),
		WindowLong:   key.NewBinding(key.WithKeys("3"), key.WithHelp("3", "15m window")),
		FocusNext:    key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "next pane")),
		Filter:       key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter")),
		ToggleEvents: key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "toggle events")),
		ToggleTheme:  key.NewBinding(key.WithKeys("t"), key.WithHelp("t", "theme")),
		Snapshot:     key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "snapshot")),
		Drilldown:    key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "details")),
		Auth:         key.NewBinding(key.WithKeys("l"), key.WithHelp("l", "login/logout")),
	}
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Help, k.Pause, k.Refresh, k.WindowShort, k.WindowMid, k.WindowLong, k.Filter, k.Drilldown, k.ToggleEvents, k.Auth, k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Help, k.Pause, k.Refresh, k.Snapshot},
		{k.WindowShort, k.WindowMid, k.WindowLong, k.FocusNext},
		{k.Filter, k.Drilldown, k.ToggleEvents, k.ToggleTheme},
		{k.Auth},
		{k.Quit},
	}
}
