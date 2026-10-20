#!/usr/bin/env python3
"""Validate SWE Markdown artifacts against each skill's declarative contract."""
import argparse
import json
import re
import sys
from pathlib import Path
from urllib.parse import unquote, urlsplit


def cells(line):
    return [v.strip().replace(r'\|', '|') for v in re.split(r'(?<!\\)\|', line.strip().strip('|'))]


def tables(lines):
    found = []
    i = 0
    while i + 1 < len(lines):
        if lines[i].strip().startswith('|') and re.fullmatch(r'\s*\|(?:\s*:?-{3,}:?\s*\|)+\s*', lines[i + 1]):
            head = cells(lines[i])
            rows = []
            i += 2
            while i < len(lines) and lines[i].strip().startswith('|'):
                rows.append(cells(lines[i]))
                i += 1
            found.append((head, rows))
        else:
            i += 1
    return found


def outside_code(text, errors):
    result = []
    fence = None
    for n, line in enumerate(text.splitlines(), 1):
        match = re.match(r'^\s*(`{3,}|~{3,})(.*)$', line)
        if match:
            token, suffix = match.groups()
            if fence is None:
                fence = token
                if not suffix.strip():
                    errors.append(f'{n}: 代码块必须声明语言')
            elif token[0] == fence[0] and len(token) >= len(fence) and not suffix.strip():
                fence = None
            result.append('')
        else:
            result.append('' if fence else line)
    if fence:
        errors.append('代码块未关闭')
    return result


def check_table(lines, columns, errors, label, enums=None):
    candidates = [rows for head, rows in tables(lines) if head == columns]
    if len(candidates) != 1:
        errors.append(f'{label}: 必须有且仅有一个字段为 {columns} 的表格')
        return []
    rows = candidates[0]
    if not rows:
        errors.append(f'{label}: 表格必须包含数据行')
    for row in rows:
        if len(row) != len(columns) or any(not value for value in row):
            errors.append(f'{label}: 表格列数错误或存在空字段')
            continue
        for field, allowed in (enums or {}).items():
            if row[columns.index(field)] not in allowed:
                errors.append(f'{label}: {field} 必须是 {allowed}')
    return rows


def validate(path, kind, contract, profile=None):
    contract = resolve_contract(contract, [kind])
    errors = []
    if kind not in contract['documents']:
        return [f'未知文档类型：{kind}']
    if not path.is_file():
        return [f'文件不存在：{path}']
    text = path.read_text(encoding='utf-8')
    spec = contract['documents'][kind]
    if profile is not None:
        owned = spec.get('profiles', {}).get(profile)
        if not owned:
            raise ValueError(f'文档类型 {kind} 不支持局部校验 {profile}')
        spec = dict(spec)
        spec['sections'] = [section for section in spec['sections'] if section['heading'] in owned]
        # Project the owned sections in memory; no writes or temporary document.
        raw = text.splitlines()
        visible = outside_code(text, errors)
        selected, keep = [], True
        for line, clean in zip(raw, visible):
            if clean.startswith('## '):
                keep = clean[3:].strip() in owned
            if keep:
                selected.append(line)
        original_ending = text.endswith('\n') and not text.endswith('\n\n')
        text = '\n'.join(selected).rstrip() + '\n'
        if not original_ending:
            errors.append('文件末尾必须恰好一个换行')
    if not text.endswith('\n') or text.endswith('\n\n'):
        errors.append('文件末尾必须恰好一个换行')
    if re.search(r'\{\{[^\n{}]+\}\}|\{填写[^\n{}]*\}', text):
        errors.append('存在未替换的模板占位符')
    lines = outside_code(text, errors)
    headings = []
    previous = 0
    for i, line in enumerate(lines):
        match = re.match(r'^(#{1,6}) (.+?)\s*$', line)
        if not match:
            continue
        level, title = len(match[1]), match[2]
        headings.append((i, level, title))
        if level > previous + 1:
            errors.append(f'{i + 1}: 标题层级跳跃')
        previous = level
        if i > 0 and lines[i - 1].strip():
            errors.append(f'{i + 1}: 标题前缺少空行')
        if i + 1 < len(lines) and lines[i + 1].strip():
            errors.append(f'{i + 1}: 标题后缺少空行')
    if [title for _, level, title in headings if level == 1] != [spec['title']]:
        errors.append(f'必须只有一个一级标题：{spec["title"]}')
    expected = [s['heading'] for s in spec['sections']]
    if [title for _, level, title in headings if level == 2] != expected:
        errors.append(f'二级章节必须完整且按此顺序：{expected}')
    chunks = {}
    h2 = [(i, title) for i, level, title in headings if level == 2]
    for index, (start, title) in enumerate(h2):
        end = h2[index + 1][0] if index + 1 < len(h2) else len(lines)
        chunks[title] = lines[start + 1:end]
    for section in spec['sections']:
        title = section['heading']
        body = chunks.get(title, [])
        content = '\n'.join(l for l in body if not l.startswith('#')).strip()
        if not content:
            errors.append(f'{title}: 内容不能为空')
        # Explicitly inapplicable sections have no fabricated rows.
        na = content.startswith('不适用：') and len(content.removeprefix('不适用：').strip()) > 0
        if 'columns' in section and not (na and title != '文档信息'):
            check_table(body, section['columns'], errors, title, section.get('enums'))
    metadata = [rows for head, rows in tables(chunks.get('文档信息', [])) if head == ['字段', '值']]
    if len(metadata) == 1:
        rows = metadata[0]
        keys = ['类型', '任务', '范围', '依据', '状态']
        if [row[0] for row in rows if row] != keys:
            errors.append(f'文档信息字段必须为：{keys}')
        values = {row[0]: row[1] for row in rows if len(row) == 2}
        if values.get('类型') != kind:
            errors.append('文档信息类型与所选契约不一致')
        if values.get('状态') not in ['草案', '待验收', '已验收']:
            errors.append('状态必须为草案、待验收或已验收')
    if 'record_fields' in spec:
        body = chunks.get('协议定义', [])
        records = [(i, l[4:]) for i, l in enumerate(body) if l.startswith('### ')]
        if not records and not '\n'.join(body).strip().startswith('不适用：'):
            errors.append('协议定义: 必须有能力三级标题及字段表')
        for index, (start, name) in enumerate(records):
            end = records[index + 1][0] if index + 1 < len(records) else len(body)
            rows = check_table(body[start + 1:end], ['字段', '值'], errors, name)
            if [r[0] for r in rows if r] != spec['record_fields']:
                errors.append(f'{name}: 协议字段缺失、重复或顺序错误')
    # Validate local inline link targets without network requests.
    visible = '\n'.join(lines)
    for target in re.findall(r'!?\[[^\]\n]*\]\((<[^>]+>|[^\s)]+)\)', visible):
        target = target.strip('<>')
        url = urlsplit(target)
        if url.scheme or url.netloc:
            continue
        local = path if not url.path else path.parent / unquote(url.path)
        if not local.exists():
            errors.append(f'本地引用不存在：{target}')
        elif url.fragment and local.suffix == '.md':
            other_errors = []
            other = outside_code(local.read_text(encoding='utf-8'), other_errors)
            anchors = []
            for line in other:
                match = re.match(r'^#{1,6} (.+)$', line)
                if match:
                    slug = re.sub(r'[^\w\s-]', '', match[1].lower()).replace(' ', '-')
                    anchors.append(slug)
            if unquote(url.fragment) not in anchors:
                errors.append(f'章节引用不存在：{target}')
    return errors


def resolve_contract(contract, kinds=None):
    """Resolve versioned package schemas; unknown/missing schemas fail closed."""
    if contract.get('schema_version', 1) != 1:
        raise ValueError('不支持的格式契约版本')
    result = {'documents': {}}
    for kind, spec in contract['documents'].items():
        if kinds is not None and kind not in kinds:
            continue
        if '$ref' in spec:
            name = spec['$ref']
            if name != kind + '.json' or Path(name).name != name:
                raise ValueError(f'非法格式契约引用：{name}')
            source = Path(__file__).resolve().parents[1] / 'references/artifact-schemas' / name
            spec = json.loads(source.read_text(encoding='utf-8'))
        result['documents'][kind] = spec
    return result


def load_contract(module=None, kinds=None):
    if module is None:
        root = Path(__file__).resolve().parents[1] / 'references/artifact-schemas'
        return {'documents': {p.stem: json.loads(p.read_text(encoding='utf-8')) for p in sorted(root.glob('*.json')) if kinds is None or p.stem in kinds}}
    contract = {'documents': {}}
    for name in ('input-contract.json', 'artifact-contract.json'):
        source = Path(module) / 'references' / name
        if source.is_file():
            data = json.loads(source.read_text(encoding='utf-8'))
            if data.get('schema_version') != 1:
                raise ValueError('不支持的格式契约版本')
            contract['documents'].update(data['documents'])
    return resolve_contract(contract, kinds)


def main(module=None):
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--kind', required=True)
    parser.add_argument('--file', type=Path, required=True)
    parser.add_argument('--profile', choices=['architecture'], help='仅检查架构负责章节；通过不表示整份设计通过')
    args = parser.parse_args()
    try:
        errors = validate(args.file, args.kind, load_contract(module, [args.kind]), args.profile)
    except (OSError, ValueError, KeyError) as exc:
        print(f'ERROR: 无法完成校验：{exc}', file=sys.stderr)
        return 2
    for error in errors:
        print(f'ERROR: {args.file}: {error}')
    scope = 'partial:' + args.profile if args.profile else 'full'
    print(f'Checked {args.file} [{scope}]: {len(errors)} error(s)')
    return 1 if errors else 0


if __name__ == '__main__':
    sys.exit(main())
