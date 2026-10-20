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
        self.path = Path(self.tmp.name) / 'model.md'
        self.contract = json.loads((SKILL / 'references/artifact-contract.json').read_text())
        self.text = (SKILL / 'scripts/tests/fixtures/business-model.md').read_text()

    def check(self, text, kind='business-model'):
        self.path.write_text(text)
        return validator.validate(self.path, kind, self.contract)

    def test_three_layers_without_metadata(self):
        self.assertEqual(self.check(self.text), [])

    def test_missing_duplicate_and_reordered_layers(self):
        cases = [
            self.text.replace('## 业务行为', '### 业务行为'),
            self.text + '\n## 业务实体\n\n重复内容\n',
            self.text.replace('## 业务实体', '## 业务流程').replace('## 业务流程\n\n用户选择', '## 业务实体\n\n用户选择'),
        ]
        for text in cases:
            with self.subTest(text=text):
                self.assertTrue(self.check(text))

    def test_extra_layers_and_legacy_format_are_rejected(self):
        for heading in ('需求依据', '实体关系', '业务概念', '文档信息', '验收场景'):
            with self.subTest(heading=heading):
                self.assertTrue(self.check(self.text + '\n## ' + heading + '\n\n附加内容\n'))
        legacy = self.text.replace('## 业务实体', '## 领域术语\n\n术语解释\n\n## 实体与关系')
        self.assertTrue(self.check(legacy))

    def test_empty_layer_placeholder_and_unknown_kind(self):
        self.assertTrue(self.check('# 业务模型\n\n## 业务实体\n\n## 业务行为\n\n行为\n\n## 业务流程\n\n流程\n'))
        self.assertTrue(self.check(self.text.replace('参与购物的人', '{{填写}}')))
        self.assertTrue(self.check(self.text, 'unknown'))

    def test_markdown_and_code_fences(self):
        self.assertTrue(self.check(self.text.rstrip()))
        self.assertTrue(self.check(self.text + '\n'))
        code = self.text + '\n```python\n## 代码标题\n```\n'
        self.assertEqual(self.check(code), [])
        self.assertTrue(self.check(code.replace('```python', '```', 1)))
        self.assertTrue(self.check(self.text + '\n```python\nprint(1)\n'))

    def test_local_links_and_fragments(self):
        self.assertTrue(self.check(self.text + '\n[依据](missing.md)\n'))
        (self.path.parent / 'rules.md').write_text('# 规则\n\n## 支付\n\n具体规则\n')
        self.assertEqual(self.check(self.text + '\n[依据](rules.md#支付)\n'), [])
        self.assertTrue(self.check(self.text + '\n[依据](rules.md#不存在)\n'))

    def test_cli_from_unrelated_directory(self):
        self.path.write_text(self.text)
        command = [sys.executable, '-B', str(SCRIPT), '--kind', 'business-model', '--file', str(self.path)]
        result = subprocess.run(command, cwd=self.tmp.name, capture_output=True, text=True)
        self.assertEqual(result.returncode, 0, result.stderr + result.stdout)
        self.path.write_text(self.text + '\n## 实体关系\n\n附加内容\n')
        result = subprocess.run(command, cwd=self.tmp.name, capture_output=True, text=True)
        self.assertEqual(result.returncode, 1, result.stderr + result.stdout)


if __name__ == '__main__':
    unittest.main()
