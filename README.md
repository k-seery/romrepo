# RomRepo

A terminal UI for syncing retro game ROMs from a central server to remote devices (e.g. RetroPie, MiSTer) over SSH/SFTP. Built with Go and [Bubble Tea](https://github.com/charmbracelet/bubbletea).

## Features

- **Multi-panel TUI** — browse devices, consoles, and ROMs in a single view
- **Multi-select transfers** — select multiple ROMs and push them in one batch
- **Network scanner** — discovers SSH-capable devices on your local subnet
- **Alphabet filtering** — quickly jump through large ROM libraries by letter
- **SSH/SFTP** — transfers over standard SSH with key or password authentication
- **YAML config** — define your server library path, consoles, file extensions, and client devices

## How It Works

When you select a device and console, RomRepo compares your server library against what's already on the device. Each ROM is marked as either **synced** (already on the device) or **server only** (available to transfer). This lets you browse your full library at a glance, see what's missing from a device, and pick new games to push over — without having to SSH in and diff directories yourself.

## Install

```bash
go build -o romrepo .
```

## Usage

```bash
./romrepo                      # uses default config: ~/.config/romrepo/config.yaml
./romrepo -config path.yaml    # use a custom config path
```

On first run a default config file is created. Edit it to set your server ROM directory and add your devices.

## Navigation

| Key         | Action              |
|-------------|---------------------|
| `tab`       | Cycle panels        |
| `enter`     | Select / toggle ROM |
| `p`         | Push selected ROMs  |
| `s`         | Scan network        |
| `a` `e` `d` | Add / edit / delete device |
| `←` `→`    | Filter ROMs by letter |
| `?`         | Help                |
| `q`         | Quit                |

## Requirements

- Go 1.21+
- Target devices must be reachable over SSH (port 22)
- Devices must have been connected to with `ssh` at least once so their host key is in `~/.ssh/known_hosts`
