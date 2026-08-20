package player

import (
	"testing"
)

func TestNew(t *testing.T) {
	t.Run("default player", func(t *testing.T) {
		p := New("")
		if p.Bin != "mpv" {
			t.Errorf("New('') Bin = %q, want %q", p.Bin, "mpv")
		}
	})

	t.Run("known player binary", func(t *testing.T) {
		p := New("sh")
		// sh should normally be in PATH; resolvePlayer will find it
		if p.Bin == "" {
			t.Errorf("New('sh') returned empty binary path")
		}
	})

	t.Run("unknown player falls back", func(t *testing.T) {
		p := New("nonexistent-player-binary-xyz")
		if p.Bin != "mpv" {
			t.Errorf("New('nonexistent') Bin = %q, want %q", p.Bin, "mpv")
		}
	})
}

func TestPlayNoBin(t *testing.T) {
	p := &Player{Bin: ""}
	err := p.Play("http://example.com/video")
	if err == nil {
		t.Errorf("Play() with empty Bin should return error")
	}
}

func TestStartNoBin(t *testing.T) {
	p := &Player{Bin: ""}
	err := p.Start("http://example.com/video", "", "")
	if err == nil {
		t.Errorf("Start() with empty Bin should return error")
	}
}

func TestStopNoCmd(t *testing.T) {
	p := &Player{Bin: "mpv"}
	err := p.Stop()
	if err != nil {
		t.Errorf("Stop() with no running command should return nil, got %v", err)
	}
}

func TestNewLauncherDesktop(t *testing.T) {
	// On desktop Linux, NewLauncher("sh") returns a *Player (not AndroidLauncher).
	launcher := NewLauncher("sh", true)
	if _, ok := launcher.(*Player); !ok {
		t.Errorf("NewLauncher(\"sh\") returned %T, want *Player", launcher)
	}
}
