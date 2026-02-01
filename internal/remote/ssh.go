package remote

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"

	"romrepo/internal/config"
)

type ConnManager struct {
	mu    sync.Mutex
	conns map[string]*ssh.Client
}

func NewConnManager() *ConnManager {
	return &ConnManager{
		conns: make(map[string]*ssh.Client),
	}
}

func (m *ConnManager) Get(client config.Client) (*ssh.Client, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if conn, ok := m.conns[client.Name]; ok {
		// Test if connection is still alive
		_, _, err := conn.SendRequest("keepalive@romrepo", true, nil)
		if err == nil {
			return conn, nil
		}
		conn.Close()
		delete(m.conns, client.Name)
	}

	conn, err := dial(client)
	if err != nil {
		return nil, err
	}

	m.conns[client.Name] = conn
	return conn, nil
}

func (m *ConnManager) CloseAll() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for name, conn := range m.conns {
		conn.Close()
		delete(m.conns, name)
	}
}

func (m *ConnManager) Close(clientName string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if conn, ok := m.conns[clientName]; ok {
		conn.Close()
		delete(m.conns, clientName)
	}
}

func dial(client config.Client) (*ssh.Client, error) {
	var authMethods []ssh.AuthMethod

	switch client.Auth.Method {
	case "key":
		keyPath := client.Auth.KeyPath
		if keyPath == "" {
			home, _ := os.UserHomeDir()
			keyPath = home + "/.ssh/id_rsa"
		}
		key, err := os.ReadFile(keyPath)
		if err != nil {
			return nil, fmt.Errorf("reading SSH key %s: %w", keyPath, err)
		}
		signer, err := ssh.ParsePrivateKey(key)
		if err != nil {
			return nil, fmt.Errorf("parsing SSH key: %w", err)
		}
		authMethods = append(authMethods, ssh.PublicKeys(signer))

	case "password":
		authMethods = append(authMethods, ssh.Password(client.Auth.Password))

	default:
		return nil, fmt.Errorf("unknown auth method: %s", client.Auth.Method)
	}

	port := client.Port
	if port == 0 {
		port = 22
	}

	hostKeyCallback, err := defaultHostKeyCallback()
	if err != nil {
		return nil, fmt.Errorf("loading known_hosts: %w", err)
	}

	sshConfig := &ssh.ClientConfig{
		User:            client.User,
		Auth:            authMethods,
		HostKeyCallback: hostKeyCallback,
	}

	addr := net.JoinHostPort(client.Host, fmt.Sprintf("%d", port))
	conn, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		return nil, fmt.Errorf("connecting to %s: %w", addr, err)
	}

	return conn, nil
}

func defaultHostKeyCallback() (ssh.HostKeyCallback, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("finding home directory: %w", err)
	}
	knownHostsPath := filepath.Join(home, ".ssh", "known_hosts")
	if _, err := os.Stat(knownHostsPath); err != nil {
		return nil, fmt.Errorf("%s not found: %w — connect to the host with ssh first to add it", knownHostsPath, err)
	}
	return knownhosts.New(knownHostsPath)
}
