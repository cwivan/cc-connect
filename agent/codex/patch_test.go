package codex

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPatchSessionSource(t *testing.T) {
	tmpDir := t.TempDir()

	sessionID := "test-session-abc123"
	sessionsDir := filepath.Join(tmpDir, ".codex", "sessions")
	if err := os.MkdirAll(sessionsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	fname := filepath.Join(sessionsDir, "rollout-"+sessionID+".jsonl")
	line1 := `{"timestamp":"2026-01-01T00:00:00Z","type":"session_meta","payload":{"id":"` + sessionID + `","source":"exec","originator":"codex_exec","cwd":"/tmp"}}`
	line2 := `{"timestamp":"2026-01-01T00:00:01Z","type":"response_item","payload":{"role":"user"}}`
	content := line1 + "\n" + line2 + "\n"

	if err := os.WriteFile(fname, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	codexHome := filepath.Join(tmpDir, ".codex")

	patchSessionSource(sessionID, codexHome)

	data, err := os.ReadFile(fname)
	if err != nil {
		t.Fatal(err)
	}

	lines := strings.SplitN(string(data), "\n", 2)

	if !strings.Contains(lines[0], `"source":"vscode"`) {
		t.Errorf("expected source:vscode, got first line: %s", lines[0])
	}
	if !strings.Contains(lines[0], `"originator":"Codex Desktop"`) {
		t.Errorf("expected originator:Codex Desktop, got first line: %s", lines[0])
	}
	if strings.Contains(lines[0], `"source":"exec"`) {
		t.Error("source:exec was not replaced")
	}

	// Second line should be untouched
	if !strings.HasPrefix(lines[1], `{"timestamp":"2026-01-01T00:00:01Z"`) {
		t.Errorf("second line was corrupted: %s", lines[1])
	}
}

func TestPatchSessionSource_AppServerSource(t *testing.T) {
	tmpDir := t.TempDir()
	sessionID := "test-app-server-abc123"
	sessionsDir := filepath.Join(tmpDir, ".codex", "sessions")
	if err := os.MkdirAll(sessionsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	fname := filepath.Join(sessionsDir, "rollout-"+sessionID+".jsonl")
	line1 := `{"timestamp":"2026-01-01T00:00:00Z","type":"session_meta","payload":{"id":"` + sessionID + `","source":"app_server","originator":"codex_app_server","cwd":"/tmp"}}`
	line2 := `{"timestamp":"2026-01-01T00:00:01Z","type":"response_item","payload":{"role":"user"}}`
	if err := os.WriteFile(fname, []byte(line1+"\n"+line2+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	patchSessionSource(sessionID, filepath.Join(tmpDir, ".codex"))

	data, err := os.ReadFile(fname)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if !strings.Contains(lines[0], `"source":"vscode"`) {
		t.Errorf("expected source:vscode, got first line: %s", lines[0])
	}
	if !strings.Contains(lines[0], `"originator":"Codex Desktop"`) {
		t.Errorf("expected originator:Codex Desktop, got first line: %s", lines[0])
	}
	if strings.Contains(lines[0], `"source":"app_server"`) {
		t.Error("source:app_server was not replaced")
	}
	if lines[1] != line2 {
		t.Errorf("second line changed: %s", lines[1])
	}
}

func TestPatchSessionSource_Idempotent(t *testing.T) {
	tmpDir := t.TempDir()
	sessionID := "test-idempotent-xyz"
	sessionsDir := filepath.Join(tmpDir, ".codex", "sessions")
	os.MkdirAll(sessionsDir, 0o755)

	fname := filepath.Join(sessionsDir, "rollout-"+sessionID+".jsonl")
	line1 := `{"type":"session_meta","payload":{"id":"` + sessionID + `","source":"vscode","originator":"Codex Desktop"}}`
	if err := os.WriteFile(fname, []byte(line1+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	codexHome := filepath.Join(tmpDir, ".codex")

	patchSessionSource(sessionID, codexHome)

	data, _ := os.ReadFile(fname)
	if string(data) != line1+"\n" {
		t.Errorf("file was modified when it shouldn't have been")
	}
}

func TestPatchSessionSource_LeavesCLISourceUntouched(t *testing.T) {
	tmpDir := t.TempDir()
	sessionID := "test-cli-source-xyz"
	sessionsDir := filepath.Join(tmpDir, ".codex", "sessions")
	os.MkdirAll(sessionsDir, 0o755)

	fname := filepath.Join(sessionsDir, "rollout-"+sessionID+".jsonl")
	line1 := `{"type":"session_meta","payload":{"id":"` + sessionID + `","source":"cli","originator":"codex_cli_rs"}}`
	if err := os.WriteFile(fname, []byte(line1+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	patchSessionSource(sessionID, filepath.Join(tmpDir, ".codex"))

	data, _ := os.ReadFile(fname)
	if string(data) != line1+"\n" {
		t.Errorf("CLI session was modified: %s", string(data))
	}
}
