//go:build windows

package main

import (
	"os"
	"reflect"
	"strconv"
	"testing"
)

func TestRestartCommandUsesConfiguredRestartScript(t *testing.T) {
	t.Setenv("CC_CONNECT_RESTART_SCRIPT", `C:\Users\me\.cc-connect\cc-connect-start.ps1`)

	name, args := restartCommand(`C:\Program Files\cc-connect\cc-connect.exe`)

	if name != "powershell.exe" {
		t.Fatalf("command = %q, want powershell.exe", name)
	}

	want := []string{
		"-WindowStyle", "Hidden",
		"-NoProfile",
		"-NonInteractive",
		"-ExecutionPolicy", "Bypass",
		"-File", `C:\Users\me\.cc-connect\cc-connect-start.ps1`,
		"-RestartFromPid", strconv.Itoa(os.Getpid()),
	}
	if !reflect.DeepEqual(args, want) {
		t.Fatalf("args = %#v, want %#v", args, want)
	}
}

func TestRestartCommandDefaultsToCurrentExecutable(t *testing.T) {
	t.Setenv("CC_CONNECT_RESTART_SCRIPT", "")
	oldArgs := os.Args
	t.Cleanup(func() { os.Args = oldArgs })
	os.Args = []string{"cc-connect.exe", "--config", `C:\Users\me\.cc-connect\config.toml`}

	name, args := restartCommand(`C:\Program Files\cc-connect\cc-connect.exe`)

	if name != `C:\Program Files\cc-connect\cc-connect.exe` {
		t.Fatalf("command = %q, want current executable", name)
	}
	want := []string{"--config", `C:\Users\me\.cc-connect\config.toml`}
	if !reflect.DeepEqual(args, want) {
		t.Fatalf("args = %#v, want %#v", args, want)
	}
}
