//go:build windows

package app

import (
	"os"
	"os/exec"
)

func RestartProcess(argv0 string, args []string, env []string) error {
	cmd := exec.Command(argv0, args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = env
	if err := cmd.Start(); err != nil {
		return err
	}
	os.Exit(0)
	return nil
}
