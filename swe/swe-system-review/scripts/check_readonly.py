#!/usr/bin/env python3
"""Print or compare a Git worktree content fingerprint; no files are created."""
import argparse
import hashlib
import json
import importlib.util
import subprocess
import sys
from pathlib import Path


def fingerprint(root):
    source = Path(__file__).resolve().parents[2] / 'scripts/check_scope.py'
    if not source.is_file():
        raise ValueError(f'缺少 SWE 共享检查脚本：{source}；部署时须保留完整 SWE 包')
    spec = importlib.util.spec_from_file_location('shared_scope', source)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    files = module.inventory(root)
    entries = [[name, *entry] for name, entry in sorted(files.items())]
    payload = json.dumps([str(root), entries], ensure_ascii=True, separators=(',', ':')).encode()
    return hashlib.sha256(payload).hexdigest(), len(entries)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--root', required=True, type=Path)
    parser.add_argument('--expected', help='审查前保存在执行上下文中的 SHA256，不写基线文件')
    args = parser.parse_args()
    try:
        if args.expected and (len(args.expected) != 64 or any(c not in '0123456789abcdef' for c in args.expected)):
            raise ValueError('expected 须为审查前输出的完整 SHA256')
        digest, count = fingerprint(args.root.resolve(strict=True))
        changed = args.expected is not None and digest != args.expected
        print(json.dumps({'sha256': digest, 'files': count,
                          'comparison': 'changed' if changed else ('unchanged' if args.expected else 'baseline'),
                          'coverage': 'Git 跟踪及非忽略文件的最终内容、类型与权限；包含子模块引用、检出版本及已初始化子模块的跟踪和非忽略文件，未初始化子模块中的现有普通文件也纳入；不含 .git、忽略文件、项目外或改后恢复的动作'}, ensure_ascii=False))
        return 1 if changed else 0
    except (OSError, ValueError, subprocess.CalledProcessError) as exc:
        print(f'ERROR: 无法核对：{exc}', file=sys.stderr)
        return 2


if __name__ == '__main__':
    sys.exit(main())
