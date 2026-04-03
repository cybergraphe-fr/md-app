<script lang="ts">
  import { onMount } from 'svelte';
  import { Marked } from 'marked';
  import { markedHighlight } from 'marked-highlight';
  import markedFootnote from 'marked-footnote';
  import DOMPurify from 'dompurify';
  import hljs from 'highlight.js/lib/core';
  import javascript from 'highlight.js/lib/languages/javascript';
  import typescript from 'highlight.js/lib/languages/typescript';
  import python from 'highlight.js/lib/languages/python';
  import go from 'highlight.js/lib/languages/go';
  import bash from 'highlight.js/lib/languages/bash';
  import css from 'highlight.js/lib/languages/css';
  import xml from 'highlight.js/lib/languages/xml';
  import json from 'highlight.js/lib/languages/json';
  import yaml from 'highlight.js/lib/languages/yaml';
  import sql from 'highlight.js/lib/languages/sql';
  import dockerfile from 'highlight.js/lib/languages/dockerfile';
  import rust from 'highlight.js/lib/languages/rust';
  import java from 'highlight.js/lib/languages/java';
  import c from 'highlight.js/lib/languages/c';
  import cpp from 'highlight.js/lib/languages/cpp';
  import csharp from 'highlight.js/lib/languages/csharp';
  import ruby from 'highlight.js/lib/languages/ruby';
  import php from 'highlight.js/lib/languages/php';
  import diff from 'highlight.js/lib/languages/diff';
  import ini from 'highlight.js/lib/languages/ini';
  import makefile from 'highlight.js/lib/languages/makefile';
  import mdLang from 'highlight.js/lib/languages/markdown';
  import plaintext from 'highlight.js/lib/languages/plaintext';
  import { activeContent } from '$lib/stores/files';
  import { hasMermaidFence, isMermaidLanguage } from '$lib/mermaid';
  import { preprocessPreviewMarkdown } from '$lib/markdown';

  // Register languages for tree-shaken hljs
  hljs.registerLanguage('javascript', javascript);
  hljs.registerLanguage('js', javascript);
  hljs.registerLanguage('typescript', typescript);
  hljs.registerLanguage('ts', typescript);
  hljs.registerLanguage('python', python);
  hljs.registerLanguage('go', go);
  hljs.registerLanguage('bash', bash);
  hljs.registerLanguage('sh', bash);
  hljs.registerLanguage('shell', bash);
  hljs.registerLanguage('css', css);
  hljs.registerLanguage('html', xml);
  hljs.registerLanguage('xml', xml);
  hljs.registerLanguage('json', json);
  hljs.registerLanguage('yaml', yaml);
  hljs.registerLanguage('yml', yaml);
  hljs.registerLanguage('sql', sql);
  hljs.registerLanguage('dockerfile', dockerfile);
  hljs.registerLanguage('docker', dockerfile);
  hljs.registerLanguage('rust', rust);
  hljs.registerLanguage('java', java);
  hljs.registerLanguage('c', c);
  hljs.registerLanguage('cpp', cpp);
  hljs.registerLanguage('csharp', csharp);
  hljs.registerLanguage('cs', csharp);
  hljs.registerLanguage('ruby', ruby);
  hljs.registerLanguage('php', php);
  hljs.registerLanguage('diff', diff);
  hljs.registerLanguage('ini', ini);
  hljs.registerLanguage('toml', ini);
  hljs.registerLanguage('makefile', makefile);
  hljs.registerLanguage('markdown', mdLang);
  hljs.registerLanguage('md', mdLang);
  hljs.registerLanguage('plaintext', plaintext);
  hljs.registerLanguage('text', plaintext);

  // ── Mermaid (lazy‑loaded) ──
  let mermaidReady = $state(false);
  let mermaidModule = $state<typeof import('mermaid')['default'] | null>(null);

  async function ensureMermaid() {
    if (mermaidModule) return;
    const m = await import('mermaid');
    mermaidModule = m.default;
    mermaidModule.initialize({
      startOnLoad: false,
      theme: 'dark',
      securityLevel: 'strict',
      flowchart: { htmlLabels: false },
    });
    mermaidReady = true;
  }

  // ── KaTeX (lazy‑loaded) ──
  let katexRender: ((tex: string, opts?: object) => string) | null = null;

  async function ensureKaTeX() {
    if (katexRender) return;
    const k = await import('katex');
    katexRender = k.default.renderToString;
  }

  // Configure Marked with extensions
  const marked = new Marked(
    markedHighlight({
      langPrefix: 'hljs language-',
      highlight(code, lang) {
        if (isMermaidLanguage(lang)) return code; // pass-through for mermaid
        const language = hljs.getLanguage(lang) ? lang : 'plaintext';
        return hljs.highlight(code, { language }).value;
      },
    }),
    {
      gfm: true,
      breaks: true,
      pedantic: false,
    }
  );

  marked.use(markedFootnote());
  marked.use({
    renderer: {
      heading(this: any, { depth, tokens }: { depth: number; tokens: any[] }): string {
        const text = this.parser.parseInline(tokens);
        const slug = text.replace(/<[^>]*>/g, '').toLowerCase().replace(/[^\w]+/g, '-');
        return `<h${depth} id="${slug}">${text}</h${depth}>\n`;
      },
      link(this: any, { href, title, tokens }: { href: string; title?: string | null; tokens: any[] }): string {
        const text = this.parser.parseInline(tokens);
        const external = href?.startsWith('http') && !href.startsWith(window.location.origin);
        const attrs = external ? ' target="_blank" rel="noopener noreferrer"' : '';
        const t = title ? ` title="${title}"` : '';
        return `<a href="${href}"${t}${attrs}>${text}</a>`;
      },
      image({ href, title, text }: { href: string; title?: string | null; text: string }): string {
        const t = title ? ` title="${title}"` : '';
        return `<img src="${href}" alt="${text}"${t} loading="lazy">`;
      },
      code({ text, lang, escaped }: { text: string; lang?: string; escaped?: boolean }): string {
        if (isMermaidLanguage(lang)) {
          const safe = text.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
          return `<pre class="mermaid-block" data-mermaid="true">${safe}</pre>`;
        }
        const language = lang && hljs.getLanguage(lang) ? lang : 'plaintext';
        const code = escaped ? text : hljs.highlight(text, { language }).value;
        return `<pre><code class="hljs language-${language}">${code}</code></pre>`;
      },
      table({ header, rows }: { header: any[]; rows: any[][] }): string {
        const hdr = header.map((h: any) => {
          const align = h.align ? ` style="text-align:${h.align}"` : '';
          return `<th${align}>${this.parser.parseInline(h.tokens)}</th>`;
        }).join('');
        const body = rows.map((row: any[]) =>
          '<tr>' + row.map((cell: any) => {
            const align = cell.align ? ` style="text-align:${cell.align}"` : '';
            return `<td${align}>${this.parser.parseInline(cell.tokens)}</td>`;
          }).join('') + '</tr>'
        ).join('\n');
        return `<div class="table-scroll"><table><thead><tr>${hdr}</tr></thead><tbody>${body}</tbody></table></div>`;
      },
    },
  });

  // ── KaTeX inline/block processing ──
  function processKaTeX(html: string): string {
    if (!katexRender) return html;
    // Block math: $$...$$
    html = html.replace(/\$\$([\s\S]+?)\$\$/g, (_match, tex) => {
      try {
        return katexRender!(tex.trim(), { displayMode: true, throwOnError: false });
      } catch { return _match; }
    });
    // Inline math: $...$  (not $$)
    html = html.replace(/(?<!\$)\$(?!\$)(.+?)(?<!\$)\$(?!\$)/g, (_match, tex) => {
      try {
        return katexRender!(tex.trim(), { displayMode: false, throwOnError: false });
      } catch { return _match; }
    });
    return html;
  }

  // ── Reactive state (Svelte 5 runes) ──
  let renderedHtml = $state('');
  let container = $state<HTMLElement | undefined>(undefined);
  let mermaidCounter = 0;

  // Re‑render on content change
  $effect(() => {
    const content = $activeContent;
    // Pre-load KaTeX if content has $ signs
    if (content.includes('$')) void ensureKaTeX();
    // Pre-load Mermaid if content has mermaid code blocks
    if (hasMermaidFence(content)) void ensureMermaid();

    try {
      // Replace page break markers before parsing
      const preprocessed = preprocessPreviewMarkdown(content);
      let html = marked.parse(preprocessed) as string;
      html = processKaTeX(html);
      renderedHtml = DOMPurify.sanitize(html, { ADD_ATTR: ['data-mermaid'] });
    } catch {
      renderedHtml = `<p class="render-error">Render error</p>`;
    }
  });

  // Post-render: copy buttons, emojis, mermaid diagrams
  $effect(() => {
    if (!container || !renderedHtml) return;
    // Dereference reactive state synchronously so Svelte 5 tracks them
    const _mReady = mermaidReady;
    const _mModule = mermaidModule;
    setTimeout(async () => {
      // Copy buttons on code blocks
      container?.querySelectorAll('pre:not([data-copy]):not([data-mermaid])').forEach((pre) => {
        pre.setAttribute('data-copy', '1');
        const btn = document.createElement('button');
        btn.className = 'copy-btn';
        btn.textContent = 'Copy';
        btn.addEventListener('click', () => {
          const code = pre.querySelector('code');
          if (!code) return;
          navigator.clipboard.writeText(code.textContent ?? '').then(() => {
            btn.textContent = 'Copied!';
            btn.classList.add('copied');
            setTimeout(() => {
              btn.textContent = 'Copy';
              btn.classList.remove('copied');
            }, 1800);
          });
        });
        pre.appendChild(btn);
      });

      // Emoji shortcodes
      container?.querySelectorAll('p, li, h1, h2, h3, h4, h5, h6').forEach((el) => {
        if (el.children.length === 0 && el.textContent?.includes(':')) {
          const text = el.textContent;
          const parts: (string | Node)[] = [];
          let lastIndex = 0;
          text.replace(/:([a-z0-9_+-]+):/g, (match, name, offset) => {
            if (offset > lastIndex) parts.push(document.createTextNode(text.slice(lastIndex, offset)));
            const emoji = emojiMap[name];
            if (emoji) {
              const span = document.createElement('span');
              span.className = 'emoji';
              span.title = `:${name}:`;
              span.textContent = emoji;
              parts.push(span);
            } else {
              parts.push(document.createTextNode(match));
            }
            lastIndex = offset + match.length;
            return match;
          });
          if (parts.length > 0) {
            if (lastIndex < text.length) parts.push(document.createTextNode(text.slice(lastIndex)));
            el.textContent = '';
            parts.forEach(p => el.appendChild(p instanceof Node ? p : document.createTextNode(String(p))));
          }
        }
      });

      // Mermaid diagrams
      const blocks = container?.querySelectorAll('pre[data-mermaid]:not([data-rendered])') ?? [];
      if (blocks.length > 0 && !_mReady) {
        void ensureMermaid();
        return;
      }

      if (_mReady && _mModule) {
        for (const block of blocks) {
          block.setAttribute('data-rendered', '1');
          const src = block.textContent ?? '';
          try {
            mermaidCounter++;
            const { svg } = await _mModule.render(`mermaid-${mermaidCounter}`, src);
            const div = document.createElement('div');
            div.className = 'mermaid-diagram';
            // Mermaid strict mode already sanitizes diagram content.
            // DOMPurify's SVG profile strips foreignObject label content.
            div.innerHTML = svg;
            block.replaceWith(div);
          } catch {
            block.classList.add('mermaid-error');
          }
        }
      }
    }, 0);
  });

  onMount(() => {
    document.title = 'MD';
  });

  const emojiMap: Record<string, string> = {
    smile: '😊', thumbsup: '👍', heart: '❤️', fire: '🔥', star: '⭐',
    rocket: '🚀', check: '✅', warning: '⚠️', info: 'ℹ️', bulb: '💡',
    eyes: '👀', tada: '🎉', wave: '👋', point_right: '👉', ok_hand: '👌',
    zap: '⚡', lock: '🔒', key: '🔑', bug: '🐛', wrench: '🔧',
    pencil: '✏️', book: '📖', folder: '📁', file: '📄', computer: '💻',
    coffee: '☕', pizza: '🍕', music: '🎵', art: '🎨', camera: '📷',
  };
</script>

<div class="preview-wrapper">
  <article
    class="prose preview-content"
    bind:this={container}
  >
    {#if renderedHtml}
      {@html renderedHtml}
    {:else}
      <div class="preview-empty">
        <div class="empty-icon">
          <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8z"/>
            <polyline points="14 2 14 8 20 8"/>
            <line x1="16" y1="13" x2="8" y2="13"/>
            <line x1="16" y1="17" x2="8" y2="17"/>
            <polyline points="10 9 9 9 8 9"/>
          </svg>
        </div>
        <p>Start writing to see a live preview</p>
        <span class="empty-hint">Supports full CommonMark, GFM tables, footnotes, emoji & syntax highlighting</span>
      </div>
    {/if}
  </article>
</div>

<style>
  .preview-wrapper {
    flex: 1;
    height: 100%;
    overflow-y: auto;
    position: relative;
  }

  .preview-content {
    padding: 2.5rem 3rem;
    max-width: 780px;
    margin: 0 auto;
    min-height: 100%;
  }

  .preview-empty {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 0.75rem;
    height: 50vh;
    color: var(--text-muted);
    font-size: 15px;
    font-family: var(--font-ui);
    text-align: center;
  }

  .empty-icon {
    opacity: 0.2;
    margin-bottom: 0.5rem;
  }

  .empty-hint {
    font-size: 12px;
    color: var(--text-muted);
    opacity: 0.6;
    max-width: 280px;
  }

  :global(.render-error) {
    color: var(--danger);
    font-family: var(--font-ui);
  }

  /* Copy button injected into pre */
  :global(pre) { position: relative !important; }

  :global(.copy-btn) {
    position: absolute;
    top: 0.5rem;
    right: 0.5rem;
    padding: 0.2rem 0.6rem;
    font-size: 11px;
    font-family: var(--font-ui);
    font-weight: 500;
    background: rgba(255, 255, 255, 0.06);
    color: rgba(255, 255, 255, 0.5);
    border: 1px solid rgba(255, 255, 255, 0.08);
    border-radius: var(--radius-sm);
    cursor: pointer;
    transition: all 0.2s;
    backdrop-filter: blur(8px);
    opacity: 0;
  }
  :global(pre:hover .copy-btn) { opacity: 1; }
  :global(.copy-btn:hover) {
    background: rgba(255, 255, 255, 0.12);
    color: rgba(255, 255, 255, 0.8);
  }
  :global(.copy-btn.copied) {
    background: rgba(16, 185, 129, 0.2);
    color: #10b981;
    border-color: rgba(16, 185, 129, 0.3);
  }

  /* Print */
  @media print {
    .preview-wrapper { border: none; }
    .preview-content { max-width: 100%; padding: 0; }
  }
</style>
