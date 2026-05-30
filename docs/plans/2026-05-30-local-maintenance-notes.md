# cc-connect local maintenance notes

Date: 2026-05-30

This document captures a sanitized maintenance workflow for running and
updating a local cc-connect deployment. It intentionally avoids recording
personal paths, chat names, tokens, email addresses, or private repository
metadata.

## Working directory

Use the checked-out repository as the canonical source tree:

```text
<repo-root>
```

Do not continue feature work in temporary download or extraction directories.
Those copies can drift from the source tree used for commits and tests.

## Local toolchain

Required tools:

```text
Go
Git
GitHub CLI (optional, for PR and auth workflows)
```

If an already-open shell cannot find a newly installed tool, close and reopen it
so the updated user environment is loaded.

## Runtime binary

Some local deployments run from an installed package location instead of the
repository checkout. Treat that installed executable as the runtime location
only. Source edits, tests, builds, commits, and project maintenance should
happen in the repository checkout.

Example placeholders:

```text
<runtime-install>\bin\cc-connect.exe
<repo-root>\cc-connect.exe
<config-dir>\config.toml
<config-dir>\logs
```

After source changes, rebuild a Windows no-web binary when the web assets are
not available:

```powershell
$commit = (git rev-parse --short HEAD).Trim() + '-local'
$buildTime = (Get-Date).ToUniversalTime().ToString('yyyy-MM-ddTHH:mm:ssZ')
$ld = "-s -w -X main.commit=$commit -X main.buildTime=$buildTime"
go build -tags no_web -ldflags $ld -o .\cc-connect.exe .\cmd\cc-connect
```

Then replace the installed binary and restart the local process:

```powershell
$dst = '<runtime-install>\bin\cc-connect.exe'
$config = '<config-dir>\config.toml'
$out = '<config-dir>\logs\cc-connect.restart.out.log'
$err = '<config-dir>\logs\cc-connect.restart.err.log'

Get-Process cc-connect -ErrorAction SilentlyContinue | Stop-Process -Force
Start-Sleep -Seconds 2
Copy-Item -LiteralPath '.\cc-connect.exe' -Destination $dst -Force
Start-Process -FilePath $dst -ArgumentList @('--config', $config) -WindowStyle Hidden -RedirectStandardOutput $out -RedirectStandardError $err
```

Useful checks:

```powershell
& '<runtime-install>\bin\cc-connect.exe' --version
Get-Process cc-connect
Get-Content '<config-dir>\logs\cc-connect.restart.out.log' -Tail 40
```

## Configuration

Main config:

```text
<config-dir>\config.toml
```

Workspace binding file:

```text
<config-dir>\workspace_bindings.json
```

For Codex history filtering, bind each chat/workspace to the exact project path
stored in Codex session metadata. Binding a chat to a parent directory will not
show sessions whose recorded cwd is a nested project directory.

Recommended checks in the messaging client:

```text
/workspace
/list
```

For existing projects with historical sessions, prefer binding to the real
project path:

```text
/workspace route <absolute-project-path>
```

For new projects placed directly under the configured base directory, using
`/workspace bind <folder-name>` is sufficient.

## Git state

Keep repository and branch details in local notes or the remote host, not in
committed documentation. If a fork or feature branch is used, record only the
generic workflow here:

```text
git status
git log --oneline -5
git push <remote> <branch>
```

Security note: never record personal access tokens, bot secrets, app secrets, or
private credentials in this repository. Revoke any exposed token immediately.

## Implemented areas

### Feishu card title and tool rows

Purpose:

- Reduce oversized card titles/tool rows on Feishu mobile.
- Avoid large markdown heading patterns in tool progress messages.
- Keep tool status as smaller body text.

Relevant areas:

```text
platform/feishu
core
```

Suggested verification:

```powershell
go test ./platform/feishu -count=1
go test ./core -run TestI18n_ToolTemplateAvoidsLargeMarkdownTitle -count=1
```

### Session list and history navigation

Purpose:

- Keep `/list` session rows clickable in card UIs.
- After selecting a historical session, show that session's history preview.
- Display session list and history timestamps in the configured/local display
  timezone rather than raw UTC.

Relevant files:

```text
core/engine.go
core/engine_test.go
```

Suggested verification:

```powershell
go test ./core -run 'TestHandleCardNav_SwitchActionShowsSelectedSessionHistory|TestCmdSwitch_ByIndex_SetsSession|TestRenderListCard_MakesEveryVisibleSessionClickable|TestRenderListCard_DisplaysSessionTimeInLocalTimezone|TestRenderHistoryCard_DisplaysHistoryTimeInLocalTimezone' -count=1
go test ./platform/feishu -count=1
```

## Standard future workflow

When iterating on cc-connect from a messaging client:

1. Confirm the chat/workspace binding with `/workspace`.
2. Make source edits only in the canonical repository checkout.
3. Add focused regression tests for behavior changes.
4. Run the smallest relevant test set first, then broader package tests as risk
   increases.
5. Rebuild the runtime binary.
6. Replace the installed runtime binary.
7. Restart `cc-connect`.
8. Verify with logs and the mobile messaging workflow.
9. Commit and push to the intended remote branch.

Before claiming completion, run fresh verification commands and inspect their
output.
