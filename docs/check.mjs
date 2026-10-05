import assert from 'node:assert/strict';
import { readFileSync, existsSync, readdirSync } from 'node:fs';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const root = resolve(process.argv[2] ?? resolve(dirname(fileURLToPath(import.meta.url)), '..'));
const landing = readFileSync(resolve(root, 'README.md'), 'utf8');
const headings = [...landing.matchAll(/^## (.+)$/gm)].map((match) => match[1]);
assert.deepEqual(headings, ['What', 'How', 'Where', 'When', 'Why', 'Who']);
assert.equal((landing.match(/^# /gm) ?? []).length, 1);
assert.equal(/^#{3,} /m.test(landing), false, 'Landing must have no nested headings');
assert.equal(landing.includes('```'), false, 'Move command blocks into docs/');

function checkLinks(file) {
  const text = readFileSync(file, 'utf8');
  for (const match of text.matchAll(/!?\[[^\]]*\]\(([^)\s]+)\)/g)) {
    const target = match[1];
    if (/^(https?:|mailto:|#)/.test(target)) continue;
    const path = resolve(dirname(file), decodeURIComponent(target.split('#')[0]));
    assert.ok(existsSync(path), `${file}: missing link target ${target}`);
    const anchor = target.split('#')[1];
    if (anchor && path.endsWith('.md')) {
      const body = readFileSync(path, 'utf8');
      const anchors = [...body.matchAll(/^#{1,6} (.+)$/gm)].map((heading) =>
        heading[1].toLowerCase().replace(/[^\w\s-]/g, '').replace(/\s/g, '-'));
      assert.ok(anchors.includes(anchor), `${file}: missing heading ${target}`);
    }
  }
}

checkLinks(resolve(root, 'README.md'));
for (const file of readdirSync(resolve(root, 'docs'))) {
  if (file.endsWith('.md')) checkLinks(resolve(root, 'docs', file));
}

const screenshots = [...landing.matchAll(/!\[[^\]]+\]\((docs\/screenshots\/[^)]+\.png)\)/g)];
assert.ok(screenshots.length >= 3, 'Embed at least three actual user screens');
assert.equal(new Set(screenshots.map((match) => match[1])).size, screenshots.length);
for (const [, path] of screenshots) {
  const image = readFileSync(resolve(root, path));
  assert.equal(image.subarray(0, 8).toString('hex'), '89504e470d0a1a0a', path);
  assert.equal(image.subarray(12, 16).toString(), 'IHDR', path);
  assert.ok(image.readUInt32BE(16) >= 800 && image.readUInt32BE(20) >= 400, path);
}
assert.match(readFileSync(resolve(root, 'docs/interface.md'), 'utf8'), /N\/A, no dashboard exists/);
console.log('Documentation checks passed: headings, links, anchors, three PNG screenshots, dashboard N/A.');
