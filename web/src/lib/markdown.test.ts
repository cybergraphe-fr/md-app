import { describe, expect, it } from 'vitest';
import { Marked } from 'marked';

import { normalizeMarkdown, preprocessPreviewMarkdown } from './markdown';

const marked = new Marked({
  gfm: true,
  breaks: true,
  pedantic: false,
});

function renderPreview(content: string): string {
  const preprocessed = preprocessPreviewMarkdown(content);
  return marked.parse(preprocessed) as string;
}

describe('Markdown normalization for preview', () => {
  it('renders heading levels h1 to h6', () => {
    const content = [
      '# Heading 1',
      '## Heading 2',
      '### Heading 3',
      '#### Heading 4',
      '##### Heading 5',
      '###### Heading 6',
    ].join('\n');

    const html = renderPreview(content);
    for (let level = 1; level <= 6; level++) {
      expect(html).toContain(`<h${level}`);
      expect(html).toContain(`Heading ${level}`);
    }
  });

  it('normalizes inline headings across all levels', () => {
    const lines = Array.from({ length: 6 }, (_, idx) => {
      const level = idx + 1;
      const hashes = '#'.repeat(level);
      return `Intro sentence. ${hashes} Inline L${level}`;
    });

    const normalized = normalizeMarkdown(lines.join('\n'));
    for (let level = 1; level <= 6; level++) {
      const hashes = '#'.repeat(level);
      expect(normalized).toContain(`\n\n${hashes} Inline L${level}`);
    }
  });

  it('normalizes tight ATX headings and renders them', () => {
    const content = '##Heading 2\n###Heading 3\nText ##Heading Inline';
    const normalized = normalizeMarkdown(content);

    expect(normalized).toContain('## Heading 2');
    expect(normalized).toContain('### Heading 3');
    expect(normalized).toContain('\n\n## Heading Inline');

    const html = renderPreview(content);
    expect(html).toContain('<h2');
    expect(html).toContain('Heading 2');
    expect(html).toContain('<h3');
    expect(html).toContain('Heading 3');
  });

  it('normalizes indented tight h2 headings and renders them', () => {
    const content = '    ##Titre section';
    const normalized = normalizeMarkdown(content);

    expect(normalized).toContain('## Titre section');

    const html = renderPreview(content);
    expect(html).toContain('<h2');
    expect(html).toContain('Titre section');
  });

  it('preserves markdown syntax for lists, blockquotes, tables, and fenced code', () => {
    const content = [
      '- Item A',
      '- Item B',
      '',
      '> Quote line',
      '',
      '| Col A | Col B |',
      '| --- | --- |',
      '| 1 | 2 |',
      '',
      '```ts',
      'const answer = 42;',
      '```',
    ].join('\n');

    const html = renderPreview(content);
    expect(html).toContain('<ul>');
    expect(html).toContain('<blockquote>');
    expect(html).toContain('<table>');
    expect(html).toContain('<pre><code');
    expect(html).toContain('const answer = 42;');
  });

  it('keeps fenced content intact while normalizing headings outside fences', () => {
    const content = '```md\n## not-a-heading\n```\nAfter text ## real heading';
    const normalized = normalizeMarkdown(content);

    expect(normalized).toContain('```md\n## not-a-heading\n```');
    expect(normalized).toContain('\n\n## real heading');
  });

  it('keeps h2 headings parsed around mermaid fences', () => {
    const content = [
      '## Section A',
      '',
      '```mermaid',
      'graph TD',
      'A-->B',
      '```',
      '',
      '## Section B',
    ].join('\n');

    const html = renderPreview(content);
    expect((html.match(/<h2/g) ?? []).length).toBeGreaterThanOrEqual(2);
    expect(html).toContain('Section A');
    expect(html).toContain('Section B');
  });

  it('converts manual page break markers to preview indicators', () => {
    const content = 'Before\n\\newpage\n## After';
    const preprocessed = preprocessPreviewMarkdown(content);

    expect(preprocessed).toContain('pagebreak-indicator');
    expect(preprocessed).not.toContain('\\newpage');
  });

  it('keeps h2 parsed after page break marker', () => {
    const content = 'Intro\n\\newpage\n## 🗺️ LA BOUCLE';
    const html = renderPreview(content);

    expect(html).toContain('pagebreak-indicator');
    expect(html).toContain('<h2');
    expect(html).toContain('LA BOUCLE');
    expect(html).not.toContain('## 🗺️ LA BOUCLE');
  });
});
