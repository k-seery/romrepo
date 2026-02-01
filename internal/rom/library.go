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

func ListServerROMs(cfg *config.Config, console config.Console) ([]ROMFile, error) {
	dir := filepath.Join(cfg.Server.ROMDir, console.Dir)

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	extSet := make(map[string]bool, len(console.Extensions))
	for _, ext := range console.Extensions {
		extSet[strings.ToLower(ext)] = true
	}

	var roms []ROMFile
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))
		if !extSet[ext] {
			continue
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
