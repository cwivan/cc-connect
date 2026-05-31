package codex

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/chenhg5/cc-connect/core"
)

type appServerThreadListResponse struct {
	Data       []appServerThread `json:"data"`
	NextCursor *string           `json:"nextCursor"`
}

type appServerThreadReadResponse struct {
	Thread appServerThread `json:"thread"`
}

type appServerThread struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Preview   string          `json:"preview"`
	Cwd       string          `json:"cwd"`
	CreatedAt int64           `json:"createdAt"`
	UpdatedAt int64           `json:"updatedAt"`
	Turns     []appServerTurn `json:"turns"`
}

type appServerTurn struct {
	StartedAt   *int64           `json:"startedAt"`
	CompletedAt *int64           `json:"completedAt"`
	Items       []map[string]any `json:"items"`
}

func listDesktopAppThreads(ctx context.Context, url, workDir, codexHome string) ([]core.AgentSessionInfo, error) {
	client, err := newDesktopAppClient(ctx, url, codexHome)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	absWorkDir, err := filepath.Abs(workDir)
	if err != nil {
		absWorkDir = workDir
	}

	var all []appServerThread
	var cursor *string
	for {
		params := map[string]any{
			"archived":      false,
			"cwd":           absWorkDir,
			"limit":         100,
			"sortDirection": "desc",
			"sortKey":       "updated_at",
		}
		if cursor != nil && *cursor != "" {
			params["cursor"] = *cursor
		}

		var resp appServerThreadListResponse
		if err := client.request("thread/list", params, &resp); err != nil {
			return nil, fmt.Errorf("codex desktop_app thread/list: %w", err)
		}
		all = append(all, resp.Data...)
		if resp.NextCursor == nil || *resp.NextCursor == "" {
			break
		}
		cursor = resp.NextCursor
	}

	return mapAppThreadsToSessions(all, readPinnedThreadOrder(codexHome)), nil
}

func getDesktopAppThreadHistory(ctx context.Context, url, codexHome, sessionID string, limit int) ([]core.HistoryEntry, error) {
	client, err := newDesktopAppClient(ctx, url, codexHome)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	var resp appServerThreadReadResponse
	if err := client.request("thread/read", map[string]any{
		"threadId":     sessionID,
		"includeTurns": true,
	}, &resp); err != nil {
		return nil, fmt.Errorf("codex desktop_app thread/read: %w", err)
	}

	entries := appThreadHistory(resp.Thread)
	if limit > 0 && len(entries) > limit {
		entries = entries[len(entries)-limit:]
	}
	return entries, nil
}

func createDesktopAppThread(ctx context.Context, url, workDir, model, effort, mode, baseURL, modelProvider, codexHome, name string) (core.AgentSessionInfo, error) {
	client, err := newDesktopAppClient(ctx, url, codexHome)
	if err != nil {
		return core.AgentSessionInfo{}, err
	}
	defer client.Close()

	client.workDir = workDir
	client.model = model
	client.effort = effort
	client.mode = mode
	client.baseURL = baseURL
	client.modelProvider = modelProvider

	var resp threadStartResponse
	if err := client.request("thread/start", client.threadRequestParams(), &resp); err != nil {
		return core.AgentSessionInfo{}, fmt.Errorf("codex desktop_app thread/start: %w", err)
	}
	threadID := strings.TrimSpace(resp.Thread.ID)
	if threadID == "" {
		return core.AgentSessionInfo{}, fmt.Errorf("codex desktop_app thread/start returned empty thread id")
	}
	if title := strings.TrimSpace(name); title != "" {
		if err := client.request("thread/name/set", map[string]any{
			"threadId": threadID,
			"name":     title,
		}, nil); err != nil {
			return core.AgentSessionInfo{}, fmt.Errorf("codex desktop_app thread/name/set: %w", err)
		}
	}
	return core.AgentSessionInfo{
		ID:         threadID,
		Summary:    strings.TrimSpace(name),
		ModifiedAt: time.Now(),
	}, nil
}

func archiveDesktopAppThread(ctx context.Context, url, codexHome, sessionID string) error {
	client, err := newDesktopAppClient(ctx, url, codexHome)
	if err != nil {
		return err
	}
	defer client.Close()
	if err := client.request("thread/archive", map[string]any{"threadId": sessionID}, nil); err != nil {
		return fmt.Errorf("codex desktop_app thread/archive: %w", err)
	}
	return nil
}

func mapAppThreadsToSessions(threads []appServerThread, pinned map[string]int) []core.AgentSessionInfo {
	out := make([]core.AgentSessionInfo, 0, len(threads))
	for _, th := range threads {
		id := strings.TrimSpace(th.ID)
		if id == "" {
			continue
		}
		summary := strings.TrimSpace(th.Name)
		if summary == "" {
			summary = strings.TrimSpace(th.Preview)
		}
		if summary == "" {
			summary = "(empty)"
		}
		if len([]rune(summary)) > 60 {
			summary = string([]rune(summary)[:60]) + "..."
		}
		out = append(out, core.AgentSessionInfo{
			ID:           id,
			Summary:      summary,
			MessageCount: len(th.Turns),
			ModifiedAt:   unixTime(th.UpdatedAt),
		})
	}

	sort.SliceStable(out, func(i, j int) bool {
		pi, iPinned := pinned[out[i].ID]
		pj, jPinned := pinned[out[j].ID]
		if iPinned != jPinned {
			return iPinned
		}
		if iPinned && jPinned && pi != pj {
			return pi < pj
		}
		return out[i].ModifiedAt.After(out[j].ModifiedAt)
	})
	return out
}

func readPinnedThreadOrder(codexHome string) map[string]int {
	path := filepath.Join(resolveCodexHomeDir(codexHome), ".codex-global-state.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil
	}
	order := map[string]int{}
	collectPinnedThreadIDs(raw, order)
	return order
}

func collectPinnedThreadIDs(v any, out map[string]int) {
	switch x := v.(type) {
	case map[string]any:
		for k, val := range x {
			if k == "pinned-thread-ids" {
				if arr, ok := val.([]any); ok {
					for _, entry := range arr {
						if id, ok := entry.(string); ok {
							if _, exists := out[id]; !exists {
								out[id] = len(out)
							}
						}
					}
				}
				continue
			}
			collectPinnedThreadIDs(val, out)
		}
	case []any:
		for _, val := range x {
			collectPinnedThreadIDs(val, out)
		}
	}
}

func appThreadHistory(thread appServerThread) []core.HistoryEntry {
	var entries []core.HistoryEntry
	for _, turn := range thread.Turns {
		ts := unixTimePtr(turn.StartedAt)
		for _, item := range turn.Items {
			switch itemType, _ := item["type"].(string); itemType {
			case "userMessage":
				for _, text := range userMessageTexts(item["content"]) {
					entries = append(entries, core.HistoryEntry{Role: "user", Content: text, Timestamp: ts})
				}
			case "agentMessage":
				if text, _ := item["text"].(string); strings.TrimSpace(text) != "" {
					entries = append(entries, core.HistoryEntry{Role: "assistant", Content: text, Timestamp: ts})
				}
			}
		}
	}
	return entries
}

func userMessageTexts(raw any) []string {
	items, ok := raw.([]any)
	if !ok {
		return nil
	}
	var out []string
	for _, entry := range items {
		m, ok := entry.(map[string]any)
		if !ok {
			continue
		}
		if typ, _ := m["type"].(string); typ != "text" {
			continue
		}
		if text, _ := m["text"].(string); strings.TrimSpace(text) != "" {
			out = append(out, text)
		}
	}
	return out
}

func unixTime(seconds int64) time.Time {
	if seconds <= 0 {
		return time.Time{}
	}
	return time.Unix(seconds, 0)
}

func unixTimePtr(seconds *int64) time.Time {
	if seconds == nil {
		return time.Time{}
	}
	return unixTime(*seconds)
}
