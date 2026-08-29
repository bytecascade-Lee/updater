# Updater

A standalone auto-updater for Tauri portable (zip) applications. Bridges the gap where `tauri-plugin-updater` only supports installers (`.msi`/`.exe`) but not portable zip distributions.

## Why This Project Exists

Tauri's official updater plugin (`tauri-plugin-updater`) natively supports automatic updates only through installer packages (`.msi`,
`.exe`). If your Tauri app is distributed as a portable zip archive, you have no built-in auto-update mechanism.

`updater` fills this gap as an external, config-driven update tool. It executes a complete **wait for old process → replace files → launch new process** pipeline, with built-in backup and rollback.

## Workflow

```
wait  →  update  →  launch  →  (rollback on failure)
```

1. **Wait** — Polls the old process PID until it exits (supports timeout with force-kill)
2. **Update** — Backs up the current version → cleans the target directory (preserving user data) → copies the new version
3. **Launch** — Starts the new process (supports detached / stay-alive / resident modes)
4. **Rollback** — On post-update launch failure, automatically restores the backup and relaunches the old version

## Configuration

`updater` accepts a JSON configuration file. Full schema:

```jsonc
{
  "version": 1,
  "runtime": {
    "headless": false,        // Headless mode (no UI output)
    "logFile": "D:/updater.log"  // Log file path
  },
  "wait": {
    "pid": 1234,                      // Old process PID; -1 or 0 to skip waiting
    "timeout": 10000,                 // Wait timeout (ms)
    "forceKillAfterTimeout": true,    // Force-kill on timeout
    "checkInterval": 300              // Polling interval (ms)
  },
  "update": {
    "source": "C:/new_app",           // New version source path
    "target": "C:/app",               // Current install path
    "preserve": ["C:/app/data"],      // Paths to preserve during update (files/dirs)
    "cleanBeforeCopy": true,          // Clean target before copying
    "backup": {
      "enabled": true,                // Enable backup
      "location": "D:/test/temp-bakup",  // Backup location
      "exclude": ["D:/test/temp/data"]   // Paths to exclude during backup (files/dirs)
    }
  },
  "launch": {
    "execution": {
      "mode": "direct",               // "direct" or "interpreted"
      "path": "app.exe",              // Executable path
      "interpreter": ["pwsh.exe"]     // Interpreter (used in interpreted mode)
    },
    "context": {
      "workspace": "C:/app",          // Working directory (defaults to exe parent)
      "args": ["--port", "8080"],     // Launch arguments
      "env": { "RUST_LOG": "INFO" }   // Environment variables
    },
    "lifecycle": {
      "stayAlive": 0,                 // 0=detached, >0=guard ms, <0=infinite resident
      "captureOutput": false          // Capture stdout/stderr
    }
  },
  "rollback": {
    "enabled": true,
    "fallbackExecutable": "C:/app/old_app.exe",
    "maxAttempts": 2
  }
}
```

## Usage

```bash
updater.exe <config.json>
```

### CLI Options

| Option                | Description                     |
|-----------------------|---------------------------------|
| `<config.json>`       | Path to JSON configuration file |
| `-v`                  | Print version                   |
| `-b` / `--build-info` | Print build info (JSON)         |
| `-h` / `--help`       | Show help                       |

### Exit Codes

| Code | Meaning                                             |
|------|-----------------------------------------------------|
| 0    | Update succeeded                                    |
| 2    | Wait stage failed                                   |
| 3    | Update stage failed                                 |
| 4    | Launch stage failed (including successful rollback) |
| 5    | Rollback stage failed                               |

## Building

### Prerequisites

Go 1.27+ (see `go.mod`).

### Manual Build

```bash
# Default architecture
go build -o updater.exe -ldflags="-X main.Version=v1.0.0" ./cmd/updater

# Cross-compile for ARM64
GOOS=windows GOARCH=arm64 go build -o updater.exe -ldflags="-X main.Version=v1.0.0" ./cmd/updater
```

### Build Scripts

```bash
# Local build (all architectures)
uv run python scripts/release_local.py v1.0.0 --target all

# CI build (GitHub Actions)
uv run python scripts/release_ci.py build --target x86_64
```

## Update Manifest

A `latest.json` is generated on release for clients to check for updates:

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

## Project Structure

```
updater/
├── cmd/updater/          # CLI entrypoint
├── internal/
│   ├── config/           # JSON config parsing & validation
│   ├── logger/           # Logging (file + console)
│   ├── runner/           # Core pipeline orchestration
│   │   ├── orchestrator.go   # wait → update → launch → rollback
│   │   ├── wait.go           # Wait for old process exit
│   │   ├── update.go         # Backup, clean, copy
│   │   ├── launch.go         # Launch new process
│   │   └── rollback.go       # Restore backup
│   └── ui/               # Progress bar & console output
├── pkg/winutil/          # Windows utilities (process ops, etc.)
├── scripts/              # Build & release scripts (Python)
│   ├── release_local.py  # Local build
│   ├── release_ci.py     # CI build
│   └── publish.py        # Publish (GitHub + CNB)
├── .github/workflows/    # GitHub Actions
│   ├── release.yaml      # Release pipeline
│   └── sync-mirrors.yaml # CNB mirror sync
└── go.mod
```

## Platform Support

| Platform | Architecture | Status      |
|----------|--------------|-------------|
| Windows  | x86_64       | ✅ Supported |
| Windows  | arm64        | ✅ Supported |

> `updater` runs exclusively on Windows, as its core logic depends on Windows system calls (process management, file lock release, etc.).

## License

[MIT](LICENSE)
