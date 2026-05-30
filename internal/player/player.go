package player

import (
	"fmt"
	"os/exec"
)

// Player launches an external media player.
type Player struct {
	Bin string // e.g. "mpv", "vlc"
	cmd *exec.Cmd
}

func New(bin string) *Player {
	return &Player{Bin: bin}
}

// Play opens the given URL in the media player with optional extra arguments.
func (p *Player) Play(url string, args ...string) error {
	cmdArgs := append([]string{url}, args...)
	cmd := exec.Command(p.Bin, cmdArgs...)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("player: %w", err)
	}
	return cmd.Wait()
}

// Start launches the player without blocking.
func (p *Player) Start(url string, args ...string) error {
	cmdArgs := append([]string{url}, args...)
	p.cmd = exec.Command(p.Bin, cmdArgs...)
	if err := p.cmd.Start(); err != nil {
		return fmt.Errorf("player: %w", err)
	}
	return nil
}

// Stop kills the currently running player process.
func (p *Player) Stop() error {
	if p.cmd != nil && p.cmd.Process != nil {
		return p.cmd.Process.Kill()
	}
	return nil
}
