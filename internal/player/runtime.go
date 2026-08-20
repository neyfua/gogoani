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
	// Termux sets PREFIX and TERMUX_VERSION.
	if os.Getenv("PREFIX") != "" || os.Getenv("TERMUX_VERSION") != "" {
		return true
	}
	// Android sets ANDROID_ROOT.
	if os.Getenv("ANDROID_ROOT") != "" {
		return true
	}
	if data, err := os.ReadFile("/proc/version"); err == nil {
		return androidFromProcVersion(data)
	}
	// Android exposes /system/build.prop; a desktop Linux box does not.
	if _, err := os.Stat("/system/build.prop"); err == nil {
		return true
	}
	// Match ani-cli's uname check (`*ndroid*`).
	if out, err := exec.Command("uname", "-a").Output(); err == nil {
		return strings.Contains(string(out), "ndroid")
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
