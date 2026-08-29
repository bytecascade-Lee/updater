# Updater

Tauri 便携版自动更新器。解决 Tauri 官方 `tauri-plugin-updater` 仅支持安装包（`.msi`/`.exe`）、不支持便携版（portable zip）自动更新的问题。

## 为什么需要这个项目

Tauri 官方的更新插件原生只支持通过安装包（`.msi`、`.exe`）进行自动更新，不支持便携版（portable zip）压缩包的分发方式。如果你的 Tauri 应用以便携版形式发布，就无法使用官方的自动更新能力。

`updater` 作为一个独立的外部更新器，补充了这个缺口：它接收 JSON 配置，执行 **等待旧进程退出 → 替换文件 → 启动新进程** 的完整更新流程，并支持备份与回滚。

## 工作流程

```
wait  →  update  →  launch  →  （失败时 rollback）
```

1. **Wait** — 轮询旧进程 PID 直到退出（支持超时强制终止）
2. **Update** — 备份当前版本 → 清理目标目录（保留用户数据）→ 复制新版本
3. **Launch** — 启动新进程（支持分离/守护/驻留模式）
4. **Rollback** — 更新后启动失败时，自动还原备份并启动旧版本

## 配置格式

`updater` 接受一个 JSON 配置文件作为输入，完整字段如下：

```jsonc
{
  "version": 1,
  "runtime": {
    "headless": false,        // 无头模式（无 UI 输出）
    "logFile": "D:/updater.log"  // 日志文件路径
  },
  "wait": {
    "pid": 1234,                      // 旧进程 PID；-1 或 0 跳过等待
    "timeout": 10000,                 // 等待超时（ms）
    "forceKillAfterTimeout": true,    // 超时后强制终止
    "checkInterval": 300              // 轮询间隔（ms）
  },
  "update": {
    "source": "C:/new_app",           // 新版本文件路径
    "target": "C:/app",               // 当前安装路径
    "preserve": ["C:/app/data"],      // 更新时保留的路径（文件/目录）
    "cleanBeforeCopy": true,          // 复制前清理目标目录
    "backup": {
      "enabled": true,                // 是否备份
      "location": "D:/test/temp-bakup",  // 备份位置
      "exclude": ["D:/test/temp/data"]   // 排除路径（文件/目录）
    }
  },
  "launch": {
    "execution": {
      "mode": "direct",               // "direct" 或 "interpreted"
      "path": "app.exe",              // 可执行文件路径
      "interpreter": ["pwsh.exe"]     // 解释器（interpreted 模式使用）
    },
    "context": {
      "workspace": "C:/app",          // 工作目录（默认为 exe 所在目录）
      "args": ["--port", "8080"],     // 启动参数
      "env": { "RUST_LOG": "INFO" }   // 环境变量
    },
    "lifecycle": {
      "stayAlive": 0,                 // 0=分离, >0=守护ms, <0=无限驻留
      "captureOutput": false          // 是否捕获 stdout/stderr
    }
  },
  "rollback": {
    "enabled": true,
    "fallbackExecutable": "C:/app/old_app.exe",
    "maxAttempts": 2
  }
}
```

## 使用方式

```bash
updater.exe <config.json>
```

### 命令行选项

| 选项                    | 说明              |
|-----------------------|-----------------|
| `<config.json>`       | JSON 配置文件路径     |
| `-v`                  | 输出版本号           |
| `-b` / `--build-info` | 输出构建信息（JSON 格式） |
| `-h` / `--help`       | 显示帮助信息          |

### 退出码

| 退出码 | 含义                    |
|-----|-----------------------|
| 0   | 更新成功                  |
| 2   | wait 阶段失败             |
| 3   | update 阶段失败           |
| 4   | launch 阶段失败（含回滚成功的情况） |
| 5   | rollback 阶段失败         |

## 构建

### 本地构建

需要 Go 1.27+（参考 `go.mod`）。

```bash
# 默认架构
go build -o updater.exe -ldflags="-X main.Version=v1.0.0" ./cmd/updater

# 指定架构
GOOS=windows GOARCH=arm64 go build -o updater.exe -ldflags="-X main.Version=v1.0.0" ./cmd/updater
```

### 使用构建脚本

```bash
# 本地构建（所有架构）
uv run python scripts/release_local.py v1.0.0 --target all

# CI 构建（GitHub Actions）
uv run python scripts/release_ci.py build --target x86_64
```

## 自动更新清单

发布时自动生成 `latest.json`，供客户端检查更新：

```json
{
    "version": "1.0.0",
    "schemaVersion": "1",
    "publishTimestamp": 1724937600000,
    "windows": {
        "x86_64": {
            "url": "https://github.com/bytecascade-Lee/releases/download/v1.0.0/updater-1.0.0-windows-x86_64.exe",
            "sha256": "...",
            "size": 12345678
        },
        "arm64": {
            "url": "https://github.com/bytecascade-Lee/releases/download/v1.0.0/updater-1.0.0-windows-arm64.exe",
            "sha256": "...",
            "size": 12345678
        }
    }
}
```

## 项目结构

```
updater/
├── cmd/updater/          # CLI 入口
├── internal/
│   ├── config/           # JSON 配置解析与校验
│   ├── logger/           # 日志（支持文件 + 控制台）
│   ├── runner/           # 核心流程编排
│   │   ├── orchestrator.go   # wait → update → launch → rollback
│   │   ├── wait.go           # 等待旧进程退出
│   │   ├── update.go         # 备份、清理、复制
│   │   ├── launch.go         # 启动新进程
│   │   └── rollback.go       # 回滚还原
│   └── ui/               # 进度条与控制台输出
├── pkg/winutil/          # Windows 系统工具（进程操作等）
├── scripts/              # 构建与发布脚本（Python）
│   ├── release_local.py  # 本地构建
│   ├── release_ci.py     # CI 构建
│   └── publish.py        # 发布（GitHub + CNB）
├── .github/workflows/    # GitHub Actions
│   ├── release.yaml      # 发布流水线
│   └── sync-mirrors.yaml # CNB 镜像同步
└── go.mod
```

## 平台支持

| 平台      | 架构     | 状态   |
|---------|--------|------|
| Windows | x86_64 | ✅ 支持 |
| Windows | arm64  | ✅ 支持 |

> `updater` 仅在 Windows 上运行，因为其核心逻辑依赖 Windows 系统调用（进程操作、文件锁释放等）。

## 许可证

[MIT](LICENSE)
