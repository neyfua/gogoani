package player

import (
	"os"
	"os/exec"
	"runtime"
	"strings"
)

var (
	isAndroidCache  *bool
	isTmuxCache     *bool
	hasAmStartCache *bool
)

// IsAndroid returns true if running on Android (Termux).
func IsAndroid() bool {
	if isAndroidCache != nil {
		return *isAndroidCache
	}
	v := detectAndroid()
	isAndroidCache = &v
	return v
}

func detectAndroid() bool {
	if runtime.GOOS != "linux" {
		return false
	}
	// Match ani-cli: uname contains "ndroid" (covers "android" and "ndroid")
	if data, err := os.ReadFile("/proc/version"); err == nil {
		return androidFromProcVersion(data)
	}
	return false
}

func androidFromProcVersion(data []byte) bool {
	return strings.Contains(string(data), "ndroid")
}

// IsTmux returns true if running inside a tmux session.
func IsTmux() bool {
	if isTmuxCache != nil {
		return *isTmuxCache
	}
	v := os.Getenv("TMUX") != ""
	isTmuxCache = &v
	return v
}

// HasAmStart returns true if the `am` command (Android activity manager) is available.
func HasAmStart() bool {
	if hasAmStartCache != nil {
		return *hasAmStartCache
	}
	_, err := exec.LookPath("am")
	v := err == nil
	hasAmStartCache = &v
	return v
}
