#!/usr/bin/env python3
"""Read-only format and cross-document checks for system design artifacts."""
import argparse
import json
import re
import sys
from pathlib import Path
from urllib.parse import unquote, urlsplit

from validate_documents import outside_code, tables, validate, load_contract


def names(value):
    return [x.strip().strip('`') for x in re.split(r'[、,，;；]', value) if x.strip()]


def sections(path):
    result = {}
    current = None
    for line in outside_code(path.read_text(encoding='utf-8'), []):
        if line.startswith('## '):
            current = line[3:]
            result[current] = []
        elif current:
            result[current].append(line)
    return result


def rows(body, columns, errors, label, optional=False):
    if optional and '\n'.join(body).strip().startswith('不适用：'):
        return []
    found = [data for head, data in tables(body) if head == columns]
    if len(found) != 1:
        errors.append(f'{label}: 缺少可检查的结构化表格；旧格式需补齐当前范围依据')
        return []
    return [dict(zip(columns, row)) for row in found[0] if len(row) == len(columns)]


def unique(records, key, errors):
    result = {}
    for row in records:
        value = row[key].strip('`')
        if value in result:
            errors.append(f'{key}重复：{value}')
        result[value] = row
    return result


def links_to(source, destination):
    visible = '\n'.join(outside_code(source.read_text(encoding='utf-8'), []))
    for target in re.findall(r'\[[^\]\n]*\]\((<[^>]+>|[^\s)]+)\)', visible):
        url = urlsplit(target.strip('<>'))
        if not url.scheme and not url.netloc and url.path:
            if (source.parent / unquote(url.path)).resolve() == destination.resolve():
                return True
    return False


def check(system, protocol):
    contract = load_contract(kinds=['system-design', 'interface-protocol'])
    errors = []
    for path, kind in [(system, 'system-design'), (protocol, 'interface-protocol')]:
        errors.extend(f'{path.name}: {e}' for e in validate(path, kind, contract))
    if errors:
        return errors
    ds, ps = sections(system), sections(protocol)
    module_rows = rows(ds.get('模块与依赖', []),
                       ['模块', '责任层', '职责', '公开能力', '允许依赖', '能力来源'], errors, '模块与依赖')
    modules = unique(module_rows, '模块', errors)
    index_rows = rows(ps.get('协议索引', []),
                      ['协议标识', '能力', '提供方', '调用方', '边界类型', '交互方式'], errors, '协议索引', True)
    index = unique(index_rows, '协议标识', errors)
    body = ps.get('协议定义', [])
    starts = [i for i, line in enumerate(body) if line.startswith('### ')]
    records = []
    for position, start in enumerate(starts):
        end = starts[position + 1] if position + 1 < len(starts) else len(body)
        record_tables = [data for head, data in tables(body[start + 1:end]) if head == ['字段', '值']]
        if record_tables:
            records.append(dict(record_tables[0]))
    definitions = unique(records, '协议标识', errors)
    if set(index) != set(definitions):
        errors.append(f'协议索引与定义不对应：{sorted(set(index) ^ set(definitions))}')
    for pid, item in index.items():
        if not re.fullmatch(r'[A-Za-z0-9_.-]+', pid):
            errors.append(f'协议标识须使用字母、数字、点、下划线或连字符：{pid}')
        if item['边界类型'] not in {'模块', '外部', '持久化'}:
            errors.append(f'{pid}: 未知边界类型 {item["边界类型"]}')
        for field in ['提供方', '调用方', '边界类型', '交互方式']:
            if pid in definitions and item[field] != definitions[pid].get(field):
                errors.append(f'{pid}: 索引与定义的{field}不一致')
        for field in ['提供方', '调用方']:
            for name in names(item[field]):
                allowed_external = item['边界类型'] == '外部' and name.startswith('外部:') and len(name) > 3
                allowed_storage = item['边界类型'] == '持久化' and name.startswith('存储:') and len(name) > 3
                if name not in modules and not allowed_external and not allowed_storage:
                    errors.append(f'{pid}: {field}引用未定义模块或错误边界主体 {name}')
        if not any(name in modules for field in ['提供方', '调用方'] for name in names(item[field])):
            errors.append(f'{pid}: 协议没有当前系统的参与模块')
    graph = {name: [] for name in modules}
    layers = {'交互层', '业务编排层', '业务能力层', '技术实现层', '横切基础设施', '运行组装'}
    for name, module in modules.items():
        if module['责任层'] not in layers:
            errors.append(f'{name}: 未知责任层 {module["责任层"]}')
        for dependency in names(module['允许依赖']):
            if dependency == '无':
                continue
            if dependency.startswith('抽象:'):
                pid = dependency[3:]
                if pid not in index or name not in names(index[pid]['调用方']):
                    errors.append(f'{name}: 抽象依赖没有相符的协议与调用方 {dependency}')
                elif module['责任层'] == '业务能力层' and any(
                        modules.get(provider, {}).get('责任层') == '业务能力层'
                        for provider in names(index[pid]['提供方'])):
                    errors.append(f'{name}: 抽象引用仍依赖平级业务能力，须重新划分责任')
                continue
            if dependency not in modules:
                errors.append(f'{name}: 依赖未定义模块 {dependency}')
                continue
            graph[name].append(dependency)
            target = modules[dependency]['责任层']
            if module['责任层'] == target == '业务能力层':
                errors.append(f'{name}: 平级业务能力直接依赖 {dependency}')
            if module['责任层'] in {'业务编排层', '业务能力层'} and target == '技术实现层':
                errors.append(f'{name}: 业务直接依赖具体技术实现 {dependency}，应使用抽象协议')
    visited, active = set(), set()
    def visit(name):
        if name in active:
            errors.append(f'模块依赖存在环：{name}')
            return
        if name in visited:
            return
        active.add(name)
        for target in graph[name]:
            visit(target)
        active.remove(name)
        visited.add(name)
    for name in graph:
        visit(name)
    owners = rows(ds.get('状态与资源所有权', []),
                  ['状态或资源', '所有者', '作用域', '修改权', '生命周期'], errors, '状态与资源所有权', True)
    unique(owners, '状态或资源', errors)
    for item in owners:
        for field in ['所有者', '修改权']:
            if field == '修改权' and item[field] == '无':
                continue
            for name in names(item[field]):
                if name not in modules:
                    errors.append(f'{item["状态或资源"]}: {field}引用未定义模块 {name}')
        if len(names(item['所有者'])) != 1:
            errors.append(f'{item["状态或资源"]}: 须明确唯一责任所有者')
    mapping = rows(ds.get('代码结构映射', []), ['设计责任', '代码位置', '公开入口', '依赖约束'], errors, '代码结构映射')
    mapped = set()
    for item in mapping:
        for name in names(item['设计责任']):
            mapped.add(name)
            if name not in modules:
                errors.append(f'代码结构映射引用未定义模块：{name}')
    for name in sorted(set(modules) - mapped):
        errors.append(f'模块没有代码结构映射：{name}')
    paths = '\n'.join(ds.get('关键协作路径', []))
    path_refs = set(re.findall(r'协议:([A-Za-z0-9_.-]+)', paths))
    if index and not path_refs:
        errors.append('关键协作路径没有协议引用；使用 `协议:标识`')
    for pid in sorted(path_refs - set(index)):
        errors.append(f'关键协作路径引用未定义协议：{pid}')
    choices = rows(ds.get('技术选型', []), ['技术项', '选择', '理由与取舍', '决策状态'], errors, '技术选型', True)
    for item in choices:
        if item['决策状态'] not in {'用户已确认', '设计建议', '待确认'}:
            errors.append(f'{item["技术项"]}: 未知决策状态 {item["决策状态"]}')
    if not links_to(system, protocol) or not links_to(protocol, system):
        errors.append('系统设计与接口协议须以本地 Markdown 链接相互引用')
    return errors


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--system-design', required=True, type=Path)
    parser.add_argument('--interface-protocol', required=True, type=Path)
    args = parser.parse_args()
    try:
        errors = check(args.system_design, args.interface_protocol)
    except (OSError, ValueError, KeyError) as exc:
        print(f'ERROR: 无法检查：{exc}', file=sys.stderr)
        return 2
    for error in errors:
        print(f'ERROR: {error}')
    print(f'设计基本检查：{len(errors)} 项错误；不包含业务语义审查和源码依赖验证')
    return 1 if errors else 0


if __name__ == '__main__':
    sys.exit(main())
