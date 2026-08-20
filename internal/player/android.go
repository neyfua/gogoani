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
// It fires-and-forgets an `am start` intent matching exactly what ani-cli
// does (no extra flags, no blocking wait, no force-stop before launch).
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

// Start sends an am start intent and returns immediately without waiting for
// the activity to fully render. This matches ani-cli's `nohup am start ... &`
// approach: the intent is fire-and-forget so the activity manager can bring
// mpv to the foreground without blocking gogoani's terminal.
func (a *AndroidLauncher) Start(url, referer, title string) error {
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
	// Detach: don't wait for am to finish. The process will exit on its own
	// after sending the intent. Releasing avoids a zombie without blocking.
	_ = cmd.Process.Release()
	return nil
}

// Stop kills any running mpv instance via am force-stop. Best effort.
func (a *AndroidLauncher) Stop() error {
	//nolint:gosec // G204: package is a fixed constant
	cmd := exec.Command("am", "force-stop", a.pkg)
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		logger.Log.Debug("player: am force-stop failed", "error", err)
		return fmt.Errorf("player: am force-stop: %w", err)
	}
	return nil
}
