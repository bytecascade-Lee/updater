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

`updater` accepts a JSON configuration file as input. Field structure, required fields and defaults are governed by the JSON Schema at the repo root:

- **Example**: [`example.json`](./example.json) — a sample configuration with the full field set
- **JSON Schema**: [`config.schema.json`](./config.schema.json) — field definitions and validation rules. Associating it with your editor gives you field hints and validation while authoring (VS Code: point `json.schemas` in `settings.json` at the file, or inline `"$schema": "./config.schema.json"` at the top of your config JSON)

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

A `latest.json` is generated on release for clients to check for updates. Field structure and format follow [`latest-example.json`](./latest-example.json) at the repo root.

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
