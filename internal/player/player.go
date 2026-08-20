package player

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

// Launcher starts a video player and can stop it. Implementations are the
// terminal player (Player) and the Android activity-manager launcher.
type Launcher interface {
	Start(url, referer, title string) error
	Stop() error
}

// Player launches an external media player.
type Player struct {
	Bin    string // e.g. "mpv", "vlc"
	Detach bool   // detach from the terminal (setsid + /dev/null stdio)
	cmd    *exec.Cmd
	mu     sync.Mutex
}

func New(bin string) *Player {
	bin = resolvePlayer(bin)
	return &Player{Bin: bin, Detach: true}
}

// NewLauncher returns the right launcher for the environment. On Android/Termux
// it returns an AndroidLauncher that opens the video via the mpv/vlc app;
// otherwise it returns a terminal Player.
func NewLauncher(bin string, detach bool) Launcher {
	if isAndroidPreferred(bin) {
		return NewAndroidLauncher(bin)
	}
	p := New(bin)
	p.Detach = detach
	return p
}

// isAndroidPreferred reports whether the Android am-start launcher should be used.
func isAndroidPreferred(bin string) bool {
	if strings.Contains(strings.ToLower(bin), "android") {
		return true
	}
	return IsAndroid() && HasAmStart()
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

// Start launches the player without blocking. When Detach is set the player is
// fully detached from the terminal (own session, stdio to /dev/null), which is
// safe inside tmux; the referer and title are passed to the player.
//
//nolint:gosec // G204: Player binary path is from user env var or config, verified at startup
func (p *Player) Start(url, referer, title string) error {
	if p.Bin == "" {
		return fmt.Errorf("player: no binary configured")
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	cmdArgs := make([]string, 0, 4)
	cmdArgs = append(cmdArgs, url)
	if referer != "" {
		cmdArgs = append(cmdArgs, "--referrer="+referer)
	}
	if title != "" {
		cmdArgs = append(cmdArgs, "--force-media-title="+title)
	}
	cmd := exec.Command(p.Bin, cmdArgs...)
	if p.Detach {
		setDetached(cmd)
	}
	p.cmd = cmd
	if err := cmd.Start(); err != nil {
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
