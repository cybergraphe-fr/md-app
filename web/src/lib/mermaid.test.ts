import { describe, expect, it } from 'vitest';

import { hasMermaidFence, isMermaidLanguage } from './mermaid';

describe('Mermaid helpers', () => {
  it('recognizes Mermaid language identifiers with metadata', () => {
    expect(isMermaidLanguage('mermaid')).toBe(true);
    expect(isMermaidLanguage('Mermaid')).toBe(true);
    expect(isMermaidLanguage('mermaid theme=dark')).toBe(true);
    expect(isMermaidLanguage('typescript')).toBe(false);
  });

  it('detects Mermaid fences with backticks and tildes', () => {
    expect(hasMermaidFence('```mermaid\ngraph TD\nA-->B\n```')).toBe(true);
    expect(hasMermaidFence('~~~ mermaid\ngraph TD\nA-->B\n~~~')).toBe(true);
    expect(hasMermaidFence('```ts\nconsole.log(1)\n```')).toBe(false);
  });
});