#!/usr/bin/env python3
"""
CI 构建脚本：构建 Go 更新器并输出单一 exe 产物。

发布（GitHub Release + CNB Release + 自动更新清单）统一由 scripts/publish.py 负责，
本脚本只负责构建产物。

版本号来源（GitHub Actions 环境变量，由脚本统一提取与校验）：
    - push tag: 从 GITHUB_REF_NAME 读取（自动去掉前导 v）
    - workflow_dispatch: 从 inputs.version 读取（INPUT_VERSION）
等级校验（min_level="rc"）：放行 rc 及以上，禁止 alpha/beta。

产物命名：updater-<版本号>-windows-<架构>.exe（版本号不带 v）
    - updater-1.0.0-windows-x86_64.exe
    - updater-1.0.0-windows-arm64.exe

用法:
    uv run python scripts/release_ci.py build --target <target>
    target: x86_64 | arm64
"""

import argparse
import os
from pathlib import Path

from common import builder, version
from common.logger import log

ROOT = Path(__file__).resolve().parent.parent


def fail(message: str) -> None:
    log("ERROR", message)
    raise SystemExit(1)


def ci_version(min_level: str = "rc") -> str:
    """
    从 GitHub Actions 环境变量提取并校验发布版本号。

    min_level: CI 发布默认放行 rc 及以上（禁止 alpha/beta）。
    """
    if os.environ.get("GITHUB_EVENT_NAME") == "workflow_dispatch":
        raw = os.environ.get("INPUT_VERSION")
    else:
        raw = os.environ.get("GITHUB_REF_NAME")
    if not raw:
        raise version.VersionNotFoundError("Unable to obtain version, please check the environment variable settings")
    log("INFO", raw)
    try:
        return version.validate(raw, min_level=min_level)
    except version.VersionError as e:
        fail(str(e))


def cmd_build(target: str) -> None:
    if target not in builder.GOARCH_MAP:
        fail(f"不支持的 target: {target!r}，可选: {', '.join(builder.GOARCH_MAP)}")
    release_version = ci_version()
    log("INFO", f"版本号: {release_version} | arch: {target}")
    # CI 产物直接输出到工作区根目录，供 upload-artifact 收集
    builder.go_build(ROOT, target, release_version, f"v{release_version}")


def main() -> None:
    parser = argparse.ArgumentParser(
        description="CI 构建脚本：构建 Go 更新器（matrix 中每个架构各跑一次）"
    )
    sub = parser.add_subparsers(dest="command", required=True)
    build_p = sub.add_parser("build", help="构建并输出 updater-<版本>-windows-<架构>.exe")
    build_p.add_argument(
        "--target", required=True,
        help="架构：x86_64 或 arm64",
    )

    args = parser.parse_args()
    if args.command == "build":
        cmd_build(args.target)


if __name__ == "__main__":
    main()
