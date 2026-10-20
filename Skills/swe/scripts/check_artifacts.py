#!/usr/bin/env python3
"""Check explicit UI-to-protocol or risk-to-scenario relations, without writing."""
import argparse
import json
import re
import sys
from pathlib import Path


import validate_documents as validator


def references(text, prefix, errors, label):
    """Read complete reference tokens; never accept only a valid prefix."""
    tokens = re.findall(re.escape(prefix) + r':([^\s`、,，;；()（）\[\]。]*)', text)
    valid = []
    for token in tokens:
        if not re.fullmatch(r'[\w.-]+', token):
            errors.append(f'{label}: 非法{prefix}标识 {token or "（空）"}')
        else:
            valid.append(token)
    return valid


def check(kind, path, protocol=None):
    if kind not in {'ui', 'test'}:
        raise ValueError(f'未知检查类型：{kind}')
    dtype = 'interaction-standards' if kind == 'ui' else 'test-design'
    kinds = [dtype, 'interface-protocol'] if kind == 'ui' and protocol is not None else [dtype]
    contract = validator.load_contract(kinds=kinds)
    result = {'errors': validator.validate(path, dtype, contract), 'checked': [], 'unverified': []}
    if result['errors']:
        return result
    lines = validator.outside_code(path.read_text(), [])
    if kind == 'ui':
        ids, capabilities = set(), {}
        if protocol is not None:
            result['errors'] += validator.validate(protocol, 'interface-protocol', contract)
            if result['errors']:
                return result
            for columns, rows in validator.tables(validator.outside_code(protocol.read_text(), [])):
                if '协议标识' in columns and '能力' in columns:
                    for values in rows:
                        row = dict(zip(columns, values))
                        pid = row['协议标识'].strip('`')
                        if pid in ids:
                            result['errors'].append(f'重复协议标识：{pid}')
                        ids.add(pid)
                        capabilities.setdefault(row['能力'], []).append(pid)
        for columns, rows in validator.tables(lines):
            if columns != ['用户操作', '公共能力', '可见状态', '反馈', '后续动作']:
                continue
            for values in rows:
                row = dict(zip(columns, values))
                ability = row['公共能力']
                before = len(result['errors'])
                refs = references(ability, '协议', result['errors'], row['用户操作'])
                if protocol is None and not (ability.startswith('界面:') and ability[3:].strip()):
                    result['errors'].append(f'{row["用户操作"]}: 业务操作缺少接口协议依据')
                elif '协议:' in ability:
                    for pid in refs:
                        if pid not in ids:
                            result['errors'].append(f'{row["用户操作"]}: 引用不存在的协议 {pid}')
                    if len(result['errors']) == before:
                        result['checked'].append(row['用户操作'])
                elif ability.startswith('界面:') and len(ability[3:].strip()) > 0:
                    result['unverified'].append(f'{row["用户操作"]}: 纯界面动作，需审查是否确实不依赖业务能力')
                elif len(capabilities.get(ability, [])) == 1:
                    result['checked'].append(row['用户操作'])
                else:
                    result['unverified'].append(f'{row["用户操作"]}: 公共能力无法唯一对应协议，人工核对')
        result['unverified'].append('可见状态、恢复承诺和验收场景是否符合业务，仍需内容审查')
    else:
        scenario_ids = []
        for line in lines:
            if line.startswith('### 场景:'):
                sid = line.removeprefix('### 场景:').strip()
                if not re.fullmatch(r'[\w.-]+', sid):
                    result['errors'].append(f'非法场景定义标识：{sid or "（空）"}')
                else:
                    scenario_ids.append(sid)
        if len(scenario_ids) != len(set(scenario_ids)):
            result['errors'].append('场景标识重复')
        risks = set()
        for columns, rows in validator.tables(lines):
            if '风险标识' not in columns or '场景' not in columns:
                continue
            for values in rows:
                row = dict(zip(columns, values))
                rid = row['风险标识']
                if rid in risks:
                    result['errors'].append(f'风险标识重复：{rid}')
                risks.add(rid)
                before = len(result['errors'])
                refs = references(row['场景'], '场景', result['errors'], rid)
                if '场景:' in row['场景']:
                    for sid in refs:
                        if sid not in scenario_ids:
                            result['errors'].append(f'{rid}: 场景未定义 {sid}')
                    if len(result['errors']) == before:
                        result['checked'].append(rid)
                else:
                    result['unverified'].append(f'{rid}: 场景为自然语言，人工核对其定义与覆盖')
        result['unverified'].append('风险是否完整、预期是否正确、替身能否证明承诺，仍需内容审查')
    return result


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--kind', choices=['ui', 'test'], required=True)
    parser.add_argument('--file', type=Path, required=True)
    parser.add_argument('--protocol', type=Path)
    args = parser.parse_args()
    try:
        result = check(args.kind, args.file, args.protocol)
        print(json.dumps(result, ensure_ascii=False, indent=2))
        return 1 if result['errors'] else 0
    except (OSError, ValueError, KeyError) as exc:
        print(f'ERROR: 无法检查：{exc}', file=sys.stderr)
        return 2


if __name__ == '__main__':
    sys.exit(main())
