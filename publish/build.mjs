import { spawnSync } from 'node:child_process';
import { access, cp, mkdtemp, mkdir, readdir, readFile, rm, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import path from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';
import { includeSkillFile } from './bin/install.mjs';

const publishRoot = path.dirname(fileURLToPath(import.meta.url));
const skillsRoot = path.resolve(publishRoot, '..');

export async function stagePackage(destination) {
  const packageJson = JSON.parse(await readFile(path.join(publishRoot, 'package.json'), 'utf8'));
  await mkdir(destination, { recursive: true });
  await cp(path.join(publishRoot, 'bin'), path.join(destination, 'bin'), { recursive: true });
  const entries = await readdir(skillsRoot, { withFileTypes: true });
  for (const entry of entries) {
    if (!entry.isDirectory() || entry.name === 'publish') continue;
    const skillDir = path.join(skillsRoot, entry.name);
    try {
      await access(path.join(skillDir, 'SKILL.md'));
    } catch (error) {
      if (error.code === 'ENOENT') continue;
      throw error;
    }
    await cp(skillDir, path.join(destination, entry.name), {
      recursive: true,
      filter: (source) => includeSkillFile(source, skillDir),
    });
  }
  await writeFile(path.join(destination, 'package.json'), `${JSON.stringify(packageJson, null, 2)}\n`);
  return packageJson;
}

async function main() {
  const packageJson = JSON.parse(await readFile(path.join(publishRoot, 'package.json'), 'utf8'));
  const publish = process.argv.includes('--publish');
  if (publish && !packageJson.license) {
    console.error('license 未设置，不执行 npm publish');
    process.exit(1);
  }
  const stage = await mkdtemp(path.join(tmpdir(), 'awayercode-skills-'));
  try {
    await stagePackage(stage);
    const args = publish ? ['publish', '--access', 'public'] : ['pack', '--dry-run'];
    const result = spawnSync('npm', args, { cwd: stage, stdio: 'inherit' });
    process.exit(result.status ?? 1);
  } finally {
    await rm(stage, { recursive: true, force: true });
  }
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  main().catch((error) => {
    console.error(error.message);
    process.exit(1);
  });
}
