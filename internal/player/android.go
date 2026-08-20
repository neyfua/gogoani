package player

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"

	"github.com/neyfua/gogoani/internal/logger"
)

// Android app component for mpv and vlc.
const (
	mpvActivity = "is.xyz.mpv/.MPVActivity"
	vlcActivity = "org.videolan.vlc/org.videolan.vlc.gui.video.VideoPlayerActivity"
)

// am start flags: FLAG_ACTIVITY_NEW_TASK | FLAG_ACTIVITY_SINGLE_TOP |
// FLAG_ACTIVITY_BROUGHT_TO_FRONT. These make mpv reuse its existing task and
// bring the player to the foreground instead of appending to background playback.
const amForegroundFlags = "0x34000000"

// AndroidLauncher launches the video via the Android activity manager (am start)
// instead of a terminal media player binary. This is the recommended launcher on
// Android/Termux, matching ani-cli's android_mpv/android_vlc behavior.
type AndroidLauncher struct {
	activity string // e.g. "is.xyz.mpv/.MPVActivity"
	pkg      string // e.g. "is.xyz.mpv"
}

// NewAndroidLauncher returns an AndroidLauncher for the given player binary name.
// Defaults to mpv when bin doesn't identify vlc. The player is used through the
// Android app (apk), not a terminal mpv binary.
func NewAndroidLauncher(bin string) *AndroidLauncher {
	activity, pkg := mpvActivity, "is.xyz.mpv"
	if strings.Contains(strings.ToLower(bin), "vlc") {
		activity, pkg = vlcActivity, "org.videolan.vlc"
	}
	return &AndroidLauncher{activity: activity, pkg: pkg}
}

// Start launches the URL in the Android player app. It returns immediately.
// The app is force-stopped first so a stale background instance (with a
// lingering background-playback task) cannot swallow the intent via
// onNewIntent and moveTaskToBack, which would keep playing audio with no
// visible player window.
func (a *AndroidLauncher) Start(url, referer, title string) error {
	a.forceStop()
	args := []string{
		"am", "start", "--user", "0",
		"-a", "android.intent.action.VIEW",
		"-d", url,
		"-t", "video/any",
		"-n", a.activity,
		"-e", "title", title,
		"-f", amForegroundFlags,
	}
	logger.Log.Debug("player: launching android intent", "args", args)

	var stdout, stderr bytes.Buffer
	//nolint:gosec // G204: am start with controlled args; url/title come from the scraper, not free user input
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		logger.Log.Error("player: am start failed", "error", err, "stdout", stdout.String(), "stderr", stderr.String())
		return fmt.Errorf("player: am start failed: %w (stderr: %s)", err, stderr.String())
	}
	logger.Log.Debug("player: am start succeeded", "stdout", stdout.String())
	return nil
}

// Stop force-stops the player app so a previous episode does not keep playing
// audio in the background when the user switches episodes or quits. Best
// effort: if `am` is unavailable it returns nil.
func (a *AndroidLauncher) Stop() error {
	if err := a.forceStop(); err != nil {
		return fmt.Errorf("player: am force-stop: %w", err)
	}
	return nil
}

func (a *AndroidLauncher) forceStop() error {
	if _, err := exec.LookPath("am"); err != nil {
		return nil
	}
	logger.Log.Debug("player: force-stopping app", "pkg", a.pkg)
	//nolint:gosec // G204: package name is a fixed constant, not user input
	cmd := exec.Command("am", "force-stop", a.pkg)
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		logger.Log.Debug("player: am force-stop failed", "error", err)
		return err
	}
	return nil
}
