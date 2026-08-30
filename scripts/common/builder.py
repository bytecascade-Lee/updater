#!/usr/bin/env python3
"""
Go 更新器构建辅助：go build + ldflags 版本注入（替代 rollcaller 的 tauri builder）。

产物命名：updater-<artifact_version>-windows-<target>.exe
    artifact_version 用于文件名（不带 v，如 1.0.0 或本地带构建信息 1.0.0+master.5.abc123）
    version_str 注入到 main.Version（带 v，如 v1.0.0）

注入的构建信息（main.GitCommit / GitCommitCount / GitCommitTime / GitBranch）
由本函数从 git 自动获取；构建时间取当前时间。
"""

import datetime
import os
import subprocess
from pathlib import Path
from typing import Optional

from . import git
from .logger import log

# 产物架构名 → GOARCH
GOARCH_MAP = {"x86_64": "amd64", "arm64": "arm64"}


def go_build(
    project_root: Path,
    target: str,
    artifact_version: str,
    version_str: str,
    out_dir: Optional[Path] = None,
) -> Path:
    """构建单个架构的更新器 exe（windowsgui 子系统，剥离调试符号）。

    Args:
        project_root: 项目根目录（go.mod 所在处）
        target: "x86_64" 或 "arm64"
        artifact_version: 产物文件名中的版本号（不带 v）
        version_str: 注入 main.Version 的版本字符串（带 v）
        out_dir: 产物输出目录，默认 project_root
    """
    goarch = GOARCH_MAP[target]
    out_dir = out_dir or project_root
    artifact = out_dir / f"updater-{artifact_version}-windows-{target}.exe"

    build_time = datetime.datetime.now().astimezone().isoformat()
    short_hash = git.get_head_hash(short=True, cwd=project_root)
    git_count = git.get_commit_count(cwd=project_root)
    git_time = git.get_head_commit_time(cwd=project_root)
    git_branch = git.get_branch(cwd=project_root)

    env = dict(os.environ)
    env["GOOS"] = "windows"
    env["GOARCH"] = goarch
    env["CGO_ENABLED"] = "0"
    ldflags = (
        f"-X main.Version={version_str} "
        f"-X main.BuildTime={build_time} "
        f"-X main.ShortHash={short_hash} "
        f"-X main.GitCommitCount={git_count} "
        f"-X main.GitCommitTime={git_time} "
        f"-X main.GitBranch={git_branch} "
        "-s -w -H windowsgui"
    )
    proc = subprocess.run(
        ["go", "build", "-ldflags", ldflags, "-o", str(artifact), "./cmd/updater"],
        cwd=project_root,
        env=env,
        capture_output=True,
        text=True,
        encoding="utf-8",
        errors="replace",
        check=False,
    )
    if proc.returncode != 0:
        raise RuntimeError(f"go build {target} 失败: {proc.stderr.strip() or proc.stdout.strip()}")

    size_mb = artifact.stat().st_size / 1024 / 1024
    log("INFO", f"已生成: {artifact} ({size_mb:.1f} MB)")
    return artifact
