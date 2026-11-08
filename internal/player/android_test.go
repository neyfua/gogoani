package player

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func resetRuntimeCaches() {
	isAndroidCache = nil
	isTmuxCache = nil
	hasAmStartCache = nil
}

func TestDetectAndroid(t *testing.T) {
	tests := []struct {
		name string
		data string
		want bool
	}{
		{"empty", "", false},
		{"desktop kernel", "Linux version 6.8.0 (gcc)", false},
		{"termux android", "Linux version 4.14.232 (Android rk30board)", true},
		{"android substring", "Linux version 5.10 (Android 12)", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := androidFromProcVersion([]byte(tt.data)); got != tt.want {
				t.Errorf("androidFromProcVersion(%q) = %v, want %v", tt.data, got, tt.want)
			}
		})
	}
}

func TestDetectAndroidTermuxEnv(t *testing.T) {
	resetRuntimeCaches()
	t.Setenv("PREFIX", "/data/data/com.termux/files/usr")
	if !detectAndroid() {
		t.Errorf("detectAndroid() = false with PREFIX set")
	}

	resetRuntimeCaches()
	t.Setenv("PREFIX", "")
	t.Setenv("TERMUX_VERSION", "0.118.0")
	if !detectAndroid() {
		t.Errorf("detectAndroid() = false with TERMUX_VERSION set")
	}

	resetRuntimeCaches()
	t.Setenv("TERMUX_VERSION", "")
	t.Setenv("ANDROID_ROOT", "/system")
	if !detectAndroid() {
		t.Errorf("detectAndroid() = false with ANDROID_ROOT set")
	}
}

func TestIsTmuxMatcher(t *testing.T) {
	t.Setenv("TMUX", "/tmp/tmux-1000/default,12345,0")
	resetRuntimeCaches()
	if !IsTmux() {
		t.Errorf("IsTmux() = false with TMUX set, want true")
	}
}

func TestHasAmStart(t *testing.T) {
	resetRuntimeCaches()
	_ = HasAmStart()
}

func TestNewAndroidLauncherActivity(t *testing.T) {
	mpv := NewAndroidLauncher("android_mpv")
	if !strings.Contains(mpv.activity, "is.xyz.mpv") {
		t.Errorf("mpv activity = %q", mpv.activity)
	}
	if mpv.pkg != "is.xyz.mpv" {
		t.Errorf("mpv pkg = %q", mpv.pkg)
	}

	vlc := NewAndroidLauncher("android_vlc")
	if !strings.Contains(vlc.activity, "org.videolan.vlc") {
		t.Errorf("vlc activity = %q", vlc.activity)
	}
	if vlc.pkg != "org.videolan.vlc" {
		t.Errorf("vlc pkg = %q", vlc.pkg)
	}

	defaultLauncher := NewAndroidLauncher("mpv")
	if defaultLauncher.pkg != "is.xyz.mpv" {
		t.Errorf("default launcher pkg = %q, want is.xyz.mpv", defaultLauncher.pkg)
	}
}

func TestAndroidLauncherStartNoAm(t *testing.T) {
	a := NewAndroidLauncher("android_mpv")
	err := a.Start("https://example.com/v.m3u8", "", "Test Episode 1", "")
	if err == nil {
		t.Skip("am present in environment")
	}
}

func TestWriteAndroidMpvConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mpv", "mpv.config.mp4")
	if err := writeAndroidMpvConfig(path, "https://example.com/watch", "https://example.com/subs.vtt"); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := "referrer=https://example.com/watch\nsub-file=https://example.com/subs.vtt\n"
	if string(got) != want {
		t.Fatalf("config = %q, want %q", got, want)
	}

	if err := writeAndroidMpvConfig(path, "https://example.com/next", ""); err != nil {
		t.Fatal(err)
	}
	got, err = os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if want := "referrer=https://example.com/next\n"; string(got) != want {
		t.Errorf("overwritten config = %q, want %q", got, want)
	}
}

func TestWriteAndroidMpvConfigErrors(t *testing.T) {
	t.Run("mkdir failure", func(t *testing.T) {
		root := t.TempDir()
		file := filepath.Join(root, "not-a-dir")
		if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := writeAndroidMpvConfig(filepath.Join(file, "config"), "", ""); err == nil {
			t.Fatal("writeAndroidMpvConfig() error = nil, want mkdir error")
		}
	})
	t.Run("write failure", func(t *testing.T) {
		dir := t.TempDir()
		if err := writeAndroidMpvConfig(dir, "", ""); err == nil {
			t.Fatal("writeAndroidMpvConfig() error = nil, want write error for directory path")
		}
	})
}

func TestPlayWithForceMediaTitle(t *testing.T) {
	p := New("sleep")
	p.Detach = true
	if err := p.Start("30", "", "", ""); err != nil {
		t.Fatalf("Start() error: %v", err)
	}
	defer func() { _ = p.cmd.Wait() }()
	if err := p.Stop(); err != nil {
		t.Fatalf("Stop() error: %v", err)
	}
}

func TestNewLauncherAndroid(t *testing.T) {
	// Explicit android_mpv should always return AndroidLauncher
	launcher := NewLauncher("android_mpv", true)
	if _, ok := launcher.(*AndroidLauncher); !ok {
		t.Errorf("NewLauncher(\"android_mpv\") = %T, want *AndroidLauncher", launcher)
	}
}
