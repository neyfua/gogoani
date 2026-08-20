package player

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/neyfua/gogoani/internal/logger"
)

const (
	mpvActivity = "is.xyz.mpv/.MPVActivity"
	vlcActivity = "org.videolan.vlc/org.videolan.vlc.gui.video.VideoPlayerActivity"
)

// AndroidLauncher launches video via the Android activity manager.
// It matches ani-cli's fire-and-forget `am start` behavior exactly.
type AndroidLauncher struct {
	activity string
	pkg      string
}

func NewAndroidLauncher(bin string) *AndroidLauncher {
	activity, pkg := mpvActivity, "is.xyz.mpv"
	if strings.Contains(strings.ToLower(bin), "vlc") {
		activity, pkg = vlcActivity, "org.videolan.vlc"
	}
	return &AndroidLauncher{activity: activity, pkg: pkg}
}

// Media returns the display name of the Android media app (mpv or vlc).
func (a *AndroidLauncher) Media() string {
	if strings.Contains(a.activity, "vlc") {
		return "vlc"
	}
	return "mpv"
}

// Start sends an am start intent and returns immediately without waiting for
// the activity to fully render. A leftover mpv instance that is stuck in
// background playback is force-stopped first (also non-blocking) so the new
// intent opens a fresh foreground window instead of being appended and hidden.
func (a *AndroidLauncher) Start(url, referer, title string) error {
	if err := a.hasAm(); err != nil {
		return err
	}
	a.forceStopAsync()

	args := []string{
		"start", "--user", "0",
		"-a", "android.intent.action.VIEW",
		"-d", url,
		"-n", a.activity,
		"-e", "title", title,
	}
	logger.Log.Debug("player: am start", "args", args)

	//nolint:gosec // G204: args are controlled; url/title come from the scraper
	cmd := exec.Command("am", args...)
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("player: am start: %w", err)
	}
	_ = cmd.Process.Release()
	return nil
}

// Stop kills any running mpv instance via am force-stop. Best effort.
func (a *AndroidLauncher) Stop() error {
	if _, err := exec.LookPath("am"); err != nil {
		return nil
	}
	logger.Log.Debug("player: am force-stop", "pkg", a.pkg)
	//nolint:gosec // G204: package is a fixed constant
	cmd := exec.Command("am", "force-stop", a.pkg)
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("player: am force-stop: %w", err)
	}
	return nil
}

func (a *AndroidLauncher) hasAm() error {
	if _, err := exec.LookPath("am"); err != nil {
		return fmt.Errorf("player: Android app launch needs the `am` command, but it's not in PATH. On Termux run: pkg install termux-am")
	}
	return nil
}

// forceStopAsync force-stops the player app without waiting for it to finish.
func (a *AndroidLauncher) forceStopAsync() {
	//nolint:gosec // G204: package is a fixed constant
	cmd := exec.Command("am", "force-stop", a.pkg)
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Start(); err != nil {
		logger.Log.Debug("player: am force-stop start failed", "error", err)
		return
	}
	_ = cmd.Process.Release()
}
