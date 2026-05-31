package codex

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/chenhg5/cc-connect/core"
)

type fakeAppServerThreadCall struct {
	method string
	params map[string]any
}

type fakeAppServerThreadClient struct {
	listResponse  appServerThreadListResponse
	readResponse  appServerThreadReadResponse
	startResponse threadStartResponse
	calls         []fakeAppServerThreadCall
	closed        bool
}

func (f *fakeAppServerThreadClient) request(method string, params any, out any) error {
	call := fakeAppServerThreadCall{method: method}
	if m, ok := params.(map[string]any); ok {
		call.params = m
	}
	f.calls = append(f.calls, call)

	switch resp := out.(type) {
	case *appServerThreadListResponse:
		*resp = f.listResponse
	case *appServerThreadReadResponse:
		*resp = f.readResponse
	case *threadStartResponse:
		*resp = f.startResponse
	}
	return nil
}

func (f *fakeAppServerThreadClient) Close() error {
	f.closed = true
	return nil
}

func withManagedAppServerThreadClient(t *testing.T, client *fakeAppServerThreadClient) *appServerRPCConfig {
	t.Helper()
	old := newManagedAppServerThreadClient
	captured := &appServerRPCConfig{}
	newManagedAppServerThreadClient = func(_ context.Context, _ string, _ string, _ []string, cfg appServerRPCConfig) (appServerRPC, error) {
		*captured = cfg
		return client, nil
	}
	t.Cleanup(func() { newManagedAppServerThreadClient = old })
	return captured
}

func TestListSessions_AppServerUsesThreadList(t *testing.T) {
	client := &fakeAppServerThreadClient{
		listResponse: appServerThreadListResponse{
			Data: []appServerThread{{
				ID:        "thread-app",
				Name:      "Mobile title",
				Preview:   "last message",
				UpdatedAt: time.Unix(1700000000, 0).Unix(),
			}},
		},
	}
	withManagedAppServerThreadClient(t, client)

	a := &Agent{backend: "app_server", appServerURL: "stdio://", workDir: t.TempDir()}
	got, err := a.ListSessions(context.Background())
	if err != nil {
		t.Fatalf("ListSessions() returned error: %v", err)
	}
	if len(got) != 1 || got[0].ID != "thread-app" || got[0].Summary != "Mobile title" {
		t.Fatalf("ListSessions() = %#v", got)
	}
	if len(client.calls) != 1 || client.calls[0].method != "thread/list" {
		t.Fatalf("calls = %#v, want thread/list", client.calls)
	}
	if client.calls[0].params["cwd"] == "" {
		t.Fatalf("thread/list params missing cwd: %#v", client.calls[0].params)
	}
	if !client.closed {
		t.Fatal("client was not closed")
	}
}

func TestGetSessionHistory_AppServerUsesThreadRead(t *testing.T) {
	startedAt := int64(1700000000)
	client := &fakeAppServerThreadClient{
		readResponse: appServerThreadReadResponse{
			Thread: appServerThread{
				Turns: []appServerTurn{{
					StartedAt: &startedAt,
					Items: []map[string]any{
						{"type": "userMessage", "content": []any{map[string]any{"type": "text", "text": "hello"}}},
						{"type": "reasoning", "content": []any{"private chain"}},
						{"type": "agentMessage", "text": "world"},
					},
				}},
			},
		},
	}
	withManagedAppServerThreadClient(t, client)

	a := &Agent{backend: "app_server", appServerURL: "stdio://"}
	got, err := a.GetSessionHistory(context.Background(), "thread-app", 10)
	if err != nil {
		t.Fatalf("GetSessionHistory() returned error: %v", err)
	}
	if want := []string{"user:hello", "assistant:world"}; !reflect.DeepEqual(historyPairs(got), want) {
		t.Fatalf("history = %#v, want %v", got, want)
	}
	if len(client.calls) != 1 || client.calls[0].method != "thread/read" {
		t.Fatalf("calls = %#v, want thread/read", client.calls)
	}
	if client.calls[0].params["threadId"] != "thread-app" || client.calls[0].params["includeTurns"] != true {
		t.Fatalf("thread/read params = %#v", client.calls[0].params)
	}
}

func TestCreateSession_AppServerCreatesNamedThread(t *testing.T) {
	client := &fakeAppServerThreadClient{}
	client.startResponse.Thread.ID = "thread-new"
	captured := withManagedAppServerThreadClient(t, client)

	a := &Agent{backend: "app_server", appServerURL: "stdio://", workDir: t.TempDir(), model: "gpt-5", reasoningEffort: "high", mode: "yolo"}
	got, err := a.CreateSession(context.Background(), "Mobile test title")
	if err != nil {
		t.Fatalf("CreateSession() returned error: %v", err)
	}
	if got.ID != "thread-new" || got.Summary != "Mobile test title" {
		t.Fatalf("CreateSession() = %#v", got)
	}
	if len(client.calls) != 2 {
		t.Fatalf("calls = %#v, want thread/start and thread/name/set", client.calls)
	}
	if client.calls[0].method != "thread/start" || client.calls[1].method != "thread/name/set" {
		t.Fatalf("calls = %#v", client.calls)
	}
	if client.calls[1].params["threadId"] != "thread-new" || client.calls[1].params["name"] != "Mobile test title" {
		t.Fatalf("thread/name/set params = %#v", client.calls[1].params)
	}
	if captured.model != "gpt-5" || captured.effort != "high" || captured.workDir != a.workDir {
		t.Fatalf("managed app-server config = %#v", captured)
	}
	if client.calls[0].params["model"] != "gpt-5" || client.calls[0].params["effort"] != "high" {
		t.Fatalf("thread/start params = %#v", client.calls[0].params)
	}
}

func TestDeleteSession_AppServerArchivesThread(t *testing.T) {
	client := &fakeAppServerThreadClient{}
	withManagedAppServerThreadClient(t, client)

	a := &Agent{backend: "app_server", appServerURL: "stdio://"}
	if err := a.DeleteSession(context.Background(), "thread-app"); err != nil {
		t.Fatalf("DeleteSession() returned error: %v", err)
	}
	if len(client.calls) != 1 || client.calls[0].method != "thread/archive" {
		t.Fatalf("calls = %#v, want thread/archive", client.calls)
	}
	if client.calls[0].params["threadId"] != "thread-app" {
		t.Fatalf("thread/archive params = %#v", client.calls[0].params)
	}
}

func historyPairs(entries []core.HistoryEntry) []string {
	out := make([]string, 0, len(entries))
	for _, entry := range entries {
		out = append(out, entry.Role+":"+entry.Content)
	}
	return out
}
