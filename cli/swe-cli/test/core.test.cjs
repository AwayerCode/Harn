const { test } = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const crypto = require('node:crypto');
const { spawnSync } = require('node:child_process');
const cli = path.resolve(__dirname, '../dist/swe-cli.cjs');
const repo = path.resolve(__dirname, '../../..');
const fixture = fs.readFileSync(path.join(repo, 'skills/swe/scripts/tests/fixtures/business-model.md'), 'utf8');
function setup(t) { const root = fs.realpathSync(fs.mkdtempSync(path.join(os.tmpdir(), 'swe 空格-'))); t.after(() => fs.rmSync(root, { recursive: true, force: true })); fs.writeFileSync(path.join(root, 'model.md'), fixture); return root; }
function request(root) { return { protocolVersion: 1, taskId: 'test', round: 1, mode: 'design', projectRoot: root, checks: [{ checkId: 'documents', required: true, input: { kind: 'business-model', file: 'model.md' } }] }; }
function run(req) { const r = spawnSync(process.execPath, [cli, 'check', '--request', '-'], { input: JSON.stringify(req), encoding: 'utf8', timeout: 10000 }); assert.ifError(r.error); return { code: r.status, out: JSON.parse(r.stdout) }; }
function extension(root, source, overrides = {}) { fs.writeFileSync(path.join(root, 'extra.ts'), source); return { id: 'extra', path: 'extra.ts', sha256: crypto.createHash('sha256').update(source).digest('hex'), interfaceVersion: 1, required: true, timeoutMs: 3000, input: {}, readOnly: true, ...overrides }; }
const passing = 'module.exports = async function (context: { taskId: string }) { return { interfaceVersion: 1, status: "passed", diagnostics: [], evidence: [context.taskId] }; };';
test('valid document, JSON identity and read-only execution', t => { const root = setup(t); const before = fs.readdirSync(root); const r = run(request(root)); assert.equal(r.code, 0); assert.equal(r.out.checks[0].status, 'passed'); assert.match(r.out.identity.inputDigest, /^[a-f0-9]{64}$/); assert.deepEqual(fs.readdirSync(root), before); });
test('malformed format fails; passing extension cannot override core', t => { const root = setup(t); fs.writeFileSync(path.join(root, 'model.md'), '# broken\n'); const req = request(root); req.extensions = [extension(root, passing)]; const r = run(req); assert.equal(r.code, 1); assert.equal(r.out.checks[0].status, 'failed'); assert.equal(r.out.checks[1].status, 'passed'); });
test('extension cannot inject a core result via output fields', t => { const root = setup(t); const req = request(root); req.extensions = [extension(root, passing.replace('evidence:', 'source: "core", evidence:'))]; assert.equal(run(req).code, 2); });
test('unknown check and missing file fail closed', t => { const root = setup(t); let req = request(root); req.checks[0].checkId = 'scope'; assert.equal(run(req).code, 2); req = request(root); req.checks[0].input.file = 'absent.md'; assert.equal(run(req).code, 2); });
test('empty, duplicate, invalid version and invalid required plans rejected', t => { const root = setup(t); for (const change of [r => r.checks = [], r => r.checks.push(r.checks[0]), r => r.protocolVersion = 2, r => r.checks[0].required = false]) { const r = request(root); change(r); assert.equal(run(r).code, 2); } });
test('extension errors, timeout and digest mismatch fail closed', t => { const root = setup(t); for (const [source, opts] of [['throw new Error("broken");', {}], ['while (true) {}', { timeoutMs: 100 }], [passing, { sha256: '0'.repeat(64) }], ['module.exports = () => ({status:"passed"});', {}]]) { const req = request(root); req.extensions = [extension(root, source, opts)]; assert.equal(run(req).code, 2); } });
test('unavailable required extension wins over a core failure; optional failure is explicit', t => { const root = setup(t); fs.writeFileSync(path.join(root, 'model.md'), '# broken\n'); const req = request(root); req.extensions = [extension(root, 'throw new Error("broken");')]; assert.equal(run(req).code, 2); req.extensions[0].required = false; const r = run(req); assert.equal(r.code, 1); assert.deepEqual(r.out.summary.warnings, ['extension:extra']); });
test('extension change to observed input invalidates passing core', t => { const root = setup(t); const req = request(root); req.extensions = [extension(root, 'module.exports = ctx => { require("node:fs").writeFileSync(require("node:path").join(ctx.projectRoot,"model.md"), "changed"); return {interfaceVersion:1,status:"passed",diagnostics:[],evidence:[]}; };', { readOnly: false })]; const r = run(req); assert.equal(r.code, 2); assert.equal(r.out.checks.at(-1).checkId, 'input-stability'); });
test('review rejects writable extension and unknown imports', t => { const root = setup(t); let req = request(root); req.mode = 'review'; req.extensions = [extension(root, passing, { readOnly: false })]; assert.equal(run(req).code, 2); req = request(root); req.extensions = [extension(root, 'require("node:child_process");' + passing)]; assert.equal(run(req).code, 2); });
test('symlink extension outside project is rejected', t => { if (process.platform === 'win32') return t.skip('requires Windows symlink privilege'); const root = setup(t); const req = request(root); req.extensions = [extension(root, passing)]; fs.unlinkSync(path.join(root, 'extra.ts')); fs.symlinkSync(cli, path.join(root, 'extra.ts')); assert.equal(run(req).code, 2); });
test('format parity with Python on valid and invalid representative artifacts', t => {
  const root = setup(t);
  const variants = [fixture, fixture.trimEnd(), fixture + '\n', fixture.replace('# 业务模型', '# 错误标题'), fixture.replace('## 业务行为', '## 未知章节'), fixture.replace('参与购物的人', '{{填写}}'), fixture + '\n```\nx\n```\n', fixture + '\n[缺失](missing.md)\n'];
  for (const text of variants) {
    fs.writeFileSync(path.join(root, 'model.md'), text);
    const py = spawnSync('python3', ['-B', path.join(repo, 'skills/swe/scripts/validate_documents.py'), '--kind', 'business-model', '--file', path.join(root, 'model.md')], { encoding: 'utf8' });
    assert.ifError(py.error); assert.equal(run(request(root)).code, py.status, py.stdout + py.stderr);
  }
});
test('all existing artifact fixtures match Python format verdicts', () => {
  for (const kind of ['business-model', 'system-design', 'interface-protocol', 'interaction-standards', 'test-design']) {
    const file = `skills/swe/scripts/tests/fixtures/${kind}.md`;
    const req = request(repo); req.checks[0].input = { kind, file };
    const py = spawnSync('python3', ['-B', path.join(repo, 'skills/swe/scripts/validate_documents.py'), '--kind', kind, '--file', path.join(repo, file)], { encoding: 'utf8' });
    assert.ifError(py.error); assert.equal(run(req).code, py.status, kind + ': ' + py.stdout + py.stderr);
  }
});
test('cancelling a running extension prevents subsequent extensions from starting', async t => {
  if (process.platform === 'win32') return t.skip('POSIX signal test');
  const { spawn } = require('node:child_process');
  const root = setup(t); const req = request(root);
  req.extensions = [extension(root, 'module.exports = ctx => { require("node:fs").writeFileSync(require("node:path").join(ctx.projectRoot,"started"), "yes"); while(true) {} };', { readOnly:false })];
  const second = 'module.exports = ctx => { require("node:fs").writeFileSync(require("node:path").join(ctx.projectRoot,"should-not-start"), "yes"); return {interfaceVersion:1,status:"passed",diagnostics:[],evidence:[]}; };';
  fs.writeFileSync(path.join(root, 'second.ts'), second);
  req.extensions.push({...req.extensions[0], id:'second',path:'second.ts',sha256:crypto.createHash('sha256').update(second).digest('hex')});
  const child=spawn(process.execPath,[cli,'check','--request','-']); t.after(()=>child.kill('SIGKILL')); let stdout=''; child.stdout.on('data', b=>stdout+=b); child.stderr.resume();
  const done=new Promise(resolve=>child.on('close',resolve)); child.stdin.end(JSON.stringify(req));
  const deadline=Date.now()+3000;
  while(!fs.existsSync(path.join(root,'started')) && Date.now()<deadline) await new Promise(r=>setTimeout(r,20));
  assert.ok(fs.existsSync(path.join(root,'started'))); child.kill('SIGTERM');
  assert.equal(await done,2); assert.equal(JSON.parse(stdout).checks.find(c=>c.checkId==='extension:second').status,'not-run'); assert.equal(fs.existsSync(path.join(root,'should-not-start')),false);
});
