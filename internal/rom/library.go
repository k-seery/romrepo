package rom

import (
	"os"
	"path/filepath"
	"strings"

	"romrepo/internal/config"
)

type ROMFile struct {
	Name string
	Size int64
	Path string
}

// DiscoverConsoles scans the server ROM directory for subdirectories that
// contain at least one file. Known consoles from config keep their extensions;
// unknown directories get an empty Extensions list (meaning all files).
func DiscoverConsoles(cfg *config.Config) []config.Console {
	entries, err := os.ReadDir(cfg.Server.ROMDir)
	if err != nil {
		return nil
	}

	configMap := make(map[string]config.Console)
	for _, c := range cfg.Server.Consoles {
		configMap[c.Dir] = c
	}

	var consoles []config.Console
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dirPath := filepath.Join(cfg.Server.ROMDir, e.Name())
		if !dirHasFiles(dirPath) {
			continue
		}

		if known, ok := configMap[e.Name()]; ok {
			consoles = append(consoles, known)
		} else {
			consoles = append(consoles, config.Console{
				Name: e.Name(),
				Dir:  e.Name(),
			})
		}
	}
	return consoles
}

func ListServerROMs(cfg *config.Config, console config.Console) ([]ROMFile, error) {
	dir := filepath.Join(cfg.Server.ROMDir, console.Dir)

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	// If no extensions configured, include all files
	includeAll := len(console.Extensions) == 0

	extSet := make(map[string]bool, len(console.Extensions))
	for _, ext := range console.Extensions {
		extSet[strings.ToLower(ext)] = true
	}

	var roms []ROMFile
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if !includeAll {
			ext := strings.ToLower(filepath.Ext(e.Name()))
			if !extSet[ext] {
				continue
			}
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		roms = append(roms, ROMFile{
			Name: e.Name(),
			Size: info.Size(),
			Path: filepath.Join(dir, e.Name()),
		})
	}
	return roms, nil
}

// dirHasFiles returns true if the directory contains at least one non-directory entry.
func dirHasFiles(path string) bool {
	entries, err := os.ReadDir(path)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if !e.IsDir() {
			return true
		}
	}
	return false
}
