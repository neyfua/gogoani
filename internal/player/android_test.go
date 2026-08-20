package player

import (
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
	err := a.Start("https://example.com/v.m3u8", "", "Test Episode 1")
	if err == nil {
		t.Skip("am present in environment")
	}
}

func TestPlayWithForceMediaTitle(t *testing.T) {
	p := New("sleep")
	p.Detach = true
	if err := p.Start("30", "", ""); err != nil {
		t.Fatalf("Start() error: %v", err)
	}
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
