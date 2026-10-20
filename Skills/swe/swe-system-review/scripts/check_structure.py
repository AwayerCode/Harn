#!/usr/bin/env python3
"""Check explicit code locations and public entries; print evidence without writing."""
import argparse
import json
import re
import sys
from pathlib import Path
sys.path.insert(0, str(Path(__file__).resolve().parents[2] / 'scripts'))
from validate_documents import outside_code, tables, validate


def inside(root, text):
    text = text.strip().strip('`')
    if not text or re.search(r'[\s*{}<>]', text) or Path(text).is_absolute():
        raise ValueError(f'无法直接检查的相对路径：{text}')
    path = (root / text).resolve()
    path.relative_to(root)
    return path


def check(root, design, scopes):
    contract = json.loads((Path(__file__).resolve().parents[1] / 'references/input-contract.json').read_text())
    errors = validate(design, 'system-design', contract)
    if errors:
        return {'findings': [], 'gaps': errors, 'unverified': [], 'checked': []}
    lines = outside_code(design.read_text(encoding='utf-8'), [])
    body, active = [], False
    for line in lines:
        if line.startswith('## '):
            active = line == '## 代码结构映射'
        elif active:
            body.append(line)
    specs = [
        ['设计责任', '代码位置', '公开入口', '依赖约束'],
        ['设计责任', '代码位置', '公共入口', '约束'],
    ]
    mappings = [(head, rows) for head, rows in tables(body) if head in specs]
    result = {'findings': [], 'gaps': [], 'unverified': [
        '本脚本不解析源码依赖、导出内容、路由配置或业务逻辑，须按工作流继续审查。'], 'checked': []}
    if len(mappings) != 1:
        result['gaps'].append('设计缺少可检查的代码结构映射表；不能从当前目录反推设计')
        return result
    selected = [inside(root, value) for value in scopes]
    seen = {}
    for head, data in mappings:
        for values in data:
            item = dict(zip(head, values))
            responsibility, raw = item['设计责任'], item['代码位置']
            try:
                location = inside(root, raw)
            except ValueError as exc:
                result['unverified'].append(f'{responsibility}: {exc}')
                continue
            if selected and not any(location == scope or scope in location.parents or location in scope.parents
                                    for scope in selected):
                continue
            if location in seen and seen[location] != responsibility:
                result['gaps'].append(f'同一路径映射多个责任，需确认是否共享：{raw} ({seen[location]}, {responsibility})')
            seen[location] = responsibility
            result['checked'].append(f'{responsibility}: 代码位置 {raw}')
            if not location.exists():
                result['findings'].append(f'{responsibility}: 设计声明的位置不存在 {raw}；结合当前实施范围判断是否违规')
                continue
            entry = item.get('公开入口', item.get('公共入口', ''))
            if entry == '无':
                continue
            if '/' not in entry and not Path(entry.strip('`')).suffix:
                result['unverified'].append(f'{responsibility}: 公开入口 {entry} 是符号或说明，需人工核对')
                continue
            try:
                public = inside(root, entry)
            except ValueError as exc:
                result['unverified'].append(f'{responsibility}: 公开入口 {exc}；导出符号或说明文字需人工检查')
                continue
            result['checked'].append(f'{responsibility}: 公开入口 {entry}')
            if not public.is_file():
                result['findings'].append(f'{responsibility}: 公开入口文件不存在 {entry}')
            base = location if location.is_dir() else location.parent
            if public != location and base not in public.parents:
                result['gaps'].append(f'{responsibility}: 公开入口位于代码位置之外，需确认设计约定 {entry}')
    if not result['checked']:
        result['gaps'].append('当前范围没有可检查的结构映射，不能交付结构一致性通过结论')
    return result


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--root', required=True, type=Path)
    parser.add_argument('--design', required=True, type=Path)
    parser.add_argument('--scope', action='append', default=[])
    args = parser.parse_args()
    try:
        root = args.root.resolve(strict=True)
        result = check(root, args.design, args.scope)
        print(json.dumps(result, ensure_ascii=False, indent=2))
        return 1 if result['findings'] or result['gaps'] else 0
    except (OSError, ValueError, KeyError) as exc:
        print(f'ERROR: 无法检查：{exc}', file=sys.stderr)
        return 2


if __name__ == '__main__':
    sys.exit(main())
