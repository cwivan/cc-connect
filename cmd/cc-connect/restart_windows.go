//go:build windows

package main

import (
	"os"
	"os/exec"
	"strconv"
	"strings"
)

func restartProcess(execPath string) error {
	name, args := restartCommand(execPath)
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.Env = os.Environ()
	return cmd.Start()
}

func restartCommand(execPath string) (string, []string) {
	if script := strings.TrimSpace(os.Getenv("CC_CONNECT_RESTART_SCRIPT")); script != "" {
		return "powershell.exe", []string{
			"-WindowStyle", "Hidden",
			"-NoProfile",
			"-NonInteractive",
			"-ExecutionPolicy", "Bypass",
			"-File", script,
			"-RestartFromPid", strconv.Itoa(os.Getpid()),
		}
	}
	return execPath, os.Args[1:]
}
