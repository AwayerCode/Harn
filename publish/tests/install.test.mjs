import assert from 'node:assert/strict';
import { mkdtemp, mkdir, readFile, lstat, symlink, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import path from 'node:path';
import test from 'node:test';
import { installSkills } from '../bin/install.mjs';

async function fixturePackage() {
  const packageRoot = await mkdtemp(path.join(tmpdir(), 'skills-pkg-'));
  await mkdir(path.join(packageRoot, 'concise-response'), { recursive: true });
  await writeFile(path.join(packageRoot, 'concise-response', 'SKILL.md'), '# concise\n');
  await mkdir(path.join(packageRoot, 'notes'), { recursive: true });
  await writeFile(path.join(packageRoot, 'notes', 'readme.md'), 'not a skill\n');
  await mkdir(path.join(packageRoot, 'swe', 'scripts', 'tests'), { recursive: true });
  await writeFile(path.join(packageRoot, 'swe', 'SKILL.md'), '# swe\n');
  await writeFile(path.join(packageRoot, 'swe', 'scripts', 'check.py'), 'print(1)\n');
  await writeFile(path.join(packageRoot, 'swe', 'scripts', 'tests', 'test_check.py'), 'assert False\n');
  return packageRoot;
}

test('install writes skill directories and the release file', async () => {
  const packageRoot = await fixturePackage();
  const projectRoot = await mkdtemp(path.join(tmpdir(), 'skills-project-'));
  await installSkills({
    packageRoot,
    projectRoot,
    packageJson: { name: '@awayercode/skills', version: '0.1.0' },
  });

  const skills = path.join(projectRoot, '.agents', 'skills');
  assert.equal(await readFile(path.join(skills, 'concise-response', 'SKILL.md'), 'utf8'), '# concise\n');
  assert.equal(await readFile(path.join(skills, 'swe', 'SKILL.md'), 'utf8'), '# swe\n');
  assert.equal(await readFile(path.join(skills, 'swe', 'scripts', 'check.py'), 'utf8'), 'print(1)\n');
  await assert.rejects(readFile(path.join(skills, 'swe', 'scripts', 'tests', 'test_check.py'), 'utf8'));
  await assert.rejects(lstat(path.join(skills, 'notes')));
  assert.deepEqual(JSON.parse(await readFile(path.join(skills, '.skills-release.json'), 'utf8')), {
    name: '@awayercode/skills',
    version: '0.1.0',
  });
});

test('install overwrites an existing skill and keeps unrelated skills', async () => {
  const packageRoot = await fixturePackage();
  const projectRoot = await mkdtemp(path.join(tmpdir(), 'skills-project-'));
  const skills = path.join(projectRoot, '.agents', 'skills');
  await mkdir(path.join(skills, 'concise-response'), { recursive: true });
  await writeFile(path.join(skills, 'concise-response', 'SKILL.md'), 'old\n');
  await writeFile(path.join(skills, 'concise-response', 'old.txt'), 'remove me\n');
  await mkdir(path.join(skills, 'local-only'), { recursive: true });
  await writeFile(path.join(skills, 'local-only', 'keep.md'), 'stay\n');

  await installSkills({
    packageRoot,
    projectRoot,
    packageJson: { name: '@awayercode/skills', version: '0.1.0' },
  });

  assert.equal(await readFile(path.join(skills, 'concise-response', 'SKILL.md'), 'utf8'), '# concise\n');
  await assert.rejects(readFile(path.join(skills, 'concise-response', 'old.txt'), 'utf8'));
  assert.equal(await readFile(path.join(skills, 'local-only', 'keep.md'), 'utf8'), 'stay\n');
});

test('install replaces a symlink without deleting its target', async () => {
  const packageRoot = await fixturePackage();
  const projectRoot = await mkdtemp(path.join(tmpdir(), 'skills-project-'));
  const skills = path.join(projectRoot, '.agents', 'skills');
  const target = path.join(projectRoot, 'real-source');
  await mkdir(path.join(target, 'concise-response'), { recursive: true });
  await writeFile(path.join(target, 'concise-response', 'SKILL.md'), 'source\n');
  await mkdir(skills, { recursive: true });
  await symlink(path.join(target, 'concise-response'), path.join(skills, 'concise-response'));

  await installSkills({
    packageRoot,
    projectRoot,
    packageJson: { name: '@awayercode/skills', version: '0.1.0' },
  });

  const installed = await lstat(path.join(skills, 'concise-response'));
  assert.equal(installed.isSymbolicLink(), false);
  assert.equal(await readFile(path.join(skills, 'concise-response', 'SKILL.md'), 'utf8'), '# concise\n');
  assert.equal(await readFile(path.join(target, 'concise-response', 'SKILL.md'), 'utf8'), 'source\n');
});

test('install ignores a Skills directory in the project', async () => {
  const packageRoot = await fixturePackage();
  const projectRoot = await mkdtemp(path.join(tmpdir(), 'skills-project-'));
  const sourceSkill = path.join(projectRoot, 'Skills', 'concise-response');
  await mkdir(sourceSkill, { recursive: true });
  await writeFile(path.join(sourceSkill, 'SKILL.md'), 'workspace source\n');

  await installSkills({
    packageRoot,
    projectRoot,
    packageJson: { name: '@awayercode/skills', version: '0.1.0' },
  });

  assert.equal(await readFile(path.join(sourceSkill, 'SKILL.md'), 'utf8'), 'workspace source\n');
  assert.equal(
    await readFile(path.join(projectRoot, '.agents', 'skills', 'concise-response', 'SKILL.md'), 'utf8'),
    '# concise\n',
  );
});
