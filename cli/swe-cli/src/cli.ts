import fs from 'node:fs';
import path from 'node:path';
import crypto from 'node:crypto';
import { spawn } from 'node:child_process';
import { stripTypeScriptTypes } from 'node:module';
import { compileFunction } from 'node:vm';
import { isSea } from 'node:sea';
import { validate, type Schema } from './documents';

declare const SCHEMAS: Record<string, Schema>;
declare const CORE_VERSION: string;
const hash = (s: string | Buffer) => crypto.createHash('sha256').update(s).digest('hex');
const PROTOCOL = 1;
const MAX_BYTES = 1024 * 1024;
type Status = 'passed' | 'failed' | 'unavailable' | 'not-run';
type Diagnostic = { ruleId: string; message: string; location?: string };
type Check = { checkId: string; source: 'core' | 'extension'; required: boolean; status: Status; diagnostics: Diagnostic[]; coverage: string; evidence: unknown[] };
type Extension = { id: string; path: string; sha256: string; interfaceVersion: number; required: boolean; timeoutMs: number; input: unknown; readOnly: boolean };
function object(value: unknown): asserts value is Record<string, any> { if (!value || typeof value !== 'object' || Array.isArray(value)) throw new Error('Expected JSON object'); }
function string(value: unknown): asserts value is string { if (typeof value !== 'string' || !value.trim()) throw new Error('Expected non-empty string'); }
function keys(value: Record<string, unknown>, allowed: string[]) { for (const k of Object.keys(value)) if (!allowed.includes(k)) throw new Error(`Unknown field: ${k}`); }
function integer(value: unknown, min: number, max: number): asserts value is number { if (!Number.isSafeInteger(value) || (value as number) < min || (value as number) > max) throw new Error(`Expected integer ${min}..${max}`); }
function textError(error: unknown): string { return error instanceof Error ? error.message : String(error); }
function output(value: unknown, code = 0) { process.stdout.write(JSON.stringify(value) + '\n'); process.exitCode = code; }
async function stdin(): Promise<string> {
  const parts: Buffer[] = []; let size = 0;
  for await (const data of process.stdin) { const b = Buffer.from(data); size += b.length; if (size > MAX_BYTES) throw new Error('Input exceeds 1 MiB'); parts.push(b); }
  return Buffer.concat(parts).toString('utf8');
}
function inside(root: string, file: string): boolean { const rel = path.relative(root, file); return !path.isAbsolute(rel) && rel !== '..' && !rel.startsWith('..' + path.sep); }
function resolveFile(root: string, name: string): string {
  const file = fs.realpathSync(path.resolve(root, name));
  if (!inside(root, file)) throw new Error(`Input outside projectRoot: ${name}`);
  if (!fs.statSync(file).isFile()) throw new Error(`Not a file: ${name}`);
  return file;
}
function cleanEnvironment() {
  const env = { ...process.env };
  for (const k of ['NODE_OPTIONS', 'NODE_PATH', 'NODE_COMPILE_CACHE', 'NODE_V8_COVERAGE']) delete env[k];
  env.NODE_NO_WARNINGS = '1';
  return env;
}
async function childExtension(payload: unknown, timeoutMs: number): Promise<any> {
  return new Promise((resolve, reject) => {
    const args = isSea() ? ['_extension'] : [process.argv[1], '_extension'];
    const child = spawn(process.execPath, args, { env: cleanEnvironment(), stdio: ['pipe', 'pipe', 'pipe'] });
    const chunks: Buffer[] = []; let size = 0; let failure: Error | undefined;
    const stop = (message: string) => { failure ??= new Error(message); child.kill('SIGKILL'); };
    const cancel = () => stop('Extension cancelled');
    process.once('SIGINT', cancel); process.once('SIGTERM', cancel);
    const timer = setTimeout(() => stop('Extension timed out'), timeoutMs);
    child.stdout.on('data', (b: Buffer) => { size += b.length; if (size > MAX_BYTES) stop('Extension output exceeds 1 MiB'); else chunks.push(b); });
    child.stderr.on('data', (b: Buffer) => { size += b.length; if (size > MAX_BYTES) stop('Extension logs exceed 1 MiB'); });
    child.stdin.on('error', () => {});
    const cleanup = () => { clearTimeout(timer); process.removeListener('SIGINT', cancel); process.removeListener('SIGTERM', cancel); };
    child.once('error', error => { cleanup(); reject(error); });
    child.once('close', code => {
      cleanup();
      if (failure || code !== 0) {
        let message = `Extension exited ${code}`;
        try { const report = JSON.parse(Buffer.concat(chunks).toString('utf8')); if (typeof report.error?.message === 'string') message += ': ' + report.error.message; } catch {}
        reject(failure ?? new Error(message)); return;
      }
      try { resolve(JSON.parse(Buffer.concat(chunks).toString('utf8'))); } catch { reject(new Error('Invalid extension JSON')); }
    });
    child.stdin.end(JSON.stringify(payload));
  });
}
async function extensionWorker() {
  const payload = JSON.parse(await stdin()); object(payload); string(payload.file); string(payload.sha256);
  const source = fs.readFileSync(payload.file, 'utf8'); if (hash(source) !== payload.sha256) throw new Error('Extension digest mismatch');
  const js = stripTypeScriptTypes(source, { mode: 'strip' });
  const extensionModule: { exports: any } = { exports: {} };
  const allowed = new Set(['node:path', 'node:url', 'node:crypto', 'node:fs', 'node:fs/promises']);
  const requireBuiltin = (name: string) => { if (!allowed.has(name)) throw new Error(`Unsupported extension module: ${name}`); return process.getBuiltinModule(name); };
  // Trusted code in a separate process, not a security sandbox. No temp files or caches.
  compileFunction(js, ['module', 'exports', 'require'], { filename: payload.file })(extensionModule, extensionModule.exports, requireBuiltin);
  if (typeof extensionModule.exports !== 'function') throw new Error('Extension must assign a function to module.exports');
  output(await extensionModule.exports(payload.context));
}
function extensionResult(value: unknown): { status: Status; diagnostics: Diagnostic[]; evidence: unknown[] } {
  object(value); keys(value, ['interfaceVersion', 'status', 'diagnostics', 'evidence']);
  if (value.interfaceVersion !== PROTOCOL || !['passed', 'failed', 'unavailable'].includes(value.status)) throw new Error('Invalid extension status/version');
  if (!Array.isArray(value.diagnostics) || !Array.isArray(value.evidence)) throw new Error('Missing extension diagnostics/evidence');
  for (const d of value.diagnostics) { object(d); keys(d, ['ruleId', 'message', 'location']); string(d.ruleId); string(d.message); if (d.location !== undefined) string(d.location); }
  if (value.status !== 'passed' && !value.diagnostics.length) throw new Error('Non-passing extension needs diagnostics');
  return { status: value.status, diagnostics: value.diagnostics, evidence: value.evidence };
}
async function check(request: unknown) {
  object(request); keys(request, ['protocolVersion', 'taskId', 'round', 'mode', 'projectRoot', 'checks', 'extensions']);
  if (request.protocolVersion !== PROTOCOL) throw new Error('Unsupported protocolVersion');
  string(request.taskId); integer(request.round, 1, Number.MAX_SAFE_INTEGER); string(request.projectRoot);
  if (!path.isAbsolute(request.projectRoot)) throw new Error('projectRoot must be absolute');
  const root = fs.realpathSync(request.projectRoot); if (!fs.statSync(root).isDirectory()) throw new Error('projectRoot is not a directory');
  if (!['design', 'implement', 'test', 'review'].includes(request.mode)) throw new Error('Invalid mode');
  if (!Array.isArray(request.checks) || !request.checks.length) throw new Error('checks must not be empty');
  if (request.extensions !== undefined && !Array.isArray(request.extensions)) throw new Error('extensions must be an array');
  const ids = new Set<string>();
  for (const c of request.checks) {
    object(c); keys(c, ['checkId', 'required', 'input']); string(c.checkId);
    if (typeof c.required !== 'boolean' || ids.has(c.checkId)) throw new Error('Invalid required flag or duplicate checkId');
    ids.add(c.checkId); object(c.input);
  }
  for (const e of request.extensions ?? []) {
    object(e); keys(e, ['id', 'path', 'sha256', 'interfaceVersion', 'required', 'timeoutMs', 'input', 'readOnly']); string(e.id); string(e.path);
    if (!/^[a-zA-Z0-9_.-]+$/.test(e.id) || ids.has('extension:' + e.id)) throw new Error('Invalid or duplicate extension id');
    ids.add('extension:' + e.id); if (typeof e.sha256 !== 'string' || !/^[a-f0-9]{64}$/.test(e.sha256)) throw new Error('Invalid extension digest');
    integer(e.timeoutMs, 1, 2147483647);
    if (typeof e.required !== 'boolean' || typeof e.readOnly !== 'boolean') throw new Error('Invalid extension flags');
  }
  if (![...request.checks, ...(request.extensions ?? [])].some(c => c.required)) throw new Error('At least one required check is needed');
  const observed = new Map<string, string>();
  const read = (file: string) => {
    const real = resolveFile(root, file); const bytes = fs.readFileSync(real); const digest = hash(bytes);
    if (observed.has(real) && observed.get(real) !== digest) throw new Error(`Input changed during check: ${real}`);
    observed.set(real, digest); return bytes.toString('utf8');
  };
  const checks: Check[] = [];
  for (const c of request.checks) {
    const result: Check = { checkId: c.checkId, source: 'core', required: c.required, status: 'unavailable', diagnostics: [], coverage: 'Only requested checks; not task completion', evidence: [] };
    try {
      if (c.checkId !== 'documents') throw new Error(`Check not implemented: ${c.checkId}`);
      keys(c.input, ['kind', 'file', 'profile']); string(c.input.kind); string(c.input.file);
      if (c.input.profile !== undefined && c.input.profile !== 'architecture') throw new Error('Unsupported profile');
      if (!Object.hasOwn(SCHEMAS, c.input.kind)) throw new Error('Unknown document kind');
      const file = resolveFile(root, c.input.file);
      const errors = validate(file, c.input.kind, SCHEMAS[c.input.kind], read, c.input.profile);
      result.status = errors.length ? 'failed' : 'passed';
      result.diagnostics = errors.map(message => ({ ruleId: 'documents.format', message, location: file }));
      result.coverage = c.input.profile ? `Partial:${c.input.profile}; not full design validation` : 'Document format and local links; not semantic correctness';
      result.evidence = [file];
    } catch (error) { result.diagnostics.push({ ruleId: 'check.unavailable', message: textError(error) }); }
    checks.push(result);
  }
  let cancelled = false;
  const markCancelled = () => { cancelled = true; };
  process.on('SIGINT', markCancelled); process.on('SIGTERM', markCancelled);
  for (const e of (request.extensions ?? []) as Extension[]) {
    const result: Check = { checkId: 'extension:' + e.id, source: 'extension', required: e.required, status: 'unavailable', diagnostics: [], coverage: 'Explicit trusted extension; not sandboxed', evidence: [] };
    if (cancelled) {
      result.status = 'not-run'; result.diagnostics.push({ ruleId: 'extension.cancelled', message: 'Request cancelled; remaining extensions not started' }); checks.push(result); continue;
    }
    try {
      if (e.interfaceVersion !== PROTOCOL) throw new Error('Unsupported extension interface');
      if (request.mode === 'review' && !e.readOnly) throw new Error('Review requires an explicitly reviewed read-only extension');
      const file = resolveFile(root, e.path); const source = read(file); if (hash(source) !== e.sha256) throw new Error('Extension digest mismatch');
      Object.assign(result, extensionResult(await childExtension({ file, sha256: e.sha256, context: { projectRoot: root, taskId: request.taskId, round: request.round, input: e.input } }, e.timeoutMs)));
    } catch (error) { result.diagnostics.push({ ruleId: 'extension.unavailable', message: textError(error) }); }
    checks.push(result);
  }
  process.removeListener('SIGINT', markCancelled); process.removeListener('SIGTERM', markCancelled);
  if (cancelled) checks.push({ checkId: 'request-cancelled', source: 'core', required: true, status: 'unavailable', diagnostics: [{ ruleId: 'request.cancelled', message: 'Request was cancelled' }], coverage: 'Request lifecycle', evidence: [] });
  const changed: string[] = [];
  for (const [file, digest] of observed) { try { if (hash(fs.readFileSync(file)) !== digest) changed.push(file); } catch { changed.push(file); } }
  if (changed.length) checks.push({ checkId: 'input-stability', source: 'core', required: true, status: 'unavailable', diagnostics: changed.map(location => ({ ruleId: 'input.changed', message: 'Observed input changed during execution', location })), coverage: 'Observed file contents', evidence: [] });
  const required = checks.filter(c => c.required);
  const code = required.some(c => ['unavailable', 'not-run'].includes(c.status)) ? 2 : required.some(c => c.status === 'failed') ? 1 : 0;
  output({ protocolVersion: PROTOCOL, taskId: request.taskId, round: request.round,
    identity: { runId: crypto.randomUUID(), coreVersion: CORE_VERSION, rulesDigest: hash(JSON.stringify(SCHEMAS)), inputDigest: hash(JSON.stringify({ request, observed: [...observed].sort() })), extensionInterfaceVersion: PROTOCOL, files: Object.fromEntries(observed) },
    checks, summary: { status: code === 0 ? 'passed' : code === 1 ? 'failed' : 'unavailable', exitCode: code, warnings: checks.filter(c => !c.required && c.status !== 'passed').map(c => c.checkId) } }, code);
}
async function main() {
  const args = process.argv.slice(2);
  if (args.length === 1 && args[0] === '_extension') { await extensionWorker(); return; }
  if (args.length === 1 && args[0] === 'version') { output({ version: CORE_VERSION, protocolVersion: PROTOCOL, runtime: process.version, standalone: isSea() }); return; }
  if (!args.length || (args.length === 1 && ['help', '--help'].includes(args[0]))) {
    process.stdout.write('swe-cli check --request <file|-> [--format json]\nswe-cli version\nRuntime prototype: documents + trusted TS extensions. Other checks return unavailable.\n'); return;
  }
  if (args[0] !== 'check') throw new Error('Unknown command');
  const options: Record<string, string> = {};
  for (let i = 1; i < args.length; i += 2) { if (!['--request', '--format'].includes(args[i]) || options[args[i]] || args[i + 1] === undefined) throw new Error('Invalid command options'); options[args[i]] = args[i + 1]; }
  if (!options['--request'] || (options['--format'] && options['--format'] !== 'json')) throw new Error('Use --request <file|-> --format json');
  let data: string;
  if (options['--request'] === '-') data = await stdin();
  else { if (fs.statSync(options['--request']).size > MAX_BYTES) throw new Error('Request exceeds 1 MiB'); data = fs.readFileSync(options['--request'], 'utf8'); }
  await check(JSON.parse(data));
}
export async function run() {
  try { await main(); }
  catch (error) { output({ protocolVersion: PROTOCOL, error: { ruleId: 'request.unavailable', message: textError(error) }, summary: { status: 'unavailable', exitCode: 2 } }, 2); }
}
