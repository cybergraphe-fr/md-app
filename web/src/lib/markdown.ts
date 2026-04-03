const reListItem = /^\s{0,3}(?:[-*+]\s+|\d+[.)]\s+)/;
const reIndentedHeading = /^[ \t]{4,}(#{1,6}[ \t]+)/;
const reIndentedTightAtxHeading = /^[ \t]{4,}(#{2,6})([^\s#])/;
const reInlineHeading = /[ \t]+(#{1,6}[ \t]+)/g;
const reTightAtxHeading = /^(\s{0,3}#{1,6})([^\s#])/;
const reInlineTightAtxHeading = /([ \t])(#{2,6})([^\s#])/g;
const reTableSep = /^[\s|:\-]+$/;

function isListItem(line: string): boolean {
  return reListItem.test(line.trimEnd());
}

function isSpecialMarkdownLine(line: string): boolean {
  const trimmed = line.trim();
  if (!trimmed) return true;
  return isListItem(line)
    || trimmed.startsWith('#')
    || trimmed.startsWith('>')
    || trimmed.startsWith('|')
    || trimmed.startsWith('```')
    || trimmed.startsWith('~~~');
}

function normalizeInlineTableLine(line: string): string[] | null {
  const trimmed = line.trim();
  if (!trimmed || !trimmed.includes('||') || (trimmed.match(/\|/g) ?? []).length < 4) {
    return null;
  }

  let normalized = line.replaceAll('—', '-').replaceAll('–', '-');
  const outRows: string[] = [];
  let hadPrefix = false;

  const firstPipe = normalized.indexOf('|');
  if (firstPipe > 0) {
    const prefix = normalized.slice(0, firstPipe).trim();
    if (prefix) {
      outRows.push(prefix);
      hadPrefix = true;
    }
    normalized = normalized.slice(firstPipe);
  }

  if (hadPrefix) outRows.push('');

  let tableRows = 0;
  for (const chunk of normalized.split('||')) {
    let row = chunk.trim();
    if (!row) continue;
    if ((row.match(/\|/g) ?? []).length < 2) {
      outRows.push(row);
      continue;
    }
    if (!row.startsWith('|')) row = `| ${row}`;
    if (!row.endsWith('|')) row = `${row} |`;

    const inner = row.replace(/^\|/, '').replace(/\|$/, '').trim();
    if (reTableSep.test(inner)) {
      const cells = row
        .replace(/^\|/, '')
        .replace(/\|$/, '')
        .split('|')
        .map((raw) => {
          const cell = raw.trim();
          const left = cell.startsWith(':');
          const right = cell.endsWith(':');
          let sep = '---';
          if (left) sep = `:${sep}`;
          if (right) sep = `${sep}:`;
          return sep;
        });
      row = `| ${cells.join(' | ')} |`;
    }

    outRows.push(row);
    tableRows++;
  }

  if (tableRows < 2) return null;
  outRows.push('');
  return outRows;
}

export function normalizeMarkdown(content: string): string {
  if (!content) return content;

  const lines = content
    .replace(/\r\n/g, '\n')
    .replace(/\r/g, '\n')
    .replace(/\u00a0/g, ' ')
    .split('\n');

  const out: string[] = [];
  let inFence = false;

  for (let index = 0; index < lines.length; index++) {
    let line = lines[index];
    const trimmed = line.trim();
    let nextSignificant = '';
    for (let j = index + 1; j < lines.length; j++) {
      const candidate = lines[j]?.trim() ?? '';
      if (candidate) {
        nextSignificant = lines[j];
        break;
      }
    }

    if (trimmed.startsWith('```') || trimmed.startsWith('~~~')) {
      inFence = !inFence;
      out.push(line);
      continue;
    }

    if (inFence) {
      out.push(line);
      continue;
    }

    line = line.replace(reIndentedHeading, '$1');
    line = line.replace(reIndentedTightAtxHeading, '$1 $2');
    line = line.replace(reTightAtxHeading, '$1 $2');

    const t = line.trim();
    if (t.startsWith('• ')) {
      line = `- ${t.slice(2)}`;
    } else if (t.startsWith('◦ ')) {
      line = `  - ${t.slice(2)}`;
    } else if (t.startsWith('* ')) {
      const leading = (line.match(/^(\s*)/)?.[1] ?? '').length;
      const prefix = leading <= 1 ? '' : ' '.repeat(leading);
      line = `${prefix}- ${t.slice(2)}`;
    }

    if (!line.trimStart().startsWith('#')) {
      line = line.replace(reInlineTightAtxHeading, '$1$2 $3');
      if (reInlineHeading.test(line)) {
        line = line.replace(reInlineHeading, '\n\n$1');
      }
      reInlineHeading.lastIndex = 0;
    }

    if (!/^\s*[-*+]/.test(line) && /\s+•\s+/.test(line)) {
      line = line.replace(/\s+•\s+/g, '\n- ');
    }

    if (/\s+◦\s+/.test(line)) {
      line = line.replace(/\s+◦\s+/g, '\n  - ');
    }

    if (/\s+\*\s+/.test(line)) {
      const replacement = isListItem(line) ? '\n  - ' : '\n- ';
      line = line.replace(/\s+\*\s+/g, replacement);
    }

    if (line.trim() && isListItem(nextSignificant) && !isSpecialMarkdownLine(line)) {
      if (out.length > 0 && out[out.length - 1]?.trim()) out.push('');
      out.push(line, '');
      continue;
    }

    for (const segment of line.split('\n')) {
      if (isListItem(segment) && out.length > 0) {
        const prevRaw = out[out.length - 1] ?? '';
        const prev = prevRaw.trim();
        if (prev && !isListItem(prevRaw) && !prev.startsWith('|') && !prev.startsWith('#') && !prev.startsWith('>')) {
          out.push('');
        }
      }

      const tableLines = normalizeInlineTableLine(segment);
      if (tableLines) out.push(...tableLines);
      else out.push(segment);
    }
  }

  return out.join('\n');
}

export const PAGE_BREAK_RE = /^(\\(?:newpage|pagebreak)\s*$|<!--\s*pagebreak\s*-->\s*$|---\s*pagebreak\s*---\s*$)/gm;
export const PAGE_BREAK_INDICATOR_HTML = '<div class="pagebreak-indicator" aria-label="Page break"><span>⸻ Saut de page ⸻</span></div>';

export function preprocessPreviewMarkdown(content: string): string {
  return normalizeMarkdown(content).replace(PAGE_BREAK_RE, PAGE_BREAK_INDICATOR_HTML);
}
