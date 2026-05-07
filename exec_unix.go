//go:build !windows

package main

import (
	"fmt"
	"os/exec"
)

func installExtension(cmdPath, extId string) error {
	cmd := exec.Command(cmdPath, "--install-extension", extId)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("CLI Error: %v | Output: %s", err, string(out))
	}
	return nil
}

func runHiddenPowershell(script string) ([]byte, error) {
	return nil, nil
}