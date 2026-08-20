//go:build !windows

package player

import (
	"os/exec"
	"syscall"
)

// setDetached gives the child its own session and sends stdio to /dev/null,
// mirroring `nohup player ... >/dev/null 2>&1 &` so it does not grab the terminal.
func setDetached(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil
}
