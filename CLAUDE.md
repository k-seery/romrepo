# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Run

```bash
go build -o romrepo .     # build binary
./romrepo                  # run (default config: ~/.config/romrepo/config.yaml)
./romrepo -config path.yaml  # run with custom config
```

No tests, linting, or CI/CD are configured yet.

## Architecture

RomRepo is a Go TUI for syncing retro game ROMs between a central server and remote devices (e.g. RetroPie) over SSH/SFTP. Built with Bubble Tea.

### Screen Stack Navigation

The app uses a stack-based screen model (`App.stack []Screen`). Screens are pushed/popped for navigation, with breadcrumbs generated from screen titles. The flow:

```
ClientScreen → ConsoleScreen → ROMScreen → TransferScreen
     ↓ (m key)
ManageScreen (add/edit/delete/scan clients)
```

All screens implement `Screen` (embeds `tea.Model` + `Title() string`). Domain events bubble up as Bubble Tea messages — screens emit typed messages (e.g. `SelectClientMsg`), and `App.Update()` handles routing, screen transitions, and context tracking (`selectedClient`, `selectedConsole`).

### Key Packages

- **internal/tui** — Bubble Tea app, all screens, keybindings, styles, and message types. Screen-specific help text is rendered inline in each screen's `View()` method, not through the list component's help system.
- **internal/config** — YAML config loading/saving/validation. Defines server ROM paths, console definitions (with file extensions), and client connection details (host, user, auth).
- **internal/remote** — SSH connection pool (`ConnManager`) with keepalive, and SFTP push/pull with progress callbacks.
- **internal/rom** — ROM file enumeration (`Library`) filtered by console extensions, and diff logic comparing server vs. client ROM sets.
- **internal/network** — Subnet scanner: probes TCP/22 across a /24 with 50-goroutine concurrency, reverse DNS lookups, 500ms timeout.

### Patterns

- Async operations (scanning, ROM loading, transfers) use Bubble Tea commands returning messages.
- Transfer progress uses atomic counters for thread-safe updates from SFTP goroutines.
- ManageScreen has three view modes (list/edit/scan) switched via internal state, not separate screens.
- Config is passed by pointer through the app; `ConfigUpdatedMsg` triggers UI rebuilds after edits.
