package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"romrepo/internal/config"
	"romrepo/internal/remote"
	"romrepo/internal/tui"
)

func main() {
	configPath := flag.String("config", "", "path to config file (default: ~/.config/romrepo/config.yaml)")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	connMgr := remote.NewConnManager()
	defer connMgr.CloseAll()

	app := tui.NewApp(cfg, connMgr)
	p := tea.NewProgram(app, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
