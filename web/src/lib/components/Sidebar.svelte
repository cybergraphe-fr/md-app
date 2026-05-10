<script lang="ts">
  import {
    files,
    filteredFiles,
    tocHeadings,
    tocActiveHeadingId,
    activeFileId,
    isLoading,
    searchQuery,
    openFile,
    createFile,
    deleteFile,
    importFile,
    jumpToHeading,
  } from '$lib/stores/files';
  import { FilePlus, Search, Trash2, Upload, FileText, List } from 'lucide-svelte';

  let fileInput: HTMLInputElement;
  let navMode = $state<'files' | 'toc'>('files');

  function formatDate(iso: string): string {
    return new Intl.DateTimeFormat('fr-FR', {
      day: '2-digit', month: '2-digit', year: '2-digit',
      hour: '2-digit', minute: '2-digit',
    }).format(new Date(iso));
  }

  function formatSize(bytes: number): string {
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
    return `${(bytes / (1024 * 1024)).toFixed(2)} MB`;
  }

  async function handleDelete(id: string, name: string): Promise<void> {
    if (!confirm(`Delete "${name}"? This action cannot be undone.`)) return;
    await deleteFile(id);
  }

  async function handleImport(e: Event): Promise<void> {
    const target = e.target as HTMLInputElement;
    const file = target.files?.[0];
    if (!file) return;
    await importFile(file);
    target.value = '';
  }

  function handleDrop(e: DragEvent): void {
    e.preventDefault();
    const file = e.dataTransfer?.files?.[0];
    if (file) importFile(file);
  }
</script>

<!-- svelte-ignore a11y_no_static_element_interactions -->
<aside
  class="sidebar"
  ondragover={(e) => e.preventDefault()}
  ondrop={handleDrop}
>
  <!-- Header -->
  <div class="sidebar-header">
    <div class="sidebar-brand">
      <div class="brand-icon">
        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
          <path d="M12 20h9" /><path d="M16.5 3.5a2.121 2.121 0 013 3L7 19l-4 1 1-4L16.5 3.5z" />
        </svg>
      </div>
      <span class="brand-name">MD</span>
    </div>
    <div class="sidebar-actions">
      <button class="btn-icon" title="New file" onclick={() => createFile()}>
        <FilePlus size={15} />
      </button>
      <button class="btn-icon" title="Import file" onclick={() => fileInput.click()}>
        <Upload size={15} />
      </button>
      <input
        bind:this={fileInput}
        type="file"
        accept=".md,.txt,.html,.markdown"
        style="display:none"
        onchange={handleImport}
      />
    </div>
  </div>

  <!-- Search -->
  <div class="sidebar-search">
    <Search size={13} />
    <input
      type="search"
      placeholder="Search files…"
      bind:value={$searchQuery}
      class="search-input"
    />
  </div>

  <div class="sidebar-nav-toggle" role="tablist" aria-label="Sidebar navigation mode">
    <button class="toggle-btn" class:active={navMode === 'files'} onclick={() => (navMode = 'files')}>
      Files
    </button>
    <button class="toggle-btn" class:active={navMode === 'toc'} onclick={() => (navMode = 'toc')}>
      <List size={12} />
      Table of contents
    </button>
  </div>

  {#if navMode === 'files'}
    <!-- File list -->
    <div class="file-list" role="list">
      {#if $isLoading && $files.length === 0}
        <div class="file-list-empty">
          <div class="loading-dots"><span></span><span></span><span></span></div>
        </div>
      {:else if $filteredFiles.length === 0}
        <div class="file-list-empty">
          {#if $searchQuery}
            <p>No results for "<strong>{$searchQuery}</strong>"</p>
          {:else}
            <FileText size={28} color="var(--text-muted)" />
            <p>No files yet.<br />Create your first document!</p>
            <button class="btn btn-primary" onclick={() => createFile()}>
              <FilePlus size={14} /> New file
            </button>
          {/if}
        </div>
      {:else}
        {#each $filteredFiles as f (f.id)}
          <!-- svelte-ignore a11y_no_static_element_interactions -->
          <div
            class="file-item"
            class:active={$activeFileId === f.id}
            onclick={() => openFile(f.id)}
            onkeydown={(e) => { if (e.key === 'Enter') openFile(f.id); }}
            role="button"
            tabindex="0"
          >
            <div class="file-item-content">
              <div class="file-name">{f.name || 'untitled'}</div>
              <div class="file-meta">
                <span>{formatDate(f.updated_at)}</span>
                <span>{formatSize(f.size)}</span>
              </div>
            </div>
            <button
              class="btn-icon file-delete"
              title="Delete"
              onclick={(e) => { e.stopPropagation(); handleDelete(f.id, f.name); }}
            >
              <Trash2 size={13} />
            </button>
          </div>
        {/each}
      {/if}
    </div>
  {:else}
    <div class="toc-list" role="list" aria-label="Document table of contents">
      {#if $tocHeadings.length === 0}
        <div class="file-list-empty">
          <List size={26} color="var(--text-muted)" />
          <p>No headings found in this document yet.</p>
        </div>
      {:else}
        {#each $tocHeadings as item (item.id)}
          <button
            class="toc-item"
            class:active={$tocActiveHeadingId === item.id}
            style={`--toc-level:${Math.min(Math.max(item.level, 1), 6)}`}
            onclick={() => jumpToHeading(item.id)}
            title={`Go to ${item.text}`}
          >
            <span class="toc-level">H{item.level}</span>
            <span class="toc-text">{item.text}</span>
          </button>
        {/each}
      {/if}
    </div>
  {/if}

  <!-- Footer -->
  <div class="sidebar-footer">
    {#if navMode === 'files'}
      <span>{$files.length} file{$files.length !== 1 ? 's' : ''}</span>
    {:else}
      <span>{$tocHeadings.length} heading{$tocHeadings.length !== 1 ? 's' : ''}</span>
    {/if}
  </div>
</aside>

<style>
  .sidebar {
    display: flex;
    flex-direction: column;
    width: 260px;
    min-width: 220px;
    max-width: 320px;
    height: 100%;
    overflow: hidden;
    flex-shrink: 0;
    background: var(--bg-sidebar);
    backdrop-filter: var(--glass-blur);
    -webkit-backdrop-filter: var(--glass-blur);
    border-right: 1px solid var(--border);
  }

  .sidebar-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0.8rem 1rem;
    border-bottom: 1px solid var(--border-subtle);
  }

  .sidebar-brand {
    display: flex;
    align-items: center;
    gap: 0.55rem;
  }

  .brand-icon {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 28px;
    height: 28px;
    border-radius: var(--radius-sm);
    background: var(--accent);
    color: white;
    box-shadow: 0 0 16px var(--accent-glow);
  }

  .brand-name {
    font-family: var(--font-ui);
    font-weight: 700;
    font-size: 15px;
    color: var(--text-primary);
    letter-spacing: -0.02em;
  }

  .sidebar-actions {
    display: flex;
    gap: 0.15rem;
  }

  .sidebar-search {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    padding: 0.6rem 0.75rem;
    border-bottom: 1px solid var(--border-subtle);
    color: var(--text-muted);
  }

  .sidebar-nav-toggle {
    display: flex;
    align-items: center;
    gap: 0.35rem;
    padding: 0.5rem 0.7rem;
    border-bottom: 1px solid var(--border-subtle);
  }

  .toggle-btn {
    flex: 1;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: 0.35rem;
    padding: 0.3rem 0.45rem;
    border: 1px solid transparent;
    border-radius: var(--radius-sm);
    background: transparent;
    color: var(--text-secondary);
    font-size: 12px;
    font-family: var(--font-ui);
    cursor: pointer;
    transition: all 0.15s;
  }
  .toggle-btn:hover {
    background: var(--bg-hover);
    color: var(--text-primary);
  }
  .toggle-btn.active {
    background: var(--accent-surface);
    border-color: color-mix(in srgb, var(--accent) 35%, transparent);
    color: var(--accent);
    font-weight: 600;
  }

  .search-input {
    flex: 1;
    padding: 0.35rem 0.5rem;
    font-size: 13px;
    font-family: var(--font-ui);
    background: var(--bg-hover);
    border: 1px solid transparent;
    border-radius: var(--radius-sm);
    color: var(--text-primary);
    outline: none;
    transition: all var(--transition);
  }
  .search-input:focus {
    border-color: var(--accent);
    background: var(--bg-active);
    box-shadow: 0 0 0 2px var(--accent-light);
  }
  .search-input::placeholder { color: var(--text-muted); }

  .file-list {
    flex: 1;
    overflow-y: auto;
    padding: 0.25rem 0;
  }

  .toc-list {
    flex: 1;
    overflow-y: auto;
    padding: 0.25rem 0;
    display: flex;
    flex-direction: column;
    gap: 0.1rem;
  }

  .toc-item {
    width: calc(100% - 0.45rem);
    margin: 0 0.225rem;
    border: 1px solid transparent;
    border-radius: var(--radius-sm);
    background: transparent;
    color: var(--text-secondary);
    display: grid;
    grid-template-columns: auto minmax(0, 1fr);
    align-items: center;
    gap: 0.45rem;
    padding: 0.38rem 0.42rem;
    padding-left: calc(0.42rem + (var(--toc-level) - 1) * 0.5rem);
    cursor: pointer;
    text-align: left;
    font-family: var(--font-ui);
  }
  .toc-item:hover {
    background: var(--bg-hover);
    color: var(--text-primary);
  }
  .toc-item.active {
    background: var(--accent-surface);
    border-color: color-mix(in srgb, var(--accent) 28%, transparent);
    color: var(--accent);
  }

  .toc-level {
    font-size: 10px;
    font-weight: 700;
    opacity: 0.75;
    letter-spacing: 0.04em;
  }

  .toc-text {
    font-size: 12px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .file-list-empty {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 0.75rem;
    padding: 2.5rem 1.5rem;
    text-align: center;
    color: var(--text-muted);
    font-size: 13px;
  }

  .file-item {
    display: flex;
    align-items: center;
    gap: 0.4rem;
    width: 100%;
    padding: 0.55rem 0.75rem;
    cursor: pointer;
    border: none;
    border-left: 2px solid transparent;
    background: transparent;
    color: inherit;
    text-align: left;
    font: inherit;
    transition: all 0.12s;
    user-select: none;
  }
  .file-item:hover { background: var(--bg-hover); }
  .file-item.active {
    background: var(--accent-surface);
    border-left-color: var(--accent);
  }
  .file-item:hover .file-delete { opacity: 1; }

  .file-item-content { flex: 1; min-width: 0; }

  .file-name {
    font-size: 13px;
    font-weight: 500;
    color: var(--text-primary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .file-meta {
    display: flex;
    gap: 0.6rem;
    font-size: 11px;
    color: var(--text-muted);
    margin-top: 0.1rem;
  }

  .file-delete {
    opacity: 0;
    transition: opacity 0.15s;
    color: var(--danger) !important;
    padding: 0.2rem;
  }

  .sidebar-footer {
    padding: 0.5rem 1rem;
    border-top: 1px solid var(--border-subtle);
    font-size: 11px;
    color: var(--text-muted);
  }

  /* Loading dots */
  .loading-dots { display: flex; gap: 4px; }
  .loading-dots span {
    width: 6px; height: 6px;
    border-radius: 50%;
    background: var(--accent);
    animation: bounce 1.2s infinite;
  }
  .loading-dots span:nth-child(2) { animation-delay: 0.2s; }
  .loading-dots span:nth-child(3) { animation-delay: 0.4s; }
  @keyframes bounce {
    0%, 80%, 100% { transform: scale(0.7); opacity: 0.4; }
    40% { transform: scale(1.1); opacity: 1; }
  }
</style>
