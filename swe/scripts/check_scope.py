#!/usr/bin/env python3
"""Fixed-scope Git file baselines or in-context read-only fingerprints."""
import argparse
import hashlib
import json
import os
import stat
import subprocess
import sys
from pathlib import Path


def git(root, *arguments, text=False):
    try:
        return subprocess.run(['git', '-C', str(root), *arguments], check=True,
                              capture_output=True, text=text, timeout=10)
    except subprocess.TimeoutExpired as exc:
        raise ValueError(f'Git 检查超过 10 秒，未完成：{root}') from exc
    except subprocess.CalledProcessError as exc:
        detail = exc.stderr if text else exc.stderr.decode('utf-8', errors='replace')
        raise ValueError(f'Git 检查失败：{root}；{detail.strip()}') from exc


def gitlinks(root):
    listing = git(root, 'ls-files', '--stage', '-z').stdout
    links = {}
    for entry in listing.split(b'\0'):
        if not entry:
            continue
        metadata, name = entry.split(b'\t', 1)
        mode, commit, stage = metadata.split()
        if mode == b'160000':
            links.setdefault(os.fsdecode(name), []).append([os.fsdecode(stage), os.fsdecode(commit)])
    return links


def directory_files(root, path):
    """List files without following symlinks or reading Git internal files."""
    names = set()
    def fail(error):
        raise error

    for parent, directories, children in os.walk(path, followlinks=False, onerror=fail):
        directories[:] = [name for name in directories if name != '.git']
        children = [name for name in children if name != '.git']
        for name in children + [name for name in directories if (Path(parent) / name).is_symlink()]:
            names.add(os.fsencode((Path(parent) / name).relative_to(root).as_posix()))
    return names


def inventory(root, extras=()):
    git_root = Path(git(root, 'rev-parse', '--show-toplevel', text=True).stdout.strip()).resolve()
    if git_root != root:
        raise ValueError('使用目标项目 Git 根目录，不能只记录子目录')
    output = git(root, 'ls-files', '-z', '--cached', '--others', '--exclude-standard').stdout
    names = set(output.split(b'\0')) - {b''}
    links = gitlinks(root)
    names.update(os.fsencode(name) for name in links)
    result = {}
    for relative, commits in links.items():
        path = root / relative
        if any(parent.is_symlink() for parent in path.parents if parent != root and root in parent.parents):
            raise ValueError(f'子模块父目录含符号链接：{relative}')
        if not path.exists() and not path.is_symlink():
            result[relative] = ['gitlink', None, sorted(commits), None]
            names.discard(os.fsencode(relative))
            continue
        if path.is_symlink() or not path.is_dir():
            # Replacements and deletions are compared against the old gitlink.
            continue
        child_root = git(path, 'rev-parse', '--show-toplevel', text=True).stdout.strip()
        if Path(child_root).resolve() == path:
            head = git(path, 'rev-parse', '--verify', 'HEAD', text=True).stdout.strip()
            for name, entry in inventory(path).items():
                result[relative + '/' + name] = entry
        else:
            head = None
            # An uninitialized gitlink may still contain ordinary files.
            names.update(directory_files(root, path))
        result[relative] = ['gitlink', stat.S_IMODE(path.lstat().st_mode), sorted(commits), head]
        names.discard(os.fsencode(relative))
    for relative in extras:
        path = root / relative
        if path.is_symlink() or path.is_file():
            names.add(os.fsencode(relative))
        elif path.is_dir():
            names.update(directory_files(root, path))
    for raw in sorted(names):
        relative = os.fsdecode(raw)
        path = root / relative
        for parent in path.parents:
            if parent == root:
                break
            if parent.is_symlink():
                raise ValueError(f'文件父目录含符号链接：{relative}')
        try:
            info = path.lstat()
        except FileNotFoundError:
            continue
        if stat.S_ISLNK(info.st_mode):
            digest, kind = hashlib.sha256(os.fsencode(os.readlink(path))).hexdigest(), 'symlink'
        elif stat.S_ISREG(info.st_mode):
            hasher = hashlib.sha256()
            with path.open('rb') as handle:
                for block in iter(lambda: handle.read(1024 * 1024), b''):
                    hasher.update(block)
            digest, kind = hasher.hexdigest(), 'file'
        else:
            raise ValueError(f'不支持的文件类型：{relative}')
        result[relative] = [kind, stat.S_IMODE(info.st_mode), digest]
    return result


def normalize(root, value):
    path = Path(os.path.abspath(root / value))
    relative = path.relative_to(root).as_posix()
    if relative == '.' or relative == '.git' or relative.startswith('.git/'):
        raise ValueError('不得将全仓或 Git 内部文件列为可写范围')
    for parent in [path, *path.parents]:
        if parent == root:
            break
        if parent.is_symlink():
            raise ValueError(f'允许路径包含符号链接：{value}')
    return relative


def check_submodule_scope(files, allowed_files, allowed_dirs):
    for name, entry in files.items():
        if entry[0] != 'gitlink':
            continue
        if any(path == name or path.startswith(name + '/') for path in allowed_files + allowed_dirs) or any(
                name.startswith(directory + '/') for directory in allowed_dirs):
            raise ValueError(f'可写范围跨入子模块 {name}；以该子模块的 Git 根目录单独建立基线')


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('mode', choices=['snapshot', 'check', 'readonly'])
    parser.add_argument('--root', type=Path, required=True)
    parser.add_argument('--baseline', type=Path)
    parser.add_argument('--allow-file', action='append', default=[])
    parser.add_argument('--allow-dir', action='append', default=[])
    parser.add_argument('--expected')
    args = parser.parse_args()
    try:
        root = args.root.resolve(strict=True)
        coverage = 'Git 跟踪及非忽略文件的最终内容、类型和权限；包含子模块引用、检出版本及已初始化子模块的跟踪和非忽略文件，未初始化子模块中的现有普通文件也纳入；不覆盖 .git、忽略文件、项目外文件及改后恢复的动作'
        if args.mode == 'readonly':
            if args.baseline or args.allow_file or args.allow_dir:
                raise ValueError('只读核对不接受基线文件或可写范围')
            if args.expected and (len(args.expected) != 64 or any(c not in '0123456789abcdef' for c in args.expected)):
                raise ValueError('expected 须为原始 SHA256')
            files = inventory(root)
            digest = hashlib.sha256(json.dumps([str(root), files], sort_keys=True).encode()).hexdigest()
            changed = bool(args.expected and args.expected != digest)
            print(json.dumps({'sha256': digest, 'files': len(files), 'changed': changed,
                              'baseline': args.expected is None, 'coverage': coverage}, ensure_ascii=False))
            return 1 if changed else 0
        if args.expected or not args.baseline:
            raise ValueError('写入范围检查需要 baseline，不使用 expected')
        baseline = args.baseline.resolve()
        if baseline == root or root in baseline.parents:
            raise ValueError('基线保存在项目外的全新临时文件')
        if args.mode == 'snapshot':
            allowed_files = [normalize(root, value) for value in args.allow_file]
            allowed_dirs = [normalize(root, value) for value in args.allow_dir]
            if not allowed_files and not allowed_dirs:
                raise ValueError('明确列出本次可写文件或职责独立目录')
            files = inventory(root, allowed_files + allowed_dirs)
            check_submodule_scope(files, allowed_files, allowed_dirs)
            with baseline.open('x', encoding='utf-8') as handle:
                json.dump({'root': str(root), 'files': files, 'allow_files': allowed_files,
                           'allow_dirs': allowed_dirs}, handle, ensure_ascii=True)
            print(json.dumps({'baseline': 'created', 'coverage': coverage + '；写入基线还包括明确允许范围中的忽略文件'}, ensure_ascii=False))
            return 0
        if args.allow_file or args.allow_dir:
            raise ValueError('检查时不得重新扩大允许范围')
        before = json.loads(baseline.read_text())
        if before['root'] != str(root):
            raise ValueError('基线根目录不一致')
        allowed_files = [normalize(root, value) for value in before['allow_files']]
        allowed_dirs = [normalize(root, value) for value in before['allow_dirs']]
        files = inventory(root, allowed_files + allowed_dirs)
        check_submodule_scope(files, allowed_files, allowed_dirs)
        coverage += '；写入检查还覆盖明确允许的文件和运行目录中的忽略文件'
        changed = sorted(name for name in set(files) | set(before['files']) if files.get(name) != before['files'].get(name))
        unexpected = [name for name in changed if name not in allowed_files and not any(
            name == directory or name.startswith(directory + '/') for directory in allowed_dirs)]
        print(json.dumps({'changed': changed, 'unexpected': unexpected, 'coverage': coverage}, ensure_ascii=False, indent=2))
        return 1 if unexpected else 0
    except (OSError, ValueError, KeyError, subprocess.CalledProcessError) as exc:
        print(f'ERROR: 无法核对：{exc}', file=sys.stderr)
        return 2


if __name__ == '__main__':
    sys.exit(main())
