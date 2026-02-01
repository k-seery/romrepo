package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server  ServerConfig `yaml:"server"`
	Clients []Client     `yaml:"clients"`
}

type ServerConfig struct {
	ROMDir   string    `yaml:"rom_dir"`
	Consoles []Console `yaml:"consoles"`
}

type Console struct {
	Name       string   `yaml:"name"`
	Dir        string   `yaml:"dir"`
	Extensions []string `yaml:"extensions"`
}

type Client struct {
	Name       string            `yaml:"name"`
	Host       string            `yaml:"host"`
	Port       int               `yaml:"port"`
	User       string            `yaml:"user"`
	Auth       AuthConfig        `yaml:"auth"`
	ROMDir     string            `yaml:"rom_dir"`
	ConsoleDirs map[string]string `yaml:"console_dirs,omitempty"`
}

type AuthConfig struct {
	Method     string `yaml:"method"` // "key" or "password"
	KeyPath    string `yaml:"key_path,omitempty"`
	Password   string `yaml:"password,omitempty"`
}

func DefaultConfig() *Config {
	home, _ := os.UserHomeDir()
	return &Config{
		Server: ServerConfig{
			ROMDir: filepath.Join(home, "roms"),
			Consoles: []Console{
				{Name: "NES", Dir: "nes", Extensions: []string{".nes", ".zip"}},
				{Name: "SNES", Dir: "snes", Extensions: []string{".sfc", ".smc", ".zip"}},
				{Name: "Game Boy", Dir: "gb", Extensions: []string{".gb", ".gbc", ".zip"}},
				{Name: "Game Boy Advance", Dir: "gba", Extensions: []string{".gba", ".zip"}},
				{Name: "Nintendo 64", Dir: "n64", Extensions: []string{".n64", ".z64", ".v64", ".zip"}},
				{Name: "Genesis", Dir: "genesis", Extensions: []string{".md", ".bin", ".zip"}},
				{Name: "PlayStation", Dir: "psx", Extensions: []string{".bin", ".cue", ".iso", ".chd", ".zip"}},
			},
		},
		Clients: []Client{
			{
				Name: "example-device",
				Host: "192.168.1.100",
				Port: 22,
				User: "pi",
				Auth: AuthConfig{Method: "key", KeyPath: filepath.Join(home, ".ssh", "id_rsa")},
				ROMDir: "/home/pi/RetroPie/roms",
			},
		},
	}
}

func defaultConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "romrepo", "config.yaml")
}

func Load(path string) (*Config, error) {
	if path == "" {
		path = defaultConfigPath()
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			cfg := DefaultConfig()
			if saveErr := Save(cfg, path); saveErr != nil {
				return nil, fmt.Errorf("creating default config: %w", saveErr)
			}
			return cfg, nil
		}
		return nil, fmt.Errorf("reading config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	if err := Validate(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func Save(cfg *Config, path string) error {
	if path == "" {
		path = defaultConfigPath()
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}
	return nil
}

func Validate(cfg *Config) error {
	if cfg.Server.ROMDir == "" {
		return fmt.Errorf("server.rom_dir is required")
	}
	if len(cfg.Server.Consoles) == 0 {
		return fmt.Errorf("at least one console must be configured")
	}
	for i, c := range cfg.Clients {
		if c.Name == "" {
			return fmt.Errorf("client[%d].name is required", i)
		}
		if c.Host == "" {
			return fmt.Errorf("client[%d].host is required", i)
		}
		if c.Port == 0 {
			cfg.Clients[i].Port = 22
		}
		if c.User == "" {
			return fmt.Errorf("client[%d].user is required", i)
		}
	}
	return nil
}
