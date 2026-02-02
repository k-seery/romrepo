package remote

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

type SFTPClient struct {
	client *sftp.Client
}

type FileInfo struct {
	Name string
	Size int64
	IsDir bool
}

type ProgressFunc func(transferred, total int64)

func NewSFTPClient(sshConn *ssh.Client) (*SFTPClient, error) {
	client, err := sftp.NewClient(sshConn)
	if err != nil {
		return nil, fmt.Errorf("creating SFTP client: %w", err)
	}
	return &SFTPClient{client: client}, nil
}

func (s *SFTPClient) Close() error {
	return s.client.Close()
}

func (s *SFTPClient) ListFiles(dir string) ([]FileInfo, error) {
	entries, err := s.client.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("listing %s: %w", dir, err)
	}

	var files []FileInfo
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		files = append(files, FileInfo{
			Name:  e.Name(),
			Size:  e.Size(),
			IsDir: false,
		})
	}
	return files, nil
}

func (s *SFTPClient) ListDir(dir string) ([]FileInfo, error) {
	entries, err := s.client.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("listing %s: %w", dir, err)
	}

	var dirs, files []FileInfo
	for _, e := range entries {
		info := FileInfo{
			Name:  e.Name(),
			Size:  e.Size(),
			IsDir: e.IsDir(),
		}
		if e.IsDir() {
			dirs = append(dirs, info)
		} else {
			files = append(files, info)
		}
	}

	sort.Slice(dirs, func(i, j int) bool { return dirs[i].Name < dirs[j].Name })
	sort.Slice(files, func(i, j int) bool { return files[i].Name < files[j].Name })

	return append(dirs, files...), nil
}

func (s *SFTPClient) HomePath() (string, error) {
	wd, err := s.client.Getwd()
	if err != nil {
		return "", fmt.Errorf("getting home directory: %w", err)
	}
	return wd, nil
}

func (s *SFTPClient) FileExists(path string) bool {
	_, err := s.client.Stat(path)
	return err == nil
}

func (s *SFTPClient) Push(localPath, remotePath string, progress ProgressFunc) error {
	localFile, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("opening local file: %w", err)
	}
	defer localFile.Close()

	info, err := localFile.Stat()
	if err != nil {
		return fmt.Errorf("stat local file: %w", err)
	}
	totalSize := info.Size()

	// Ensure remote directory exists
	remoteDir := filepath.Dir(remotePath)
	s.client.MkdirAll(remoteDir)

	remoteFile, err := s.client.Create(remotePath)
	if err != nil {
		return fmt.Errorf("creating remote file: %w", err)
	}
	defer remoteFile.Close()

	return copyWithProgress(localFile, remoteFile, totalSize, progress)
}

func (s *SFTPClient) Pull(remotePath, localPath string, progress ProgressFunc) error {
	remoteFile, err := s.client.Open(remotePath)
	if err != nil {
		return fmt.Errorf("opening remote file: %w", err)
	}
	defer remoteFile.Close()

	info, err := remoteFile.Stat()
	if err != nil {
		return fmt.Errorf("stat remote file: %w", err)
	}
	totalSize := info.Size()

	// Ensure local directory exists
	localDir := filepath.Dir(localPath)
	if err := os.MkdirAll(localDir, 0o755); err != nil {
		return fmt.Errorf("creating local directory: %w", err)
	}

	localFile, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("creating local file: %w", err)
	}
	defer localFile.Close()

	return copyWithProgress(remoteFile, localFile, totalSize, progress)
}

func copyWithProgress(src io.Reader, dst io.Writer, total int64, progress ProgressFunc) error {
	buf := make([]byte, 32*1024)
	var transferred int64

	for {
		n, err := src.Read(buf)
		if n > 0 {
			written, wErr := dst.Write(buf[:n])
			if wErr != nil {
				return fmt.Errorf("write error: %w", wErr)
			}
			transferred += int64(written)
			if progress != nil {
				progress(transferred, total)
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("read error: %w", err)
		}
	}
	return nil
}
