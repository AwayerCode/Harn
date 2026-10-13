import assert from 'node:assert/strict';
import { access, mkdtemp, readFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import path from 'node:path';
import test from 'node:test';
import { stagePackage } from '../build.mjs';

test('stage copies skills and the installer without tests or the index', async () => {
  const dest = await mkdtemp(path.join(tmpdir(), 'skills-stage-'));
  await stagePackage(dest);

  await access(path.join(dest, 'bin', 'install.mjs'));
  await access(path.join(dest, 'concise-response', 'SKILL.md'));
  await access(path.join(dest, 'design-skill', 'SKILL.md'));
  await access(path.join(dest, 'knowledge-doc-guard', 'SKILL.md'));
  await access(path.join(dest, 'swe', 'SKILL.md'));
  await access(path.join(dest, 'swe', 'scripts', 'check_scope.py'));
  await assert.rejects(access(path.join(dest, '0-Skills.md')));
  await assert.rejects(access(path.join(dest, 'publish')));
  await assert.rejects(access(path.join(dest, 'swe', 'swe-code-impl', 'scripts', 'tests', 'test_validate_documents.py')));

  const pkg = JSON.parse(await readFile(path.join(dest, 'package.json'), 'utf8'));
  assert.equal(pkg.name, '@awayercode/skills');
  assert.equal(pkg.version, '0.1.1');
  assert.equal(pkg.license, undefined);
});
