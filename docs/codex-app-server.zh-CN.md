# Codex app_server 交付配置

本文档用于把 cc-connect 通过飞书接入 Codex，并让会话通过 Codex `app-server` 同步到同一账号下的 Codex 移动 App。团队默认使用 `backend = "app_server"` 和 `app_server_url = "stdio"`，不依赖 Codex Desktop App 开放 WebSocket。

## 适用范围

- 用户通过飞书和 Codex 对话。
- cc-connect 在本机启动 `codex app-server` stdio RPC。
- `/list`、`/switch`、`/new 名称`、普通消息都走 Codex thread/turn。
- Codex 移动 App 可通过账号同步看到对应 thread。
- Codex Desktop App 实时 UI 刷新不作为 v1 验收条件。

`desktop_app` 仅作为实验后端保留，只有在明确存在可连接的 Desktop App WS/IPC 时才使用。

## 前置条件

1. 安装 Go 1.22+。
2. 安装并登录 Codex CLI：

```bash
npm install -g @openai/codex
codex --version
codex login
```

3. 确认 Codex app-server 支持 stdio：

```bash
codex app-server --help
```

输出中应包含 `--listen <URL>`，并支持 `stdio://`。

4. 已有飞书机器人 `app_id` 和 `app_secret`，并已开通消息事件。

## 下载和编译

当前交付分支：

```bash
git clone -b codex/guide-alias-feishu-card https://github.com/cwivan/cc-connect.git
cd cc-connect
go test ./agent/codex ./core ./platform/feishu -count=1
go build -tags no_web -o cc-connect-dev ./cmd/cc-connect
```

如果仓库已经存在：

```bash
cd cc-connect
git fetch
git checkout codex/guide-alias-feishu-card
git pull
go build -tags no_web -o cc-connect-dev ./cmd/cc-connect
```

## 配置

建议使用全局配置文件：

```bash
mkdir -p ~/.cc-connect
cp config.example.toml ~/.cc-connect/config.toml
vim ~/.cc-connect/config.toml
```

最小配置示例：

```toml
[[projects]]
name = "my-codex-project"

[projects.agent]
type = "codex"

[projects.agent.options]
work_dir = "/absolute/path/to/your/project"
mode = "suggest" # suggest | auto-edit | full-auto | yolo
backend = "app_server"
app_server_url = "stdio"
# model = "gpt-5"              # 可选
# reasoning_effort = "high"    # 可选：low | medium | high | xhigh

[[projects.platforms]]
type = "feishu"

[projects.platforms.options]
app_id = "your-feishu-app-id"
app_secret = "your-feishu-app-secret"
enable_feishu_card = true
allow_from = "*"
```

关键点：

- `backend = "app_server"` 是团队默认方案。
- `app_server_url = "stdio"` 表示 cc-connect 自己启动 `codex app-server` 并通过 stdio 通信。
- `app`、`codex-app` 会映射到 `app_server`。
- `desktop-app`、`desktop_app` 才会映射到实验性的 `desktop_app`。
- 不要为了移动 App 可见配置 `desktop_app_url`。

## 启动

前台启动：

```bash
./cc-connect-dev --force --config ~/.cc-connect/config.toml
```

后台启动示例：

```bash
nohup ./cc-connect-dev --force --config ~/.cc-connect/config.toml \
  > ~/.cc-connect/cc-connect.out.log \
  2> ~/.cc-connect/cc-connect.err.log &
```

如果需要重启：

```bash
pkill -f cc-connect-dev || true
./cc-connect-dev --force --config ~/.cc-connect/config.toml
```

## 验证

在飞书中按顺序测试：

1. 发送 `/list`：应返回 Codex 会话列表，优先显示 Codex App thread 标题，其次显示 preview。
2. 发送 `/new 测试标题`：应创建新的 Codex thread，并在 Codex 移动 App 中看到同名会话。
3. 发送普通消息：消息应进入当前绑定的 Codex thread，飞书收到进度和最终回复。
4. 发送 `/switch 1` 或点击列表按钮：应切换到对应 Codex thread。

验收口径：

- Codex 移动 App/同账号云端可见作为 v1 验收标准。
- Desktop App 是否实时刷新取决于官方桌面端是否暴露可连接通道，不作为默认交付要求。

## 常见问题

### `/list` 报 `desktop_app connect ... refused`

配置仍在使用 `desktop_app`。改为：

```toml
backend = "app_server"
app_server_url = "stdio"
```

### `/new` 后移动 App 看不到

检查三项：

1. 本机 Codex CLI 是否已登录同一个 Codex 账号。
2. cc-connect 是否使用 `backend = "app_server"`。
3. 普通消息是否已经进入当前 thread；可通过 `/current` 或 `/list` 确认。

### 修改代码后不生效

重新编译并重启同一个二进制：

```bash
go build -tags no_web -o cc-connect-dev ./cmd/cc-connect
pkill -f cc-connect-dev || true
./cc-connect-dev --force --config ~/.cc-connect/config.toml
```

避免同时运行多个旧二进制，否则飞书事件可能被旧进程消费。
