package remote

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
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

	hostKeyCallback, hostKeyAlgorithms, err := defaultHostKeyCallback()
	if err != nil {
		return nil, fmt.Errorf("loading known_hosts: %w", err)
	}

	addr := net.JoinHostPort(client.Host, fmt.Sprintf("%d", port))

	sshConfig := &ssh.ClientConfig{
		User:              client.User,
		Auth:              authMethods,
		HostKeyCallback:   hostKeyCallback,
		HostKeyAlgorithms: hostKeyAlgorithms(addr),
	}

	conn, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		return nil, fmt.Errorf("connecting to %s: %w", addr, err)
	}

	return conn, nil
}

func defaultHostKeyCallback() (ssh.HostKeyCallback, func(string) []string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, nil, fmt.Errorf("finding home directory: %w", err)
	}
	knownHostsPath := filepath.Join(home, ".ssh", "known_hosts")
	if _, err := os.Stat(knownHostsPath); err != nil {
		return nil, nil, fmt.Errorf("%s not found: %w — connect to the host with ssh first to add it", knownHostsPath, err)
	}
	cb, err := knownhosts.New(knownHostsPath)
	if err != nil {
		return nil, nil, err
	}
	algosFor := hostKeyAlgorithmsFromFile(knownHostsPath)
	return cb, algosFor, nil
}

// hostKeyAlgorithmsFromFile returns a function that, given an address,
// returns the host key algorithms present in the known_hosts file for
// that host. This constrains the SSH handshake to only negotiate key
// types we can verify, avoiding false "key mismatch" errors.
func hostKeyAlgorithmsFromFile(path string) func(string) []string {
	return func(addr string) []string {
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			host = addr
			port = "22"
		}

		f, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer f.Close()

		var algos []string
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}

			// Try to parse as a known_hosts line: markers are optional
			_, hosts, keyType, _, _, err := parseKnownHostsLine(line)
			if err != nil {
				continue
			}

			if matchesHost(hosts, host, port) {
				algos = append(algos, keyType)
			}
		}
		return algos
	}
}

func parseKnownHostsLine(line string) (marker, hosts, keyType, key, comment string, err error) {
	fields := strings.Fields(line)
	if len(fields) < 3 {
		return "", "", "", "", "", fmt.Errorf("too few fields")
	}
	idx := 0
	if strings.HasPrefix(fields[0], "@") {
		marker = fields[0]
		idx++
	}
	hosts = fields[idx]
	keyType = fields[idx+1]
	key = fields[idx+2]
	if len(fields) > idx+3 {
		comment = fields[idx+3]
	}
	return
}

func matchesHost(hostsField, targetHost, targetPort string) bool {
	normalized := knownhosts.Normalize(net.JoinHostPort(targetHost, targetPort))
	for _, h := range strings.Split(hostsField, ",") {
		// Hashed entries start with |1|
		if strings.HasPrefix(h, "|1|") {
			// Can't match hashed entries by inspection; rely on the
			// knownhosts callback itself for verification. Include all
			// algorithms from hashed entries as candidates.
			return true
		}
		if knownhosts.Normalize(h) == normalized {
			return true
		}
	}
	return false
}
