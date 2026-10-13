#!/usr/bin/env python3
"""Snapshot and compare tracked/non-ignored Git files without changing the project."""
import argparse
import importlib.util
import json
import subprocess
import sys
from pathlib import Path


def scope_checker():
    source = Path(__file__).resolve().parents[2] / 'scripts/check_scope.py'
    if not source.is_file():
        raise ValueError(f'缺少 SWE 共享检查脚本：{source}；部署时须保留完整 SWE 包')
    spec = importlib.util.spec_from_file_location('shared_scope', source)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


def inventory(root):
    return scope_checker().inventory(root)


def allowed_path(root, value):
    return scope_checker().normalize(root, value)


def compare(before, after, allowed):
    changed = sorted(name for name in set(before) | set(after) if before.get(name) != after.get(name))
    return changed, [name for name in changed if name not in allowed]


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('mode', choices=['snapshot', 'check'])
    parser.add_argument('--root', required=True, type=Path)
    parser.add_argument('--baseline', required=True, type=Path)
    parser.add_argument('--allow', action='append', default=[])
    args = parser.parse_args()
    try:
        root = args.root.resolve(strict=True)
        checker = scope_checker()
        git_root = Path(checker.git(root, 'rev-parse', '--show-toplevel', text=True).stdout.strip()).resolve()
        if git_root != root:
            raise ValueError('root 须为目标项目 Git 根目录，避免漏掉其他目录的越界改动')
        baseline = args.baseline.resolve()
        if baseline == root or root in baseline.parents:
            raise ValueError('基线必须保存在目标项目之外的临时目录')
        if args.mode == 'snapshot':
            if len(args.allow) != 2 or len(set(args.allow)) != 2:
                raise ValueError('明确列出两份设计文档作为允许文件')
            allowed = [allowed_path(root, value) for value in args.allow]
            if len(set(allowed)) != 2:
                raise ValueError('两份允许文档须为不同路径')
            current = inventory(root)
            scope_checker().check_submodule_scope(current, allowed, [])
            for name in allowed:
                if name not in current:
                    ignored = subprocess.run(['git', '-C', str(root), 'check-ignore', '-q', '--', name], timeout=10)
                    if ignored.returncode == 0:
                        raise ValueError(f'允许文档被 Git 忽略，无法纳入检查：{name}')
                    if ignored.returncode != 1:
                        raise ValueError(f'无法判断允许文档的忽略状态：{name}')
            if any(not name.endswith('.md') for name in allowed):
                raise ValueError('允许文件须为已确认的 Markdown 设计文档')
            payload = {'root': str(root), 'allowed': allowed, 'files': current}
            with baseline.open('x', encoding='utf-8') as handle:
                json.dump(payload, handle, ensure_ascii=False)
            print('基线已记录；覆盖 Git 跟踪及非忽略文件、子模块引用和检出版本，以及子模块内可检查文件；不覆盖忽略文件和项目外文件')
            return 0
        if args.allow:
            raise ValueError('检查时沿用基线中的允许文件，不能重新扩大范围')
        payload = json.loads(baseline.read_text(encoding='utf-8'))
        if payload['root'] != str(root):
            raise ValueError('基线不属于当前目标项目')
        for value in payload['allowed']:
            allowed_path(root, value)
        changed, unexpected = compare(payload['files'], inventory(root), payload['allowed'])
        for name in changed:
            print(('ERROR: 越界改动 ' if name in unexpected else '允许改动 ') + name)
        print('范围检查完成；忽略文件、项目外文件及执行中改后恢复的动作未验证')
        return 1 if unexpected else 0
    except (OSError, ValueError, KeyError, subprocess.CalledProcessError, subprocess.TimeoutExpired) as exc:
        print(f'ERROR: 无法检查：{exc}', file=sys.stderr)
        return 2


if __name__ == '__main__':
    sys.exit(main())
