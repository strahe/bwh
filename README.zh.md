# BWH

> **[English](README.md) | 中文**

功能完备的 Go 库、CLI 工具和 MCP 服务器，用于管理搬瓦工 VPS 实例。通过命令行和编程方式提供 KiwiVM 的大部分 VPS 管理功能。

## 安装

```bash
# CLI 工具
go install github.com/strahe/bwh/cmd/bwh@latest

# Go 库
go get github.com/strahe/bwh
```

## 快速开始

```bash
# 配置您的 VPS
bwh node add production --api-key <API_KEY> --veid <VEID>

# 基本操作
bwh info                              # 查看服务器详情
bwh start/stop/restart                # 电源管理
bwh usage --period 7d                 # 检查使用统计
bwh snapshot create "备份名称"         # 创建快照
bwh connect                           # SSH 连接

# 探索更多命令: bwh --help
```

### 多服务器管理

```bash
# 添加多个 VPS 实例
bwh node add prod --api-key <KEY> --veid <VEID>
bwh node add dev --api-key <KEY> --veid <VEID>

# 针对特定实例操作或设置默认实例
bwh --instance prod info
bwh node set-default prod

# 查看所有选项: bwh node --help
```

## Go SDK

```go
import "github.com/strahe/bwh/pkg/client"

// 初始化客户端
c := client.NewClient("your-api-key", "your-veid")
ctx := context.Background()

// 获取服务器信息
info, err := c.GetServiceInfo(ctx)
if err != nil {
    log.Fatal(err)
}

// 电源管理
err = c.Start(ctx)                    // 启动 VPS
err = c.Stop(ctx)                     // 停止 VPS  
err = c.Restart(ctx)                  // 重启 VPS

// 监控
usage, err := c.GetRawUsageStats(ctx) // 使用统计
live, err := c.GetLiveServiceInfo(ctx) // 实时状态

// 备份管理
snapshot, err := c.CreateSnapshot(ctx, "backup-name")
backups, err := c.ListBackups(ctx)
```

### 可用方法

**服务器管理**: `GetServiceInfo`、`GetLiveServiceInfo`、`Start`、`Stop`、`Restart`、`Kill`、`SetHostname`、`ReinstallOS`、`ResetRootPassword`

**监控**: `GetRawUsageStats`、`GetBasicServiceInfo`、审计日志访问

**备份和恢复**: `CreateSnapshot`、`RestoreSnapshot`、`DeleteSnapshot`、备份管理

**网络**: SSH 密钥管理、IP/反向 DNS 配置

*完整 API 参考*: 查看 [pkg/client 文档](./pkg/client) 或运行 `go doc github.com/strahe/bwh/pkg/client` 获取所有可用方法。

## MCP 服务器

内置 MCP (Model Context Protocol) 服务器，实现与 AI 工具的安全集成，提供 VPS 管理工作流支持。

### 启动服务器

```bash
bwh mcp serve
```

### 配置方法

BWH MCP 服务器可与多种 AI 工具和编辑器无缝集成。向您的 MCP 客户端添加相应配置：

#### Claude Desktop

添加到 `claude_desktop_config.json` 文件：

```json
{
  "mcpServers": {
    "bwh": {
      "command": "bwh",
      "args": ["mcp", "serve"]
    }
  }
}
```

自定义配置文件: `"args": ["mcp", "serve", "--config", "/path/to/config.yaml"]`

#### Claude Code

```bash
claude mcp add bwh -- bwh mcp serve
```

自定义配置: `claude mcp add bwh -- bwh mcp serve --config /path/to/config.yaml`

#### Cursor

添加到 Cursor MCP 配置：

```json
{
  "mcpServers": {
    "bwh": {
      "command": "bwh",
      "args": ["mcp", "serve"]
    }
  }
}
```

自定义配置文件: `"args": ["mcp", "serve", "--config", "/path/to/config.yaml"]`

#### Continue (VS Code 扩展)

添加到 Continue 配置：

```json
{
  "mcpServers": {
    "bwh": {
      "transport": {
        "type": "stdio",
        "command": "bwh",
        "args": ["mcp", "serve"]
      }
    }
  }
}
```

自定义配置文件: `"args": ["mcp", "serve", "--config", "/path/to/config.yaml"]`

### 配置说明

- **自定义配置**: 使用 `--config /path/to/config.yaml` 参数而非环境变量
- **多实例**: 服务器自动使用配置中的默认实例
- **集成**: 添加到现有 MCP 配置文件中，不替换其他服务器

### 可用工具

- **vps_info_get**: 获取 VPS 信息 (`instance?`, `compact?`, `live?`)
- **vps_usage_get**: 获取使用统计 (`instance?`, `period?`, `days?`, `group_by?`)
- **snapshot_list**: 列出快照 (`instance?`, `sticky_only?`, `name_contains?`, `sort_by?`, `order?`, `limit?`)
- **backup_list**: 列出备份 (`instance?`, `os_contains?`, `since?`, `until?`, `sort_by?`, `order?`, `limit?`)
- **vps_audit_get**: 获取审计日志 (`instance?`, `since?`, `until?`, `limit?`, `ip_contains?`, `type?`)

所有 MCP 工具都是安全的只读操作，不会修改您的 VPS 配置或数据。

## 可用命令

```
node            管理 BWH VPS 节点配置
info            显示综合 VPS 信息
rate-limit      检查 API 限制状态
connect         SSH 连接到 VPS（无密码，使用本地 SSH 密钥）
ssh             管理 SSH 密钥
start/stop      启动/停止 VPS
restart         重启 VPS
kill            强制停止卡住的 VPS（警告：可能数据丢失）
hostname        设置 VPS 主机名
setPTR          设置 IP 地址的 PTR（rDNS）记录
reinstall       重装 VPS 操作系统（警告：摧毁所有数据）
usage           显示详细 VPS 使用统计
audit           显示审计日志条目
reset-password  重置 root 密码
snapshot        管理 VPS 快照
backup          管理 VPS 备份
mcp             运行 MCP 服务器以进行只读 BWH 管理
```

使用 `bwh <command> --help` 查看每个命令的详细选项和用法示例。

## 构建

```bash
make build  # 或者: go build -o bwh ./cmd/bwh
```