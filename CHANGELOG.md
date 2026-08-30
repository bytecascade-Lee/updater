# CHANGELOG

All notable changes to this project will be documented in this file.

---

## 0.1.2

### Changed

- 重构构建信息字段命名与 JSON 输出：`GitCommit` → `ShortHash`，时间戳格式改为 ISO 8601，`buildInfoJSON` 新增 `branch` 字段，新增 `build-info-example.json` 示例文件
- 自动更新清单 `latest.json` 格式调整：`schemaVersion` 从字符串改为数字，`publishTimestamp`（毫秒时间戳）改为 `publishDate`（ISO 8601 含时区）

---

## 0.1.1

### Changed

- 重构 `ui/console.go` 控制台策略并新增 `SetupCLI`：CLI 命令（`-v`/`-h`/`--build-info`/无参数/配置错误）在 stdout 为有效管道/重定向时不触碰窗口（Rust 捕获场景），否则优先 `AttachConsole(-1)` 复用父终端，双击等无父控制台场景 `AllocConsole` 弹窗并在 `Teardown` 时等待按键退出
- 调整 `ui.Setup` 升级流程：`headless=false` 时优先附加父终端复用，失败才 `AllocConsole` 弹独立窗口；`headless=true` 保持完全静默且不触碰标准句柄
- `main.go` 各 CLI 分支接入 `SetupCLI`/`Teardown`，保证 `-H windowsgui` 下 `-v`/`-h`/`--build-info`/无参数/配置错误输出可达
- 日志模块 `logger` 在打开日志文件前自动创建父目录，避免因目录缺失导致 `OpenFile` 失败
- 配置加载模块 `config` 使用 `new(T)` 简化默认值赋值，移除 `update.preserve` 非空校验，允许空数组

---

## 0.1.0

### Added

- 新增更新流程编排模块 `runner`，按序执行 wait → update → launch 流程，launch 失败时若启用备份与回滚则自动执行 rollback
- 新增 JSON 配置文件加载模块 `config`，支持 `Load` 读取、`Validate` 校验、`applyDefaults` 默认值填充，以及 `FlexibleInt` 兼容整数与数字字符串
- 新增等待阶段模块 `wait`，轮询指定 PID 直至进程退出，支持超时后按 `forceKill` 决定是否强制终止，强制终止后等待 5 秒确保文件锁释放
- 新增文件更新阶段模块 `update`，执行备份 → 清理 → 拷贝四步流程；`syncSourceToTarget` 支持 preserve 路径跳过写入，`copyDir` 支持 excludeSet 路径排除
- 新增启动阶段模块 `launch`，支持 direct/interpreted 两种启动模式，stayAlive 的分离/守护/驻留三种行为，以及 captureOutput 将子进程输出重定向至日志
- 新增回滚阶段模块 `rollback`，还原备份至 target 目录并以分离模式启动 fallback 进程，支持 `maxAttempts` 多次重试
- 新增 Windows 路径工具函数 `pkg/winutil`：`NormPath` 归一化路径、`IsPathInside` 判断路径包含关系、`PathInSet` 命中归一化集合
- 新增 Windows 进程管理函数：`ProcessExists` 检查进程是否存在、`KillProcess` 强制终止、`NewDetachedProcAttr` 返回分离启动属性
- 新增基于 logrus 的日志模块 `logger`，支持 headless 模式控制控制台与文件双写，`lineHook` 在每条日志前清除进度条所在行
- 新增进度条模块 `ui`，通过 `\r` 原地覆盖实现固定在输出最下方的状态行，支持 headless 场景下所有操作为空操作
- 新增 Windows 控制台生命周期管理模块 `ui/console.go`，按 headless 模式按需分配新控制台并通过 `sync.Once` 确保仅分配一次
- 新增 CLI 入口 `cmd/updater`，支持 `-v` 输出版本号、`-b`/`--build-info` 输出 JSON 构建信息、`-h`/`--help` 显示帮助
- 新增构建信息注入机制，通过 ldflags 注入 Version、BuildTime、GitCommit、GitCommitCount、GitCommitTime、GitBranch
- 新增 Python 构建工具链 `scripts/common`：`git.py` Git 操作封装、`version.py` 语义化版本处理、`builder.py` Go 构建辅助、`logger.py` 结构化日志
- 新增本地构建脚本 `release_local.py`，支持 `--target` 指定架构（x64/arm64/all），产物输出至 `release/Local/`
- 新增 CI 构建脚本 `release_ci.py`，版本号从 GitHub Actions 环境变量自动提取，默认校验 min_level="rc"
- 新增统一发布脚本 `publish.py`，发布 GitHub Release 并同步 CNB Release，自动生成 latest-github.json 与 latest-cnb.json 自动更新清单
- 新增变更日志生成脚本 `generate_commit_message.py`，根据 Git 提交范围生成 Markdown 格式变更日志
- 新增 Release 工作流 `.github/workflows/release.yaml`，支持 tag push 与 workflow_dispatch 触发，validate → build → release 三阶段
- 新增代码镜像同步工作流 `.github/workflows/sync-mirrors.yaml`，GitHub 推送自动同步至 CNB 镜像
- 新增配置示例文件 `example.json` 与自动更新清单格式示例 `latest-example.json`
- 新增 README.md（中文）与 README-en-US.md（英文）项目文档
- 新增 `docs/develop/CHANGELOG_STANDARD.md` CHANGELOG 书写规范
- 新增 `docs/develop/RELEASE_NOTES_STANDARD.md` RELEASE_NOTES 书写规范
- 添加 MIT 许可证

### Fixed

- 修复 `ProcessExists` 在进程被终止后仍误判存活的问题：`OpenProcess` 成功后增加 `GetExitCodeProcess` 检查退出码是否为 STILL_ACTIVE (259)

### Changed

- 重构 Release 工作流，build job 从矩阵策略改为单 job 内串行构建 x86_64 和 arm64，各架构产物分别上传为独立 artifact
- 添加 `.gitattributes` 配置文本文件 LF 规范化、Go 源文件 diff=golang、Shell 脚本强制 LF 行尾、二进制文件标记
- 添加 `.gitignore` 忽略 IDE 配置、Python 缓存、构建产物目录、文档草稿、日志文件、环境变量文件及 Go 可执行文件
