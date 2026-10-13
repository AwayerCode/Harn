#!/usr/bin/env node
import { access, cp, lstat, mkdir, readdir, readFile, rm, unlink, writeFile } from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';

export function includeSkillFile(source, skillDir) {
  const relative = path.relative(skillDir, source);
  if (!relative) return true;
  const parts = relative.split(path.sep);
  if (parts.includes('__pycache__') || parts.some((part) => part.endsWith('.pyc'))) return false;
  for (let index = 0; index < parts.length - 1; index += 1) {
    if (parts[index] === 'scripts' && parts[index + 1] === 'tests') return false;
  }
  return true;
}

async function replaceDestination(destination) {
  let stat;
  try {
    stat = await lstat(destination);
  } catch (error) {
    if (error.code === 'ENOENT') return;
    throw error;
  }
  if (stat.isSymbolicLink()) {
    await unlink(destination);
    return;
  }
  await rm(destination, { recursive: true, force: true });
}

export async function installSkills({ packageRoot, projectRoot, packageJson }) {
  const destinationRoot = path.join(projectRoot, '.agents', 'skills');
  await mkdir(destinationRoot, { recursive: true });
  const entries = await readdir(packageRoot, { withFileTypes: true });
  for (const entry of entries) {
    if (!entry.isDirectory()) continue;
    const skillDir = path.join(packageRoot, entry.name);
    try {
      await access(path.join(skillDir, 'SKILL.md'));
    } catch (error) {
      if (error.code === 'ENOENT') continue;
      throw error;
    }
    const destination = path.join(destinationRoot, entry.name);
    await replaceDestination(destination);
    await cp(skillDir, destination, {
      recursive: true,
      filter: (source) => includeSkillFile(source, skillDir),
    });
  }
  await writeFile(
    path.join(destinationRoot, '.skills-release.json'),
    `${JSON.stringify({ name: packageJson.name, version: packageJson.version }, null, 2)}\n`,
  );
}

async function main() {
  const packageRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
  const packageJson = JSON.parse(await readFile(path.join(packageRoot, 'package.json'), 'utf8'));
  await installSkills({ packageRoot, projectRoot: process.cwd(), packageJson });
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  main().catch((error) => {
    console.error(error.message);
    process.exit(1);
  });
}
