package tui

import (
	"romrepo/internal/config"
	"romrepo/internal/network"
	"romrepo/internal/rom"
)

// Navigation messages
type SelectClientMsg struct {
	Client config.Client
}

type SelectConsoleMsg struct {
	Console config.Console
}

type GoBackMsg struct{}

// Data loading messages
type ROMsLoadedMsg struct {
	ROMs []rom.ROMStatus
}

type ROMsLoadErrorMsg struct {
	Err error
}

// Transfer messages
type TransferStartMsg struct {
	ROMName   string
	Direction string // "push" or "pull"
}

type TransferProgressMsg struct {
	Transferred int64
	Total       int64
}

type TransferCompleteMsg struct {
	Err error
}

// Error messages
type ErrorMsg struct {
	Err error
}

type ClearErrorMsg struct{}

// Config messages
type ConfigUpdatedMsg struct {
	Config *config.Config
}

// SSH connection message
type SSHConnectedMsg struct {
	ClientName string
}

type SSHConnectErrorMsg struct {
	Err error
}

// Screen management
type OpenManageMsg struct{}
type OpenSettingsMsg struct{}

// Network scan messages
type ScanResultMsg struct {
	Devices []network.Device
	Err     error
}
