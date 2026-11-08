package player

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
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
	err := p.Start("http://example.com/video", "", "", "")
	if err == nil {
		t.Errorf("Start() with empty Bin should return error")
	}
}

func TestStartArgs(t *testing.T) {
	const (
		video    = "https://example.com/video.m3u8?token=a&b=c"
		referer  = "https://example.com/watch"
		title    = "Test Episode 1"
		subtitle = "https://example.com/English subtitles.vtt?token=a&b=c"
	)
	for _, tt := range []struct {
		name     string
		bin      string
		subtitle string
		want     []string
	}{
		{"mpv subtitles", "mpv", subtitle, []string{video, "--referrer=" + referer, "--force-media-title=" + title, "--sub-file=" + subtitle, "--sid=auto", "--sub-visibility=yes"}},
		{"mpv no subtitles", "mpv", "", []string{video, "--referrer=" + referer, "--force-media-title=" + title}},
		{"vlc subtitles", "vlc", subtitle, []string{video, "--http-referrer", referer, "--meta-title", title, "--sub-file", subtitle}},
		{"vlc no subtitles", "vlc", "", []string{video, "--http-referrer", referer, "--meta-title", title}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			output := filepath.Join(dir, "args")
			t.Setenv("PLAYER_TEST_ARGS", output)
			bin := filepath.Join(dir, tt.bin)
			if err := os.WriteFile(bin, []byte("#!/bin/sh\nprintf '%s\\n' \"$@\" > \"$PLAYER_TEST_ARGS\"\n"), 0o700); err != nil {
				t.Fatal(err)
			}
			p := &Player{Bin: bin}
			if err := p.Start(video, referer, title, tt.subtitle); err != nil {
				t.Fatal(err)
			}
			if err := p.cmd.Wait(); err != nil {
				t.Fatal(err)
			}
			data, err := os.ReadFile(output)
			if err != nil {
				t.Fatal(err)
			}
			got := strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Start args = %q, want %q", got, tt.want)
			}
		})
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
