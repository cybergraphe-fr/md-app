#!/usr/bin/env node
// ─────────────────────────────────────────────────────────────
// Mermaid SSR — Server-side rendering for PDF export
// ─────────────────────────────────────────────────────────────
// Thin wrapper around @mermaid-js/mermaid-cli (mmdc).
// Reads a .mmd file and outputs SVG to stdout.
//
// Usage: node render.mjs <input.mmd>
// ─────────────────────────────────────────────────────────────

import { execFileSync } from 'node:child_process';
import { readFileSync, unlinkSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';

const inputFile = process.argv[2];
const outputMode = (process.argv[3] ?? 'svg').toLowerCase();

if (!inputFile || (outputMode !== 'svg' && outputMode !== 'png')) {
  process.stderr.write('Usage: render.mjs <input.mmd> [svg|png]\n');
  process.exit(1);
}

const extension = outputMode === 'png' ? 'png' : 'svg';
const outFile = join(tmpdir(), `mermaid-${process.pid}-${Date.now()}.${extension}`);

try {
  const mmdc = join(import.meta.dirname, 'node_modules', '.bin', 'mmdc');
  execFileSync(mmdc, [
    '-i', inputFile,
    '-o', outFile,
    '-c', join(import.meta.dirname, 'mermaid.config.json'),
    '-t', 'default',
    '-b', 'transparent',
    '--puppeteerConfigFile', join(import.meta.dirname, 'puppeteer.json'),
  ], { timeout: 30_000, stdio: ['ignore', 'ignore', 'pipe'] });
  const output = readFileSync(outFile);
  if (outputMode === 'png') {
    process.stdout.write(output.toString('base64'));
  } else {
    process.stdout.write(output.toString('utf-8'));
  }
} catch (err) {
  process.stderr.write(`Mermaid render error: ${err.message}\n`);
  process.exit(2);
} finally {
  try { unlinkSync(outFile); } catch { /* ignore */ }
}
