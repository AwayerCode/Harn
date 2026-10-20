import { build } from 'esbuild';
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const schemaRoot = path.resolve(root, '../../skills/swe/references/artifact-schemas');
const schemas = Object.fromEntries(fs.readdirSync(schemaRoot).filter(n => n.endsWith('.json')).sort().map(n => [n.slice(0, -5), JSON.parse(fs.readFileSync(path.join(schemaRoot, n), 'utf8'))]));
const pkg = JSON.parse(fs.readFileSync(path.join(root, 'package.json')));
await build({ entryPoints: [path.join(root, 'src/main.ts')], bundle: true, platform: 'node', target: 'node22', format: 'cjs', outfile: path.join(root, 'dist/swe-cli.cjs'), define: { SCHEMAS: JSON.stringify(schemas), CORE_VERSION: JSON.stringify(pkg.version) } });
