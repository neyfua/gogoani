package player

import (
	"fmt"
	"os/exec"
	"sync"
)

// Player launches an external media player.
type Player struct {
	Bin string // e.g. "mpv", "vlc"
	cmd *exec.Cmd
	mu  sync.Mutex
}

func New(bin string) *Player {
	return &Player{Bin: bin}
}

// Play opens the given URL in the media player with optional extra arguments.
func (p *Player) Play(url string, args ...string) error {
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
