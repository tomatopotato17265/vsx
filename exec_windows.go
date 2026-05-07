//go:build windows

package main

import (
	"os/exec"
	"syscall"
)

func installExtension(cmdPath, extId string) error {
    cmd := exec.Command("cmd.exe", "/c", cmdPath, "--install-extension", extId)
    cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
    out, err := cmd.CombinedOutput()
    if err != nil {
        return fmt.Errorf("CLI Error: %v | Output: %s", err, string(out))
    }
    return nil
}

func runHiddenPowershell(script string) ([]byte, error) {
	cmd := exec.Command("powershell", "-ExecutionPolicy", "Bypass", "-NoProfile", "-WindowStyle", "Hidden", "-Command", script)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return cmd.Output()
}