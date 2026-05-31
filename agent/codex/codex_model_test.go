package codex

import (
	"testing"

	"github.com/chenhg5/cc-connect/core"
)

func TestConfiguredModels_BoundaryConditions(t *testing.T) {
	a := &Agent{
		providers: []core.ProviderConfig{
			{Models: []core.ModelOption{{Name: "first"}}},
			{Models: []core.ModelOption{{Name: "second"}}},
		},
	}

	tests := []struct {
		name      string
		activeIdx int
		wantNil   bool
		wantName  string
	}{
		{name: "negative index", activeIdx: -1, wantNil: true},
		{name: "out of range", activeIdx: 2, wantNil: true},
		{name: "valid index", activeIdx: 1, wantName: "second"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a.activeIdx = tt.activeIdx
			got := a.configuredModels()
			if tt.wantNil {
				if got != nil {
					t.Fatalf("configuredModels() = %v, want nil", got)
				}
				return
			}
			if len(got) != 1 || got[0].Name != tt.wantName {
				t.Fatalf("configuredModels() = %v, want %q", got, tt.wantName)
			}
		})
	}
}

func TestGetModel_PrefersActiveProviderModel(t *testing.T) {
	a := &Agent{
		model: "gpt-4.1-mini",
		providers: []core.ProviderConfig{
			{Name: "openai", Model: "gpt-5.4"},
		},
		activeIdx: 0,
	}

	if got := a.GetModel(); got != "gpt-5.4" {
		t.Fatalf("GetModel() = %q, want gpt-5.4", got)
	}
}

func TestNormalizeAppServerURL_StdIOIsExplicit(t *testing.T) {
	for _, raw := range []string{"stdio", " stdio "} {
		if got := normalizeAppServerURL(raw); got != "stdio://" {
			t.Fatalf("normalizeAppServerURL(%q) = %q, want stdio://", raw, got)
		}
	}
}

func TestNormalizeAppServerURL_EmptyUsesStdIODefault(t *testing.T) {
	if got := normalizeAppServerURL(""); got != "stdio://" {
		t.Fatalf("normalizeAppServerURL(empty) = %q, want stdio://", got)
	}
}

func TestNormalizeBackend_AppAliasesUseAppServer(t *testing.T) {
	for _, raw := range []string{"app", "codex-app"} {
		if got := normalizeBackend(raw); got != "app_server" {
			t.Fatalf("normalizeBackend(%q) = %q, want app_server", raw, got)
		}
	}
}

func TestNormalizeBackend_DesktopAliasesUseDesktopApp(t *testing.T) {
	for _, raw := range []string{"desktop-app", "desktop_app", "desktop"} {
		if got := normalizeBackend(raw); got != "desktop_app" {
			t.Fatalf("normalizeBackend(%q) = %q, want desktop_app", raw, got)
		}
	}
}

func TestDesktopAppURLRequiresWebSocketTransport(t *testing.T) {
	if got := normalizeDesktopAppURL(""); got != "ws://127.0.0.1:3845" {
		t.Fatalf("normalizeDesktopAppURL(empty) = %q, want default ws URL", got)
	}
	if got := normalizeDesktopAppURL("stdio"); got != "stdio://" {
		t.Fatalf("normalizeDesktopAppURL(stdio) = %q, want stdio://", got)
	}
	if isWebSocketURL(normalizeDesktopAppURL("stdio")) {
		t.Fatal("stdio desktop_app URL should not be treated as a reachable Desktop App websocket")
	}
}

func TestDesktopAppURLFallbackIgnoresStdIOAppServer(t *testing.T) {
	if got := desktopAppURLFallback("stdio://"); got != "" {
		t.Fatalf("desktopAppURLFallback(stdio://) = %q, want empty", got)
	}
	if got := desktopAppURLFallback("ws://127.0.0.1:4444"); got != "ws://127.0.0.1:4444" {
		t.Fatalf("desktopAppURLFallback(ws) = %q", got)
	}
}

func TestWorkspaceAgentOptions_PreservesStdIOAppServerURL(t *testing.T) {
	a := &Agent{
		backend:      "app_server",
		appServerURL: normalizeAppServerURL("stdio"),
	}

	opts := a.WorkspaceAgentOptions()
	if got := opts["app_server_url"]; got != "stdio://" {
		t.Fatalf("WorkspaceAgentOptions()[app_server_url] = %#v, want stdio://", got)
	}
}

func TestWorkspaceAgentOptions_IncludesDesktopAppURL(t *testing.T) {
	a := &Agent{
		backend:       "desktop_app",
		desktopAppURL: "ws://127.0.0.1:3845",
	}

	opts := a.WorkspaceAgentOptions()
	if got := opts["backend"]; got != "desktop_app" {
		t.Fatalf("WorkspaceAgentOptions()[backend] = %#v, want desktop_app", got)
	}
	if got := opts["desktop_app_url"]; got != "ws://127.0.0.1:3845" {
		t.Fatalf("WorkspaceAgentOptions()[desktop_app_url] = %#v", got)
	}
}
