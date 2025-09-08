# bwh

[![CI](https://github.com/strahe/bwh/actions/workflows/ci.yml/badge.svg)](https://github.com/strahe/bwh/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/strahe/bwh)](https://goreportcard.com/report/github.com/strahe/bwh)
[![Release](https://img.shields.io/github/v/release/strahe/bwh)](https://github.com/strahe/bwh/releases)

> **[English](README.md) | 中文**

## 概述

用于管理搬瓦工（KiwiVM）VPS 的 Go SDK、CLI 和 MCP 服务器，支持命令行与编程方式访问大部分功能。

## 安装

前置条件：Go 1.24+；KiwiVM VEID 与 API Key。

```bash
# CLI 或者 MCP
go install github.com/strahe/bwh/cmd/bwh@latest

# Go SDK
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
bwh iso images                        # 列出可用 ISO 镜像
bwh iso mount ubuntu-20.04.iso        # 挂载 ISO 用于救援/安装
bwh ipv6 add                          # 分配新的 IPv6 /64 subnet
bwh ipv6 list                         # 列出 IPv6 子网
bwh pi info                           # 显示私有 IPv4 信息（等同 `private-ip info`）
bwh connect                           # SSH 连接

# 保持 BWH CLI 最新
bwh update                            # 检查并安装更新
bwh update --check                    # 仅检查更新

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
import (
    "context"
    "log"
    "github.com/strahe/bwh/pkg/client"
)

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

**服务器管理**: `GetServiceInfo`、`GetLiveServiceInfo`、`Start`、`Stop`、`Restart`、`Kill`、`SetHostname`、`ReinstallOS`、`ResetRootPassword`、`MountISO`、`UnmountISO`

**监控**: `GetRawUsageStats`、`GetBasicServiceInfo`、审计日志访问

**备份和恢复**: `CreateSnapshot`、`RestoreSnapshot`、`DeleteSnapshot`、备份管理

**迁移**: `GetMigrateLocations`、`StartMigration`（支持 `StartMigrationWithTimeout` 自定义超时）

**网络**: SSH 密钥管理、IP/反向 DNS 配置、IPv6 子网管理、私有 IPv4 管理

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



#### Claude Code

```bash
claude mcp add bwh -- bwh mcp serve
```



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



### 配置说明

- **自定义配置**: 使用 `--config /path/to/config.yaml` 指定配置文件
- **多实例**: 服务器自动使用配置中的默认实例
- **集成**: 添加到现有 MCP 配置文件中，不替换其他服务器

### 可用工具

- **instance_list**: 列出所有已配置的实例及元数据（无参数）
- **vps_info_get**: 获取 VPS 信息 (`instance?`, `compact?`, `live?`)
- **vps_usage_get**: 获取使用统计 (`instance?`, `period?`, `days?`, `group_by?`)
- **snapshot_list**: 列出快照 (`instance?`, `sticky_only?`, `name_contains?`, `sort_by?`, `order?`, `limit?`)
- **backup_list**: 列出备份 (`instance?`, `os_contains?`, `since?`, `until?`, `sort_by?`, `order?`, `limit?`)
- **vps_audit_get**: 获取审计日志 (`instance?`, `since?`, `until?`, `limit?`, `ip_contains?`, `type?`)
- **iso_list**: 列出可用和已挂载的 ISO 镜像 (`instance?`)

所有 MCP 工具都是安全的只读操作，不会修改您的 VPS 配置或数据。

## 自动补全

```bash
bwh completion bash > /usr/local/share/bash-completion/completions/bwh  # bash (Linux)
bwh completion bash > /opt/homebrew/etc/bash_completion.d/bwh           # bash (macOS)
bwh completion zsh > /usr/local/share/zsh/site-functions/_bwh           # zsh (系统级)
bwh completion fish > ~/.config/fish/completions/bwh.fish              # fish (用户级)
```

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
set-ptr         设置 IP 地址的 PTR（rDNS）记录
iso             管理 VPS 启动用 ISO 镜像
reinstall       重装 VPS 操作系统（警告：摧毁所有数据）
usage           显示详细 VPS 使用统计
audit           显示审计日志条目
reset-password  重置 root 密码
snapshot        管理 VPS 快照
backup          管理 VPS 备份
migrate         迁移 VPS 至其他位置（支持 --wait/--timeout）
ipv6            管理 IPv6 子网（添加、删除、列出）
private-ip (pi) 管理私有 IPv4 地址（info、available、assign、delete）
mcp             运行 MCP 服务器以进行只读 BWH 管理
update          检查更新并将 BWH CLI 更新到最新版本
completion      生成 shell 自动补全脚本
```

使用 `bwh <command> --help` 查看每个命令的详细选项和用法示例。

## 构建

```bash
make build  # 或者: go build -o bwh ./cmd/bwh
```

运行测试：`make test`

## 许可证

MIT，详见 `LICENSE`。
