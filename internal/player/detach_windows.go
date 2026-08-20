//go:build windows

package player

import (
	"os/exec"
)

// setDetached on Windows keeps the default stdio behavior (no session concept).
func setDetached(cmd *exec.Cmd) {}
