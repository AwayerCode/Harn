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

test('installed SWE includes shared standards and new/legacy validators', async () => {
  const { installSkills } = await import('../bin/install.mjs');
  const { spawnSync } = await import('node:child_process');
  const { writeFile } = await import('node:fs/promises');
  const stage = await mkdtemp(path.join(tmpdir(), 'skills-shared-stage-'));
  const project = await mkdtemp(path.join(tmpdir(), 'skills-shared-project-'));
  const pkg = await stagePackage(stage);
  await installSkills({ packageRoot: stage, projectRoot: project, packageJson: pkg });
  const swe = path.join(project, '.agents', 'skills', 'swe');
  await access(path.join(swe, 'references', 'artifact-schemas', 'interface-protocol.json'));
  await access(path.join(swe, 'references', 'standards', 'engineering.md'));
  await access(path.join(swe, 'swe-architecture-design', 'workflow.md'));
  await access(path.join(swe, 'swe-engineering-standards', 'workflow.md'));
  await assert.rejects(access(path.join(swe, 'scripts', 'tests', 'test_standards.py')));
  const helper = path.resolve(import.meta.dirname, '../../swe/scripts/tests');
  const create = spawnSync('python3', ['-B', '-c', 'import sys; from pathlib import Path; sys.path.insert(0, sys.argv[1]); from test_standards import fixture; fixture(Path(sys.argv[2]))', helper, project], { encoding: 'utf8' });
  assert.equal(create.status, 0, create.stderr + create.stdout);
  const designArgs = ['-B', path.join(swe, 'scripts', 'check_design.py'), '--system-design', path.join(project, 'system.md'), '--interface-protocol', path.join(project, 'protocol.md')];
  const design = spawnSync('python3', designArgs, { cwd: project, encoding: 'utf8' });
  assert.equal(design.status, 0, design.stderr + design.stdout);
  const system = await readFile(path.join(project, 'system.md'), 'utf8');
  await writeFile(path.join(project, 'system.md'), system.replace('抽象:order.read', 'Store'));
  const invalidDesign = spawnSync('python3', designArgs, { cwd: project, encoding: 'utf8' });
  assert.equal(invalidDesign.status, 1, invalidDesign.stderr + invalidDesign.stdout);
  const source = path.resolve(import.meta.dirname, '../../swe/swe-code-impl/scripts/tests/fixtures/interface-protocol.md');
  const fixture = await readFile(source, 'utf8');
  const target = path.join(project, 'protocol.md');
  for (const module of ['swe-code-impl', 'swe-module-design', 'swe-business-impl', 'swe-infrastructure-impl', 'swe-runtime-impl']) {
    const args = ['-B', path.join(swe, module, 'scripts', 'validate_documents.py'), '--kind', 'interface-protocol', '--file', target];
    await writeFile(target, fixture);
    const good = spawnSync('python3', args, { cwd: project, encoding: 'utf8' });
    assert.equal(good.status, 0, good.stderr + good.stdout);
    await writeFile(target, fixture.replace('| 生命周期与资源 | 具体内容 |\n', ''));
    const bad = spawnSync('python3', args, { cwd: project, encoding: 'utf8' });
    assert.equal(bad.status, 1, bad.stderr + bad.stdout);
  }
});
