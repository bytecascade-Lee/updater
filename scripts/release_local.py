#!/usr/bin/env python3
"""
本地构建打包脚本：构建 Go 更新器。

用法:
    uv run python scripts/release_local.py <版本号> [--target <target>] [--output-dir <dir>]

版本号可带 v 也可不带，例如: v0.1.0-beta.2 或 0.1.0-rc.1
与 CI 发布流程一致，但本地构建对 alpha/beta 不做限制，且产物名携带构建信息。

target 支持简化别名与 all：
    --target all                   # 打包全部支持架构（x86_64 + arm64）
    --target x64 / x86_64 / x86-64 # → x86_64
    --target arm64 / aarch64       # → arm64
    （缺省为本机默认架构）

产物输出到 <output-dir>/<版本号>+<分支名>.<提交数>.<短哈希>/
    - updater-<版本号>+<构建信息>-windows-<架构>.exe
"""

import argparse
import platform
import sys
from pathlib import Path

from common import builder, git, version
from common.logger import log

ROOT = Path(__file__).resolve().parent.parent
DEFAULT_OUTPUT = ROOT / "release" / "Local"

# target 别名 → 产物架构名
TARGET_ALIASES = {
    "x64": "x86_64",
    "x86_64": "x86_64",
    "x86-64": "x86_64",
    "amd64": "x86_64",
    "arm64": "arm64",
    "aarch64": "arm64",
}


def fail(message: str) -> None:
    log("ERROR", message)
    raise SystemExit(1)


def host_arch() -> str:
    """返回本机默认产物架构名。"""
    machine = platform.machine().lower()
    if "aarch64" in machine or "arm64" in machine:
        return "arm64"
    return "x86_64"


def targets_for(target: str) -> list:
    """解析 --target 参数为产物架构名列表。"""
    if target == "all":
        return ["x86_64", "arm64"]
    if not target:
        return [host_arch()]
    if target not in TARGET_ALIASES:
        fail(f"不支持的 target: {target!r}，可选: all / x64 / x86_64 / x86-64 / arm64 / aarch64")
    return [TARGET_ALIASES[target]]


def build_targets(target_list, full_version: str, out_dir: Path) -> None:
    """对每个 target 依次构建。"""
    for t in target_list:
        builder.go_build(ROOT, t, full_version, f"v{full_version}", out_dir)


def main() -> None:
    if not sys.platform.startswith("win"):
        fail(f"不支持当前操作系统: {sys.platform}，本地构建仅支持 Windows")

    parser = argparse.ArgumentParser(
        description="本地构建 Go 更新器并输出 updater-<版本>+<构建信息>-windows-<架构>.exe"
    )
    parser.add_argument(
        "version",
        help="版本号，可带 v 也可不带，例如 v0.1.0-beta.2 或 0.1.0-rc.1",
    )
    parser.add_argument(
        "--target",
        default=None,
        help="架构（别名/all），如 x64、arm64、all；缺省为本机默认",
    )
    parser.add_argument(
        "--output-dir",
        default=str(DEFAULT_OUTPUT),
        help="产物根目录（内部按 full_version 分子目录），默认 release/Local",
    )
    args = parser.parse_args()

    try:
        release_version = version.validate(args.version, min_level=None)
    except version.VersionError as e:
        fail(str(e))
    if "+" in release_version:
        fail(
            f"本地构建会在版本号后追加构建信息，不允许版本号自带 + 构建元数据: {args.version!r}"
        )

    target_list = targets_for(args.target)

    build_info = git.get_build_info(cwd=ROOT)
    full_version = f"{release_version}+{build_info}"
    out_dir = Path(args.output_dir) / full_version
    log("INFO", f"版本号: {release_version} | 构建信息: {build_info}")
    log("INFO", f"目标架构: {', '.join(target_list)}")
    log("INFO", f"产物目录: {out_dir}")

    build_targets(target_list, full_version, out_dir)
    log("INFO", f"本地构建完成，产物位于 {out_dir}")


if __name__ == "__main__":
    main()
