package tui

import (
	"romrepo/internal/config"
	"romrepo/internal/network"
	"romrepo/internal/remote"
	"romrepo/internal/rom"
)

// Navigation messages
type SelectClientMsg struct {
	Client config.Client
}

type SelectConsoleMsg struct {
	Console config.Console
}

// Data loading messages
type ROMsLoadedMsg struct {
	ROMs      []rom.ROMStatus
	ClientErr error // non-nil if client connection/listing failed
}

type ROMsLoadErrorMsg struct {
	Err error
}

// Transfer messages
type TransferStartMsg struct {
	ROMNames []string
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

// Password messages
type PasswordNeededMsg struct {
	ClientName string
	Host       string
	User       string
}

type PasswordEnteredMsg struct {
	ClientName string
	Password   string
}

// Overlay messages
type CancelOverlayMsg struct{}

// Network scan messages
type ScanResultMsg struct {
	Devices []network.Device
	Err     error
}

// Directory browser messages
type DirListedMsg struct {
	Path    string
	Entries []remote.FileInfo
	Err     error
}

type DirConnectedMsg struct {
	SFTPClient *remote.SFTPClient
	HomePath   string
}

type DirConnectErrorMsg struct {
	Err error
}
