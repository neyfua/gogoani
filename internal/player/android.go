package player

import (
	"fmt"
	"os/exec"
	"strings"
)

// Android app component for mpv and vlc.
const (
	mpvActivity = "is.xyz.mpv/.MPVActivity"
	vlcActivity = "org.videolan.vlc/org.videolan.vlc.gui.video.VideoPlayerActivity"
)

// AndroidLauncher launches the video via the Android activity manager (am start)
// instead of a terminal media player binary. This is the recommended launcher on
// Android/Termux, matching ani-cli's android_mpv/android_vlc behavior.
type AndroidLauncher struct {
	activity string // e.g. "is.xyz.mpv/.MPVActivity"
}

// NewAndroidLauncher returns an AndroidLauncher for the given player binary name.
// Defaults to mpv when bin doesn't identify vlc.
func NewAndroidLauncher(bin string) *AndroidLauncher {
	activity := mpvActivity
	if strings.Contains(strings.ToLower(bin), "vlc") {
		activity = vlcActivity
	}
	return &AndroidLauncher{activity: activity}
}

// Start launches the URL in the Android player app. It returns immediately.
func (a *AndroidLauncher) Start(url, referer, title string) error {
	//nolint:gosec // G204: am start with controlled args; url/title come from the scraper, not free user input
	cmd := exec.Command("am", "start", "--user", "0",
		"-a", "android.intent.action.VIEW",
		"-d", url,
		"-n", a.activity,
		"-e", "title", title,
	)
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("player: am start: %w", err)
	}
	return cmd.Process.Release()
}

// Stop is a no-op on Android: the player app manages its own window/process.
func (a *AndroidLauncher) Stop() error {
	return nil
}
