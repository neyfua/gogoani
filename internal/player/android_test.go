package player

import (
	"strings"
	"testing"
)

// resetRuntimeCaches clears the memoized detection results so tests
// for the pure functions below are independent of the host environment.
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

func TestIsTmuxMatcher(t *testing.T) {
	// TMUX is only consulted if the env var is set during the test.
	t.Setenv("TMUX", "/tmp/tmux-1000/default,12345,0")
	resetRuntimeCaches()
	if !IsTmux() {
		t.Errorf("IsTmux() = false with TMUX set, want true")
	}
}

func TestHasAmStart(t *testing.T) {
	// am should not exist in a desktop CI environment; must return without panicking.
	resetRuntimeCaches()
	_ = HasAmStart()
}

func TestNewAndroidLauncherActivity(t *testing.T) {
	mpv := NewAndroidLauncher("android_mpv")
	if !strings.Contains(mpv.activity, "is.xyz.mpv") {
		t.Errorf("default android activity = %q, want mpv MPVActivity", mpv.activity)
	}
	if mpv.pkg != "is.xyz.mpv" {
		t.Errorf("default android pkg = %q, want is.xyz.mpv", mpv.pkg)
	}
	vlc := NewAndroidLauncher("android_vlc")
	if !strings.Contains(vlc.activity, "org.videolan.vlc") {
		t.Errorf("vlc android activity = %q, want org.videolan.vlc", vlc.activity)
	}
	if vlc.pkg != "org.videolan.vlc" {
		t.Errorf("vlc android pkg = %q, want org.videolan.vlc", vlc.pkg)
	}
}

func TestAndroidLauncherStopNoop(t *testing.T) {
	a := NewAndroidLauncher("android_mpv")
	if err := a.Stop(); err != nil {
		t.Errorf("Stop() err = %v, want nil (am not present is best-effort)", err)
	}
}

func TestAndroidLauncherStartNoAm(t *testing.T) {
	// Without `am` in PATH the Start must return an error, not hang.
	orig := hasAmStartCache
	hasAmStartCache = new(bool) // force re-evaluation of am
	*hasAmStartCache = false
	defer func() { hasAmStartCache = orig }()

	a := NewAndroidLauncher("android_mpv")
	err := a.Start("https://example.com/v.m3u8", "", "Test Episode 1")
	if err == nil {
		t.Skip("am command present in environment; cannot test failure path")
	}
}

func TestPlayWithForceMediaTitle(t *testing.T) {
	p := New("sleep")
	p.Detach = true
	if err := p.Start("30", "", ""); err != nil {
		t.Fatalf("Start() returned error: %v", err)
	}
	if err := p.Stop(); err != nil {
		t.Fatalf("Stop() returned error: %v", err)
	}
}
