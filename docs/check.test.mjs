import assert from 'node:assert/strict';
import { cpSync, mkdirSync, mkdtempSync, readFileSync, rmSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { dirname, resolve } from 'node:path';
import { spawnSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';
import test from 'node:test';

const root = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const checker = resolve(root, 'docs/check.mjs');

function fixture(t) {
  const folder = mkdtempSync(resolve(tmpdir(), 'misthelper-docs-'));
  t.after(() => rmSync(folder, { recursive: true, force: true }));
  for (const path of ['README.md', 'docs', 'CHANGELOG.md', 'LICENSE',
    'SECURITY.md', 'CODE_OF_CONDUCT.md', 'specs/README.md', '.github/copilot-instructions.md']) {
    const target = resolve(folder, path);
    mkdirSync(dirname(target), { recursive: true });
    cpSync(resolve(root, path), target, { recursive: true });
  }
  return folder;
}

function rewrite(folder, path, from, to) {
  const file = resolve(folder, path);
  const text = readFileSync(file, 'utf8');
  assert.ok(text.includes(from), `Fixture must include ${from}`);
  writeFileSync(file, text.replace(from, to));
}

function run(folder) {
  return spawnSync(process.execPath, [checker, folder], { encoding: 'utf8' });
}

test('current documentation satisfies landing and screenshot requirements', (t) => {
  const result = run(fixture(t));
  assert.equal(result.status, 0, result.stderr);
  assert.match(result.stdout, /Documentation checks passed/);
});

test('unexpected landing heading is rejected', (t) => {
  const folder = fixture(t);
  rewrite(folder, 'README.md', '## What', '## Status');
  assert.notEqual(run(folder).status, 0);
});

test('broken documentation heading anchor is rejected', (t) => {
  const folder = fixture(t);
  rewrite(folder, 'README.md', '#current-status', '#missing-status');
  const result = run(folder);
  assert.notEqual(result.status, 0);
  assert.match(result.stderr, /missing heading/);
});

test('missing screenshot reference is rejected', (t) => {
  const folder = fixture(t);
  rewrite(folder, 'README.md', 'screenshots/menu.png', 'screenshots/missing.png');
  const result = run(folder);
  assert.notEqual(result.status, 0);
  assert.match(result.stderr, /missing link target/);
});

test('duplicate screenshots do not count as several user screens', (t) => {
  const folder = fixture(t);
  rewrite(folder, 'README.md', 'screenshots/stub-operation.png', 'screenshots/menu.png');
  assert.notEqual(run(folder).status, 0);
});

test('non-PNG content cannot masquerade as a screenshot', (t) => {
  const folder = fixture(t);
  writeFileSync(resolve(folder, 'docs/screenshots/menu.png'), 'not an actual PNG image');
  assert.notEqual(run(folder).status, 0);
});
