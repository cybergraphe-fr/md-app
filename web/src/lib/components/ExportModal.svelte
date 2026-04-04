<script lang="ts">
  import { activeFileId, activeName, activeContent } from '$lib/stores/files';
  import { api, type PDFExportOptions } from '$lib/api';
  import { X, Download, Loader } from 'lucide-svelte';
  import DOMPurify from 'dompurify';
  import { get } from 'svelte/store';

  let { isOpen, onClose }: { isOpen: boolean; onClose: () => void } = $props();

  const formats = [
    { id: 'markdown', label: 'Markdown (.md)', desc: 'Source Markdown file', icon: '✍️' },
    { id: 'html', label: 'HTML', desc: 'Standalone web page', icon: '🌐' },
    { id: 'pdf', label: 'PDF', desc: 'Portable document', icon: '📄' },
    { id: 'docx', label: 'Word (.docx)', desc: 'Microsoft Word', icon: '📝' },
    { id: 'odt', label: 'OpenDocument (.odt)', desc: 'LibreOffice', icon: '📃' },
    { id: 'epub', label: 'EPUB', desc: 'E-book format', icon: '📚' },
    { id: 'latex', label: 'LaTeX (.tex)', desc: 'LaTeX source', icon: '🔣' },
    { id: 'rst', label: 'reStructuredText', desc: 'Python docs', icon: '🐍' },
    { id: 'asciidoc', label: 'AsciiDoc (.adoc)', desc: 'Technical docs', icon: '📋' },
    { id: 'textile', label: 'Textile', desc: 'Lightweight markup', icon: '🧵' },
    { id: 'mediawiki', label: 'MediaWiki', desc: 'Wikipedia format', icon: '📖' },
    { id: 'plain', label: 'Plain text (.txt)', desc: 'Strip formatting', icon: '📜' },
  ];

  let exporting: string | null = $state(null);
  let exportError: string | null = $state(null);
  let pdfMargin: string = $state('standard');
  let pdfHeader: string = $state('');
  let pdfFooter: string = $state('');

  const maxDecorLength = 120;

  const marginOptions = [
    { id: 'standard', label: 'Standard', desc: '2.5 cm' },
    { id: 'narrow', label: 'Narrow', desc: '1.5 cm' },
    { id: 'wide', label: 'Wide', desc: '3.5 cm' },
  ];

  function downloadBlob(blob: Blob, filename: string): void {
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = filename;
    a.click();
    setTimeout(() => URL.revokeObjectURL(url), 5000);
  }

  function buildPDFOptions(): PDFExportOptions {
    return {
      margin: pdfMargin,
      header: pdfHeader.trim(),
      footer: pdfFooter.trim(),
    };
  }

  async function handleExport(formatId: string): Promise<void> {
    const id = get(activeFileId);
    const name = get(activeName) || 'document';
    const content = get(activeContent);
    const ext = getExtension(formatId);

    exporting = formatId;
    exportError = null;

    try {
      // Markdown: pure client-side — content is already Markdown, no server call needed
      if (formatId === 'markdown') {
        const blob = new Blob([content ?? ''], { type: 'text/markdown;charset=utf-8' });
        downloadBlob(blob, `${name}${ext}`);
        exporting = null;
        return;
      }

      if (id) {
        // File is saved — use the saved-file export endpoint
        if (formatId === 'html') {
          const a = document.createElement('a');
          a.href = api.exportHTML(id);
          a.download = `${name}${ext}`;
          a.click();
        } else {
          const pdfOptions = formatId === 'pdf' ? buildPDFOptions() : undefined;
          const res = await fetch(api.exportFormat(id, formatId, pdfOptions), { method: 'POST' });
          if (!res.ok) {
            const err = await res.json().catch(() => ({ error: res.statusText }));
            throw new Error(err.error ?? `Export failed: HTTP ${res.status}`);
          }
          downloadBlob(await res.blob(), `${name}${ext}`);
        }
      } else {
        // File NOT saved — use raw export endpoint (no save required!)
        if (formatId === 'html') {
          // Generate HTML client-side from current content
          const safeName = name.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
          const safeContent = DOMPurify.sanitize(content ?? '');
          const blob = new Blob([
            `<!DOCTYPE html><html><head><meta charset="utf-8"><title>${safeName}</title></head><body>\n${safeContent}\n</body></html>`
          ], { type: 'text/html' });
          downloadBlob(blob, `${name}${ext}`);
        } else {
          const pdfOptions = formatId === 'pdf' ? buildPDFOptions() : undefined;
          const res = await fetch(api.exportRawFormat(formatId, pdfOptions), {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ content, name }),
          });
          if (!res.ok) {
            const err = await res.json().catch(() => ({ error: res.statusText }));
            throw new Error(err.error ?? `Export failed: HTTP ${res.status}`);
          }
          downloadBlob(await res.blob(), `${name}${ext}`);
        }
      }
    } catch (e: unknown) {
      exportError = e instanceof Error ? e.message : 'Export failed';
    } finally {
      exporting = null;
    }
  }

  function getExtension(formatId: string): string {
    const exts: Record<string, string> = {
      markdown: '.md', pdf: '.pdf', docx: '.docx', odt: '.odt', epub: '.epub',
      latex: '.tex', rst: '.rst', asciidoc: '.adoc', textile: '.textile',
      mediawiki: '.wiki', plain: '.txt', html: '.html',
    };
    return exts[formatId] ?? '.md';
  }

  function handleBackdropClick(e: MouseEvent): void {
    if (e.target === e.currentTarget) onClose();
  }

  function handleKeydown(e: KeyboardEvent): void {
    if (e.key === 'Escape') onClose();
  }
</script>

{#if isOpen}
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div
    class="modal-backdrop"
    onclick={handleBackdropClick}
    onkeydown={handleKeydown}
    role="dialog"
    aria-modal="true"
    aria-label="Export dialog"
    tabindex="-1"
  >
    <div class="modal">
      <div class="modal-header">
        <div>
          <h2 class="modal-title">Export document</h2>
          <p class="modal-subtitle">
            {#if $activeFileId}
              Download "<strong>{$activeName}</strong>"
            {:else}
              Export current content <span class="badge">no save needed</span>
            {/if}
          </p>
        </div>
        <button class="btn-icon" onclick={onClose} aria-label="Close">
          <X size={18} />
        </button>
      </div>

      {#if exportError}
        <div class="export-error">
          <span>⚠</span> {exportError}
        </div>
      {/if}

      <div class="margin-selector">
        <span class="margin-label">PDF margins</span>
        <div class="margin-options">
          {#each marginOptions as opt}
            <button
              class="margin-btn"
              class:active={pdfMargin === opt.id}
              onclick={() => pdfMargin = opt.id}
            >
              <span class="margin-btn-label">{opt.label}</span>
              <span class="margin-btn-desc">{opt.desc}</span>
            </button>
          {/each}
        </div>
      </div>

      <div class="pdf-decor">
        <div class="pdf-decor-head">
          <span class="margin-label">PDF header & footer</span>
          <span class="decor-meta">optional, max {maxDecorLength} chars</span>
        </div>

        <div class="pdf-decor-grid">
          <label class="decor-field" for="pdf-header-input">
            <span class="decor-label">Header</span>
            <input
              id="pdf-header-input"
              class="decor-input"
              type="text"
              maxlength={maxDecorLength}
              bind:value={pdfHeader}
              placeholder="Example: Confidential report"
            />
          </label>

          <label class="decor-field" for="pdf-footer-input">
            <span class="decor-label">Footer</span>
            <input
              id="pdf-footer-input"
              class="decor-input"
              type="text"
              maxlength={maxDecorLength}
              bind:value={pdfFooter}
              placeholder="Example: Internal use only"
            />
          </label>
        </div>

        <p class="decor-note">Page numbering is automatic in PDF: current/total (example: 9/14).</p>
      </div>

      <div class="formats-grid">
        {#each formats as fmt}
          <button
            class="format-card"
            class:loading={exporting === fmt.id}
            onclick={() => handleExport(fmt.id)}
            disabled={!!exporting}
          >
            <span class="format-icon">{fmt.icon}</span>
            <div class="format-info">
              <span class="format-label">{fmt.label}</span>
              <span class="format-desc">{fmt.desc}</span>
            </div>
            {#if exporting === fmt.id}
              <Loader size={14} class="spin" />
            {:else}
              <Download size={14} class="format-dl-icon" />
            {/if}
          </button>
        {/each}
      </div>

      <div class="modal-footer">
        <button class="btn" onclick={onClose}>Close</button>
      </div>
    </div>
  </div>
{/if}

<style>
  .modal-backdrop {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.55);
    backdrop-filter: blur(6px);
    -webkit-backdrop-filter: blur(6px);
    z-index: 1000;
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 1rem;
    animation: fade-in 0.15s ease-out;
  }

  @keyframes fade-in {
    from { opacity: 0; }
    to { opacity: 1; }
  }

  .modal {
    background: var(--bg-editor);
    backdrop-filter: var(--glass-blur);
    -webkit-backdrop-filter: var(--glass-blur);
    border: 1px solid var(--border);
    border-radius: var(--radius-xl);
    box-shadow: 0 24px 64px rgba(0, 0, 0, 0.5), 0 0 0 1px rgba(255,255,255,0.04);
    width: 100%;
    max-width: 560px;
    max-height: 90vh;
    overflow-y: auto;
    display: flex;
    flex-direction: column;
    animation: modal-in 0.2s ease-out;
  }

  @keyframes modal-in {
    from { opacity: 0; transform: translateY(10px) scale(0.98); }
    to { opacity: 1; transform: translateY(0) scale(1); }
  }

  .modal-header {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    padding: 1.25rem 1.5rem 1rem;
    border-bottom: 1px solid var(--border-subtle);
  }

  .modal-title {
    font-size: 1.1rem;
    font-weight: 700;
    margin: 0 0 0.25rem;
    font-family: var(--font-ui);
    color: var(--text-primary);
  }

  .modal-subtitle {
    font-size: 13px;
    color: var(--text-secondary);
    margin: 0;
    font-family: var(--font-ui);
  }

  .badge {
    display: inline-block;
    font-size: 10px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.04em;
    background: var(--accent-surface);
    color: var(--accent);
    border: 1px solid var(--accent-light);
    border-radius: 20px;
    padding: 0.1rem 0.5rem;
    margin-left: 0.3rem;
    vertical-align: middle;
  }

  .export-error {
    margin: 0.75rem 1.5rem 0;
    padding: 0.6rem 0.9rem;
    background: var(--danger-light);
    border: 1px solid rgba(239, 68, 68, 0.2);
    border-radius: var(--radius-sm);
    color: var(--danger);
    font-size: 13px;
    font-family: var(--font-ui);
  }

  .margin-selector {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    padding: 0.6rem 1.5rem;
    border-bottom: 1px solid var(--border-subtle);
    background: var(--bg-surface);
  }

  .margin-label {
    font-size: 12px;
    font-weight: 600;
    color: var(--text-secondary);
    font-family: var(--font-ui);
    white-space: nowrap;
    text-transform: uppercase;
    letter-spacing: 0.04em;
  }

  .margin-options {
    display: flex;
    gap: 0.35rem;
    flex: 1;
  }

  .margin-btn {
    flex: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 0.05rem;
    padding: 0.35rem 0.5rem;
    border: 1px solid var(--border-subtle);
    border-radius: var(--radius-sm);
    background: var(--bg-surface);
    cursor: pointer;
    transition: all 0.15s;
    font-family: var(--font-ui);
    color: var(--text-secondary);
  }
  .margin-btn:hover { background: var(--bg-hover); border-color: var(--border); }
  .margin-btn.active {
    background: var(--accent-surface);
    border-color: var(--accent);
    color: var(--accent);
  }
  .margin-btn-label { font-size: 12px; font-weight: 600; }
  .margin-btn-desc { font-size: 10px; opacity: 0.7; }

  .formats-grid {
    display: flex;
    flex-direction: column;
    padding: 0.75rem 1rem;
    gap: 0.3rem;
    background: var(--bg-surface);
  }

  .format-card {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    padding: 0.65rem 0.9rem;
    border: 1px solid var(--border-subtle);
    border-radius: var(--radius);
    background: var(--bg-editor);
    cursor: pointer;
    transition: all 0.15s;
    text-align: left;
    width: 100%;
    font-family: var(--font-ui);
    color: inherit;
  }
  .format-card:hover:not(:disabled) {
    background: var(--bg-hover);
    border-color: var(--accent);
    box-shadow: 0 0 0 1px var(--accent-light);
  }
  .format-card:disabled { opacity: 0.5; cursor: wait; }
  .format-card.loading {
    background: var(--accent-surface);
    border-color: var(--accent);
  }

  .format-icon { font-size: 1.25rem; flex-shrink: 0; }

  .format-info {
    flex: 1;
    display: flex;
    flex-direction: column;
    gap: 0.1rem;
  }

  .format-label {
    font-size: 13px;
    font-weight: 600;
    color: var(--text-primary);
  }

  .format-desc {
    font-size: 11px;
    color: var(--text-secondary);
  }

  .pdf-decor {
    padding: 0.75rem 1.5rem;
    border-bottom: 1px solid var(--border-subtle);
    background: var(--bg-editor);
    display: flex;
    flex-direction: column;
    gap: 0.55rem;
  }

  .pdf-decor-head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 0.6rem;
  }

  .decor-meta {
    font-size: 11px;
    color: var(--text-secondary);
    font-family: var(--font-ui);
  }

  .pdf-decor-grid {
    display: grid;
    gap: 0.55rem;
    grid-template-columns: 1fr;
  }

  .decor-field {
    display: flex;
    flex-direction: column;
    gap: 0.25rem;
  }

  .decor-label {
    font-size: 12px;
    font-weight: 600;
    color: var(--text-primary);
    font-family: var(--font-ui);
  }

  .decor-input {
    width: 100%;
    border: 1px solid var(--border);
    border-radius: var(--radius-sm);
    min-height: 34px;
    padding: 0.45rem 0.55rem;
    background: var(--bg-surface);
    color: var(--text-primary);
    font-size: 13px;
    font-family: var(--font-ui);
    transition: border-color 0.15s, box-shadow 0.15s, background-color 0.15s;
  }

  .decor-input::placeholder {
    color: var(--text-secondary);
  }

  .decor-input:focus-visible {
    outline: none;
    border-color: var(--accent);
    box-shadow: 0 0 0 3px var(--accent-light);
    background: var(--bg-editor);
  }

  .decor-note {
    margin: 0;
    font-size: 11px;
    color: var(--text-secondary);
    font-family: var(--font-ui);
  }

  :global([data-theme='light']) .modal {
    background: rgba(255, 255, 255, 0.97);
    border-color: rgba(15, 23, 42, 0.16);
    box-shadow: 0 24px 64px rgba(15, 23, 42, 0.18), 0 1px 0 rgba(255, 255, 255, 0.8) inset;
  }

  :global([data-theme='light']) .modal-backdrop {
    background: rgba(15, 23, 42, 0.34);
  }

  :global([data-theme='light']) .margin-selector,
  :global([data-theme='light']) .formats-grid,
  :global([data-theme='light']) .pdf-decor {
    background: rgba(248, 250, 252, 0.9);
  }

  :global([data-theme='light']) .margin-btn,
  :global([data-theme='light']) .format-card,
  :global([data-theme='light']) .decor-input {
    border-color: rgba(15, 23, 42, 0.16);
    background: rgba(255, 255, 255, 0.96);
  }

  :global([data-theme='light']) .format-card:hover:not(:disabled) {
    background: #ffffff;
    box-shadow: 0 0 0 1px rgba(124, 58, 237, 0.2);
  }

  :global([data-theme='light']) .format-desc,
  :global([data-theme='light']) .decor-meta,
  :global([data-theme='light']) .decor-note,
  :global([data-theme='light']) .margin-label,
  :global([data-theme='light']) .modal-subtitle {
    color: #475569;
  }

  :global(.format-dl-icon) { color: var(--text-muted); flex-shrink: 0; transition: color 0.15s; }
  .format-card:hover :global(.format-dl-icon) { color: var(--accent); }

  :global(.spin) {
    animation: spin 0.8s linear infinite;
    color: var(--accent);
    flex-shrink: 0;
  }
  @keyframes spin { to { transform: rotate(360deg); } }

  .modal-footer {
    padding: 0.75rem 1.5rem 1.25rem;
    border-top: 1px solid var(--border-subtle);
    display: flex;
    align-items: center;
    justify-content: flex-end;
  }
</style>
