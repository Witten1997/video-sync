//go:build !windows

package app

import "syscall"

func RestartProcess(argv0 string, args []string, env []string) error {
	return syscall.Exec(argv0, args, env)
}
