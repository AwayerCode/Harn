import importlib.util
import json
import re
import subprocess
import sys
import tempfile
import shutil
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT / 'scripts'))
import validate_documents as validator
import check_design as design


def document(kind, content):
    schema = validator.load_contract()['documents'][kind]
    out = ['# ' + schema['title'], '']
    for section in schema['sections']:
        title = section['heading']
        out += ['## ' + title, '']
        if title == '文档信息':
            out += ['| 字段 | 值 |', '| --- | --- |', '| 类型 | ' + kind + ' |',
                    '| 任务 | 模块设计 |', '| 范围 | 业务与存储 |',
                    '| 依据 | 已确认业务 |', '| 状态 | 待验收 |']
        else:
            out += [content.get(title, '不适用：当前范围无此责任。')]
        out += ['']
    return '\n'.join(out)


def fixture(root):
    system = root / 'system.md'
    protocol = root / 'protocol.md'
    system.write_text(document('system-design', {
        '设计依据与范围': '[协议](protocol.md)',
        '责任分层': '业务层通过抽象协议使用技术实现，运行组装连接二者。',
        '模块与依赖': '| 模块 | 责任层 | 职责 | 公开能力 | 允许依赖 | 能力来源 |\n| --- | --- | --- | --- | --- | --- |\n| Order | 业务能力层 | 查询订单 | 查询 | 抽象:order.read | 自研 |\n| Store | 技术实现层 | 读存储 | 读取 | 无 | 自研 |',
        '关键协作路径': 'Order 通过 协议:order.read 查询 Store。',
        '代码结构映射': '| 设计责任 | 代码位置 | 公开入口 | 依赖约束 |\n| --- | --- | --- | --- |\n| Order | src/order | src/order/index.py | 抽象协议 |\n| Store | src/store | src/store/index.py | 无 |',
    }))
    fields = validator.load_contract()['documents']['interface-protocol']['record_fields']
    values = {key: '明确约定' for key in fields}
    values.update({'协议标识': 'order.read', '提供方': 'Store', '调用方': 'Order', '边界类型': '模块', '交互方式': '函数',
                   '输入与输出': '订单标识到订单快照', '状态变化与副作用': '纯查询，不修改状态',
                   '验证依据': '隔离环境固定数据集，验证输入、输出与未找到行为'})
    protocol.write_text(document('interface-protocol', {
        '范围与权威来源': '[系统设计](system.md)',
        '协议索引': '| 协议标识 | 能力 | 提供方 | 调用方 | 边界类型 | 交互方式 |\n| --- | --- | --- | --- | --- | --- |\n| order.read | 读取订单 | Store | Order | 模块 | 函数 |',
        '协议定义': '### 查询订单\n\n| 字段 | 值 |\n| --- | --- |\n' + '\n'.join('| ' + key + ' | ' + values[key] + ' |' for key in fields),
    }))
    return system, protocol


class StandardsTests(unittest.TestCase):
    def test_manifests_reference_single_source(self):
        for path in ROOT.glob('*/references/*contract.json'):
            data = json.loads(path.read_text())
            self.assertEqual(data['schema_version'], 1, str(path))
            for kind, spec in data['documents'].items():
                self.assertEqual(spec, {'$ref': kind + '.json'})
            self.assertEqual(set(validator.load_contract(path.parents[1])['documents']),
                             set().union(*(set(json.loads(p.read_text())['documents']) for p in path.parent.glob('*contract.json'))))

    def test_bad_schema_reference_and_version_fail_closed(self):
        for data in [
            {'documents': {'system-design': {'$ref': '../outside.json'}}},
            {'schema_version': 99, 'documents': {}},
            {'documents': {'missing': {'$ref': 'missing.json'}}},
        ]:
            with self.assertRaises((ValueError, OSError)):
                validator.resolve_contract(data)

    def test_module_contract_and_illegal_dependencies(self):
        with tempfile.TemporaryDirectory() as tmp:
            system, protocol = fixture(Path(tmp))
            self.assertEqual(design.check(system, protocol), [])
            original = system.read_text()
            system.write_text(original.replace('抽象:order.read', 'Store'))
            self.assertTrue(any('直接依赖具体技术' in x for x in design.check(system, protocol)))
            system.write_text(original.replace('抽象:order.read', '抽象:missing'))
            self.assertTrue(any('抽象依赖' in x for x in design.check(system, protocol)))
            system.write_text(original)
            protocol.write_text(protocol.read_text().replace('| 调用方 | Order |', '| 调用方 | Unknown |'))
            self.assertTrue(design.check(system, protocol))

    def test_shared_validator_cli_and_no_module_wrappers(self):
        with tempfile.TemporaryDirectory() as tmp:
            system, protocol = fixture(Path(tmp))
            script = ROOT / 'scripts/validate_documents.py'
            command = [sys.executable, '-B', str(script), '--kind', 'interface-protocol', '--file', str(protocol)]
            good = subprocess.run(command, cwd=tmp, capture_output=True, text=True)
            self.assertEqual(good.returncode, 0, good.stdout + good.stderr)
            text = protocol.read_text()
            protocol.write_text(text.replace('| 调用方 | Order |\n', ''))
            bad = subprocess.run(command, cwd=tmp, capture_output=True, text=True)
            self.assertEqual(bad.returncode, 1, bad.stdout + bad.stderr)
            protocol.write_text(text)
            for module in ROOT.glob('swe-*'):
                self.assertFalse((module / 'scripts/validate_documents.py').exists(), str(module))
            missing = subprocess.run([sys.executable, '-B', str(ROOT / 'scripts/validate_documents.py'), '--kind', 'system-design', '--file', str(Path(tmp)/'missing.md')], capture_output=True, text=True)
            self.assertEqual(missing.returncode, 1)

    def test_architecture_handoff_does_not_require_module_implementation(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            system, protocol = fixture(root)
            schema = validator.load_contract(kinds=['system-design'])
            owned = schema['documents']['system-design']['profiles']['architecture']
            original = system.read_text()
            chunks = re.split(r'(?m)(?=^## )', original)
            partial = chunks[0] + ''.join(chunk for chunk in chunks[1:] if chunk.splitlines()[0][3:] in owned)
            system.write_text(partial)
            self.assertEqual(validator.validate(system, 'system-design', schema, 'architecture'), [])
            self.assertTrue(validator.validate(system, 'system-design', schema))
            system.write_text(partial + '\n## 模块与依赖\n\n{{尚未由模块设计填写}} [未完成](missing.md)\n')
            self.assertEqual(validator.validate(system, 'system-design', schema, 'architecture'), [])
            self.assertTrue(validator.validate(system, 'system-design', schema))
            system.write_text(partial.replace('## 责任分层', '## 未定义的层'))
            self.assertTrue(validator.validate(system, 'system-design', schema, 'architecture'))
            system.write_text(original)
            self.assertEqual(design.check(system, protocol), [])
            result = subprocess.run([sys.executable, '-B', str(ROOT / 'scripts/validate_documents.py'), '--kind', 'system-design', '--profile', 'architecture', '--file', str(system)], capture_output=True, text=True)
            self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
            self.assertIn('[partial:architecture]', result.stdout)
            wrong = subprocess.run([sys.executable, '-B', str(ROOT / 'scripts/validate_documents.py'), '--kind', 'interface-protocol', '--profile', 'architecture', '--file', str(protocol)], capture_output=True, text=True)
            self.assertEqual(wrong.returncode, 2)

    def test_shared_checks_load_only_requested_types_without_producer_modules(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp) / 'swe'
            shutil.copytree(ROOT / 'scripts', root / 'scripts', ignore=shutil.ignore_patterns('tests', '__pycache__'))
            shutil.copytree(ROOT / 'references/artifact-schemas', root / 'references/artifact-schemas')
            for kind in ['system-design', 'interface-protocol']:
                (root / 'references/artifact-schemas' / (kind + '.json')).write_text('invalid json')
            fixture = ROOT / 'scripts/tests/fixtures/test-design.md'
            for args in [
                [str(root / 'scripts/validate_documents.py'), '--kind', 'test-design', '--file', str(fixture)],
                [str(root / 'scripts/check_artifacts.py'), '--kind', 'test', '--file', str(fixture)],
            ]:
                result = subprocess.run([sys.executable, '-B', *args], cwd=tmp, capture_output=True, text=True)
                self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
            result = subprocess.run([sys.executable, '-B', str(root / 'scripts/validate_documents.py'), '--kind', 'interface-protocol', '--file', str(fixture)], capture_output=True, text=True)
            self.assertEqual(result.returncode, 2)

    def test_architecture_template_matches_owned_sections(self):
        template = (ROOT / 'swe-architecture-design/assets/architecture-design.md').read_text()
        expected = validator.load_contract(kinds=['system-design'])['documents']['system-design']['profiles']['architecture']
        self.assertEqual(re.findall(r'^## (.+)$', template, re.M), expected)

    def test_ui_only_work_needs_no_business_protocol_but_business_work_does(self):
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / 'ui.md'
            path.write_text(document('interaction-standards', {
                '操作与业务映射': '| 用户操作 | 公共能力 | 可见状态 | 反馈 | 后续动作 |\n| --- | --- | --- | --- | --- |\n| 展开详情 | 界面:展开 | 展开 | 动画 | 收起 |',
            }))
            command = [sys.executable, '-B', str(ROOT / 'scripts/check_artifacts.py'), '--kind', 'ui', '--file', str(path)]
            pure = subprocess.run(command, cwd=tmp, capture_output=True, text=True)
            self.assertEqual(pure.returncode, 0, pure.stdout + pure.stderr)
            self.assertIn('纯界面动作', pure.stdout)
            path.write_text(path.read_text().replace('界面:展开', '协议:order.read'))
            business = subprocess.run(command, cwd=tmp, capture_output=True, text=True)
            self.assertEqual(business.returncode, 1, business.stdout + business.stderr)
            self.assertIn('缺少接口协议', business.stdout)
            path.write_text(path.read_text().replace('协议:order.read', '界面:'))
            invalid = subprocess.run(command, cwd=tmp, capture_output=True, text=True)
            self.assertEqual(invalid.returncode, 1)

    def test_document_links(self):
        for path in ROOT.rglob('*.md'):
            if 'tests' in path.parts:
                continue
            text = '\n'.join(validator.outside_code(path.read_text(), []))
            for link in re.findall(r'\[[^\]\n]*\]\(([^\s)]+)\)', text):
                target = link.split('#')[0]
                if ':' not in target and target:
                    self.assertTrue((path.parent / target).exists(), f'{path}: {link}')


if __name__ == '__main__':
    unittest.main()
