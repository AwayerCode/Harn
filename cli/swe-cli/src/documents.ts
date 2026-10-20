import fs from 'node:fs';
import path from 'node:path';

export type Schema = { title: string; sections: { heading: string; columns?: string[]; enums?: Record<string, string[]> }[]; profiles?: Record<string, string[]>; record_fields?: string[] };
export type Reader = (file: string) => string;
export function visible(text: string, errors: string[]): string[] {
  let fence = ''; const lines: string[] = [];
  for (const [i, line] of text.replace(/\r\n/g, '\n').split('\n').entries()) {
    const m = /^\s*(`{3,}|~{3,})(.*)$/.exec(line);
    if (m) {
      if (!fence) { fence = m[1]; if (!m[2].trim()) errors.push(`${i + 1}: 代码块必须声明语言`); }
      else if (m[1][0] === fence[0] && m[1].length >= fence.length && !m[2].trim()) fence = '';
      lines.push('');
    } else lines.push(fence ? '' : line);
  }
  if (fence) errors.push('代码块未关闭');
  return lines;
}
export function tables(lines: string[]): { head: string[]; rows: string[][] }[] {
  const cells = (s: string) => s.trim().replace(/^\||\|$/g, '').split(/(?<!\\)\|/).map(v => v.trim().replace(/\\\|/g, '|'));
  const result = [];
  for (let i = 0; i + 1 < lines.length;) {
    if (lines[i].trim().startsWith('|') && /^\s*\|(?:\s*:?-{3,}:?\s*\|)+\s*$/.test(lines[i + 1])) {
      const head = cells(lines[i]); const rows: string[][] = []; i += 2;
      while (i < lines.length && lines[i].trim().startsWith('|')) rows.push(cells(lines[i++]));
      result.push({ head, rows });
    } else i++;
  }
  return result;
}
const same = (a: unknown, b: unknown) => JSON.stringify(a) === JSON.stringify(b);
function table(lines: string[], columns: string[], errors: string[], label: string, enums: Record<string, string[]> = {}): string[][] {
  const candidates = tables(lines).filter(t => same(t.head, columns));
  if (candidates.length !== 1) { errors.push(`${label}: 必须有且仅有一个字段为 ${columns.join('、')} 的表格`); return []; }
  const rows = candidates[0].rows;
  if (!rows.length) errors.push(`${label}: 表格必须包含数据行`);
  for (const row of rows) {
    if (row.length !== columns.length || row.some(v => !v)) { errors.push(`${label}: 表格列数错误或存在空字段`); continue; }
    for (const [field, allowed] of Object.entries(enums)) if (!allowed.includes(row[columns.indexOf(field)])) errors.push(`${label}: ${field} 必须是 ${allowed.join('、')}`);
  }
  return rows;
}
export function validate(file: string, kind: string, schema: Schema, read: Reader, profile?: string): string[] {
  const errors: string[] = []; let text = read(file).replace(/\r\n/g, '\n'); let spec = schema;
  if (profile) {
    const owned = schema.profiles?.[profile];
    if (!owned) throw new Error(`文档类型 ${kind} 不支持局部校验 ${profile}`);
    spec = { ...schema, sections: schema.sections.filter(s => owned.includes(s.heading)) };
    const raw = text.split('\n'); let keep = true;
    const clean = visible(text, errors);
    text = raw.filter((_, i) => { if (clean[i].startsWith('## ')) keep = owned.includes(clean[i].slice(3).trim()); return keep; }).join('\n').trimEnd() + '\n';
    if (!raw.join('\n').endsWith('\n') || raw.join('\n').endsWith('\n\n')) errors.push('文件末尾必须恰好一个换行');
  }
  if (!text.endsWith('\n') || text.endsWith('\n\n')) errors.push('文件末尾必须恰好一个换行');
  if (/\{\{[^\n{}]+\}\}|\{填写[^\n{}]*\}/.test(text)) errors.push('存在未替换的模板占位符');
  const lines = visible(text, errors); const headings: { i: number; level: number; title: string }[] = []; let previous = 0;
  for (const [i, line] of lines.entries()) {
    const m = /^(#{1,6}) (.+?)\s*$/.exec(line); if (!m) continue;
    const level = m[1].length; headings.push({ i, level, title: m[2] });
    if (level > previous + 1) errors.push(`${i + 1}: 标题层级跳跃`); previous = level;
    if (i && lines[i - 1].trim()) errors.push(`${i + 1}: 标题前缺少空行`);
    if (i + 1 < lines.length && lines[i + 1].trim()) errors.push(`${i + 1}: 标题后缺少空行`);
  }
  if (!same(headings.filter(h => h.level === 1).map(h => h.title), [spec.title])) errors.push(`必须只有一个一级标题：${spec.title}`);
  const h2 = headings.filter(h => h.level === 2);
  if (!same(h2.map(h => h.title), spec.sections.map(s => s.heading))) errors.push('二级章节必须完整且按契约顺序');
  const chunks = Object.fromEntries(h2.map((h, i) => [h.title, lines.slice(h.i + 1, h2[i + 1]?.i ?? lines.length)]));
  for (const section of spec.sections) {
    const body = chunks[section.heading] ?? []; const content = body.filter(l => !l.startsWith('#')).join('\n').trim();
    if (!content) errors.push(`${section.heading}: 内容不能为空`);
    const na = content.startsWith('不适用：') && content.slice(4).trim();
    if (section.columns && !(na && section.heading !== '文档信息')) table(body, section.columns, errors, section.heading, section.enums);
  }
  const metadata = tables(chunks['文档信息'] ?? []).filter(t => same(t.head, ['字段', '值']));
  if (metadata.length === 1) {
    const rows = metadata[0].rows; const values = Object.fromEntries(rows.filter(r => r.length === 2));
    if (!same(rows.map(r => r[0]), ['类型', '任务', '范围', '依据', '状态'])) errors.push('文档信息字段缺失、重复或顺序错误');
    if (values['类型'] !== kind) errors.push('文档信息类型与所选契约不一致');
    if (!['草案', '待验收', '已验收'].includes(values['状态'])) errors.push('状态必须为草案、待验收或已验收');
  }
  if (spec.record_fields) {
    const body = chunks['协议定义'] ?? []; const records = body.flatMap((l, i) => l.startsWith('### ') ? [i] : []);
    if (!records.length && !body.join('\n').trim().startsWith('不适用：')) errors.push('协议定义: 必须有能力三级标题及字段表');
    for (const [index, start] of records.entries()) {
      const rows = table(body.slice(start + 1, records[index + 1] ?? body.length), ['字段', '值'], errors, body[start]);
      if (!same(rows.map(r => r[0]), spec.record_fields)) errors.push(`${body[start]}: 协议字段缺失、重复或顺序错误`);
    }
  }
  for (const match of lines.join('\n').matchAll(/!?\[[^\]\n]*\]\((<[^>]+>|[^\s)]+)\)/g)) {
    const target = match[1].replace(/^<|>$/g, '');
    if (/^[a-z][a-z0-9+.-]*:|^\/\//i.test(target)) continue;
    const [beforeHash, fragment] = target.split('#'); const pathname = beforeHash.split('?')[0];
    const local = pathname ? path.resolve(path.dirname(file), decodeURIComponent(pathname)) : file;
    if (!fs.existsSync(local)) { errors.push(`本地引用不存在：${target}`); continue; }
    // Reading linked Markdown also binds the result to its observed contents.
    if (fragment && path.extname(local) === '.md') {
      const anchors = visible(read(local), []).flatMap(l => {
        const m = /^#{1,6} (.+)$/.exec(l);
        return m ? [m[1].toLowerCase().replace(/[^\p{L}\p{N}_\s-]/gu, '').replace(/ /g, '-')] : [];
      });
      if (!anchors.includes(decodeURIComponent(fragment))) errors.push(`章节引用不存在：${target}`);
    }
  }
  return errors;
}
