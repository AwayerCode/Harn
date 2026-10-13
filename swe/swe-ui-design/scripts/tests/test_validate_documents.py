import importlib.util
import json
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path

SKILL = Path(__file__).resolve().parents[2]
SCRIPT = SKILL / 'scripts/validate_documents.py'
SPEC = importlib.util.spec_from_file_location('validator', SCRIPT)
validator = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(validator)


class DocumentValidationTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.path = Path(self.tmp.name) / 'doc.md'
        self.contract = {'documents': {}}
        for name in ('input-contract.json', 'artifact-contract.json'):
            source = SKILL / 'references' / name
            if source.exists():
                self.contract['documents'].update(json.loads(source.read_text())['documents'])
        self.kind = next(iter(self.contract['documents']))
        self.text = self.fixture(self.kind)

    def fixture(self, kind):
        return (SKILL / 'scripts/tests/fixtures' / (kind + '.md')).read_text()

    def check(self, text, kind=None):
        self.path.write_text(text)
        return validator.validate(self.path, kind or self.kind, self.contract)

    def test_all_accepted_documents(self):
        for kind in self.contract['documents']:
            with self.subTest(kind=kind):
                self.assertEqual(self.check(self.fixture(kind), kind), [])

    def test_missing_and_duplicate_sections(self):
        heading = self.contract['documents'][self.kind]['sections'][1]['heading']
        self.assertTrue(self.check(self.text.replace('## ' + heading, '### ' + heading)))
        self.assertTrue(self.check(self.text + '\n## ' + heading + '\n\n重复内容\n'))

    def test_table_values_and_escaped_pipe(self):
        self.assertTrue(self.check(self.text.replace('| 任务 | 具体内容 |', '| 任务 |  |')))
        self.assertEqual(self.check(self.text.replace('| 任务 | 具体内容 |', r'| 任务 | 查询 A \| B |')), [])

    def test_placeholder_unknown_kind_and_status(self):
        self.assertTrue(self.check(self.text.replace('具体内容', '{{填写}}', 1)))
        self.assertTrue(self.check(self.text, 'unknown'))
        self.assertTrue(self.check(self.text.replace('| 状态 | 草案 |', '| 状态 | 完成 |')))

    def test_code_fences(self):
        text = self.text + '\n```python\n# 代码标题\n## 代码章节\n```\n'
        self.assertEqual(self.check(text), [])
        self.assertTrue(self.check(text.replace('```python', '```', 1)))

    def test_local_link_and_fragment(self):
        self.assertTrue(self.check(self.text + '\n[依据](missing.md)\n'))
        (self.path.parent / 'model.md').write_text('# 业务模型\n\n## 状态规则\n\n有效内容\n')
        self.assertEqual(self.check(self.text + '\n[依据](model.md#状态规则)\n'), [])
        self.assertTrue(self.check(self.text + '\n[依据](model.md#不存在)\n'))

    def test_reasoned_inapplicability(self):
        spec = self.contract['documents'][self.kind]
        section = next(s for s in spec['sections'] if s.get('columns') and s['heading'] != '文档信息')
        start = self.text.index('## ' + section['heading'])
        end = self.text.find('\n## ', start + 1)
        end = len(self.text) if end == -1 else end
        replacement = '## ' + section['heading'] + '\n\n不适用：本次范围无需此能力。\n'
        self.assertEqual(self.check(self.text[:start] + replacement + self.text[end:]), [])

    def test_protocol_fields_when_accepted(self):
        if 'interface-protocol' not in self.contract['documents']:
            self.skipTest('本 Skill 不接受接口协议')
        text = self.fixture('interface-protocol')
        self.assertEqual(self.check(text, 'interface-protocol'), [])
        self.assertTrue(self.check(text.replace('| 生命周期与资源 | 具体内容 |\n', ''), 'interface-protocol'))

    def test_cli_from_unrelated_directory(self):
        self.path.write_text(self.text)
        result = subprocess.run([sys.executable, str(SCRIPT), '--kind', self.kind, '--file', str(self.path)], cwd=self.tmp.name, capture_output=True, text=True)
        self.assertEqual(result.returncode, 0, result.stderr + result.stdout)
        self.path.write_text(self.text.replace('| 状态 | 草案 |', '| 状态 | 完成 |'))
        result = subprocess.run([sys.executable, str(SCRIPT), '--kind', self.kind, '--file', str(self.path)], cwd=self.tmp.name, capture_output=True, text=True)
        self.assertEqual(result.returncode, 1, result.stderr + result.stdout)


if __name__ == '__main__':
    unittest.main()
