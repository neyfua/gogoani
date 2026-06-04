package player

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

// Player launches an external media player.
type Player struct {
	Bin string // e.g. "mpv", "vlc"
	cmd *exec.Cmd
	mu  sync.Mutex
}

func New(bin string) *Player {
	bin = resolvePlayer(bin)
	return &Player{Bin: bin}
}

func resolvePlayer(bin string) string {
	if bin == "" {
		return "mpv"
	}
	if strings.Contains(bin, string(filepath.Separator)) {
		fi, err := os.Stat(bin)
		if err != nil || fi.IsDir() || fi.Mode()&0o111 == 0 {
			return "mpv"
		}
		return bin
	}
	path, err := exec.LookPath(bin)
	if err != nil {
		return "mpv"
	}
	return path
}

// Play opens the given URL in the media player with optional extra arguments.
func (p *Player) Play(url string, args ...string) error {
	if p.Bin == "" {
		return fmt.Errorf("player: no binary configured")
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	cmdArgs := append([]string{url}, args...)
	cmd := exec.Command(p.Bin, cmdArgs...)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("player: %w", err)
	}
	return cmd.Wait()
}

// Start launches the player without blocking.
//
//nolint:gosec // G204: Player binary path is from user env var or config, verified at startup
func (p *Player) Start(url string, args ...string) error {
	if p.Bin == "" {
		return fmt.Errorf("player: no binary configured")
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	cmdArgs := append([]string{url}, args...)
	p.cmd = exec.Command(p.Bin, cmdArgs...)
	if err := p.cmd.Start(); err != nil {
		return fmt.Errorf("player: %w", err)
	}
	return nil
}

// Stop kills the currently running player process.
func (p *Player) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.cmd != nil && p.cmd.Process != nil {
		if err := p.cmd.Process.Kill(); err != nil {
			return fmt.Errorf("player stop: %w", err)
		}
	}
	return nil
}
