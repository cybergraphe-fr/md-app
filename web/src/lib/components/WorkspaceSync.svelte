<script lang="ts">
  import { api, type DesktopDownloadVariant } from '$lib/api';
  import { detectDesktopClient, desktopClientLabel, pickDesktopDownload } from '$lib/desktop-download';
  import { loadFiles } from '$lib/stores/files';
  import { Link2, Copy, Check, Laptop } from 'lucide-svelte';

  let { isOpen = $bindable(false) } = $props<{ isOpen: boolean }>();

  let syncCode = $state('');
  let syncStatus = $state<'idle' | 'loading' | 'ready' | 'error'>('idle');
  let syncError = $state('');
  let linkCode = $state('');
  let status = $state<'idle' | 'loading' | 'success' | 'error'>('idle');
  let message = $state('');
  let copied = $state(false);
  let fetchRequestId = 0;
  let desktopDownloads = $state<DesktopDownloadVariant[]>([]);
  let desktopPageURL = $state('');
  let desktopStatus = $state<'idle' | 'loading' | 'ready' | 'error'>('idle');
  let desktopError = $state('');
  let desktopRequestId = 0;
  const detectedClient = detectDesktopClient();

  async function fetchInfo(): Promise<void> {
    const reqID = ++fetchRequestId;
    syncStatus = 'loading';
    syncError = '';
    syncCode = '';
    try {
      const info = await api.getWorkspace();
      if (reqID !== fetchRequestId) return;
      const code = (info.sync_code ?? '').trim().toLowerCase();
      if (!/^[a-z0-9]{8}$/.test(code)) throw new Error('invalid sync code payload');
      syncCode = code;
      syncStatus = 'ready';
    } catch {
      if (reqID !== fetchRequestId) return;
      syncCode = '--------';
      syncStatus = 'error';
      syncError = 'Code indisponible temporairement. Reessayez.';
    }
  }

  async function fetchDesktopDownloads(): Promise<void> {
    const reqID = ++desktopRequestId;
    desktopStatus = 'loading';
    desktopError = '';

    try {
      const payload = await api.getDesktopDownloads();
      if (reqID !== desktopRequestId) return;
      desktopDownloads = payload.variants ?? [];
      desktopPageURL = payload.page_url ?? '';
      desktopStatus = 'ready';
    } catch {
      if (reqID !== desktopRequestId) return;
      desktopStatus = 'error';
      desktopError = 'Catalogue desktop indisponible pour le moment.';
      desktopDownloads = [];
      desktopPageURL = '';
    }
  }

  function availableDesktopDownloads(): DesktopDownloadVariant[] {
    return desktopDownloads.filter((variant) => variant.available && !!variant.url);
  }

  function recommendedDesktopDownload(): DesktopDownloadVariant | null {
    return pickDesktopDownload(availableDesktopDownloads(), detectedClient);
  }

  function alternateDesktopDownloads(recommendedID: string): DesktopDownloadVariant[] {
    return availableDesktopDownloads().filter((variant) => variant.id !== recommendedID);
  }

  async function linkWorkspace(): Promise<void> {
    const code = linkCode.trim().toLowerCase();
    if (code.length !== 8) {
      status = 'error';
      message = 'Le code doit contenir 8 caractères.';
      return;
    }
    status = 'loading';
    try {
      await api.linkWorkspace(code);
      status = 'success';
      message = 'Espace lié ! Rechargement…';
      await loadFiles();
      setTimeout(() => { isOpen = false; status = 'idle'; linkCode = ''; }, 1200);
    } catch (e: any) {
      status = 'error';
      message = e.message ?? 'Code inconnu';
    }
  }

  async function copyCode(): Promise<void> {
    if (syncStatus !== 'ready' || !syncCode) return;
    try {
      await navigator.clipboard.writeText(syncCode);
      copied = true;
      setTimeout(() => (copied = false), 1500);
    } catch {
      syncError = 'Impossible de copier le code pour le moment.';
    }
  }

  $effect(() => {
    if (isOpen) {
      void fetchInfo();
      void fetchDesktopDownloads();
      return;
    }
    fetchRequestId++;
    desktopRequestId++;
    syncStatus = 'idle';
    syncError = '';
    copied = false;
    desktopStatus = 'idle';
    desktopError = '';
    desktopDownloads = [];
    desktopPageURL = '';
  });
</script>

{#if isOpen}
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div class="overlay" onclick={() => (isOpen = false)} onkeydown={(e) => e.key === 'Escape' && (isOpen = false)}>
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div
      class="modal"
      role="dialog"
      aria-modal="true"
      aria-label="Synchroniser vos fichiers"
      tabindex="0"
      onclick={(e) => e.stopPropagation()}
      onkeydown={(e) => e.key === 'Escape' && (isOpen = false)}
    >
      <h3><Link2 size={16} /> Synchroniser vos fichiers</h3>

      <section>
        <p class="section-label" id="sync-code-label">Votre code de synchronisation</p>
        <div class="code-row" aria-labelledby="sync-code-label">
          <code class="sync-code" class:placeholder={syncStatus !== 'ready'}>
            {syncStatus === 'loading' ? 'chargement…' : (syncCode || '--------')}
          </code>
          <button class="btn-icon" title="Copier" disabled={syncStatus !== 'ready'} onclick={copyCode}>
            {#if copied}<Check size={14} />{:else}<Copy size={14} />{/if}
          </button>
        </div>
        <p class="hint">Entrez ce code sur un autre appareil pour retrouver vos fichiers.</p>
        {#if syncStatus === 'error'}
          <p class="status error">{syncError}</p>
          <button class="btn-secondary" onclick={() => void fetchInfo()}>Reessayer</button>
        {/if}
      </section>

      <hr />

      <section>
        <p class="section-label"><Laptop size={14} /> Client desktop recommande</p>

        {#if desktopStatus === 'loading'}
          <p class="hint">Detection OS en cours...</p>
        {:else if desktopStatus === 'error'}
          <p class="status error">{desktopError}</p>
          <button class="btn-secondary" onclick={() => void fetchDesktopDownloads()}>Reessayer</button>
        {:else}
          {@const recommended = recommendedDesktopDownload()}

          {#if recommended}
            <a class="btn-download" href={recommended.url} target="_blank" rel="noopener noreferrer">
              Telecharger {recommended.label}
            </a>
            <p class="download-context">
              OS detecte: {desktopClientLabel(detectedClient)}. Apres installation, ouvrez "Synchroniser" dans le desktop
              et saisissez ce code pour retrouver vos fichiers web.
            </p>

            {@const alternatives = alternateDesktopDownloads(recommended.id)}
            {#if alternatives.length > 0}
              <div class="download-links">
                {#each alternatives as variant}
                  <a href={variant.url} target="_blank" rel="noopener noreferrer">{variant.label}</a>
                {/each}
              </div>
            {/if}
          {:else if desktopPageURL}
            <a class="btn-download" href={desktopPageURL} target="_blank" rel="noopener noreferrer">
              Voir toutes les versions desktop
            </a>
            <p class="download-context">Aucun binaire cible n est encore publie pour cet OS.</p>
          {:else}
            <p class="hint">Les clients desktop seront proposes ici des qu ils sont publies.</p>
          {/if}
        {/if}
      </section>

      <hr />

      <section>
        <label for="link-input">Lier un espace existant</label>
        <div class="link-row">
          <input
            id="link-input"
            type="text"
            placeholder="code à 8 caractères"
            maxlength="8"
            bind:value={linkCode}
            onkeydown={(e) => e.key === 'Enter' && linkWorkspace()}
          />
          <button class="btn-primary" disabled={status === 'loading'} onclick={linkWorkspace}>
            {status === 'loading' ? '…' : 'Lier'}
          </button>
        </div>
        {#if message}
          <p class="status" class:error={status === 'error'} class:success={status === 'success'}>{message}</p>
        {/if}
      </section>

      <button class="btn-close" onclick={() => (isOpen = false)}>Fermer</button>
    </div>
  </div>
{/if}

<style>
  .overlay {
    position: fixed;
    inset: 0;
    z-index: 9000;
    display: flex;
    align-items: center;
    justify-content: center;
    background: rgba(0, 0, 0, 0.55);
    backdrop-filter: blur(4px);
  }

  .modal {
    background: var(--surface-elevated, #1e1e2e);
    border: 1px solid var(--border, rgba(255, 255, 255, 0.08));
    border-radius: 12px;
    padding: 1.5rem;
    width: min(400px, 90vw);
    color: var(--text-primary, #e0e0f0);
  }

  h3 {
    margin: 0 0 1rem;
    font-size: 1rem;
    display: flex;
    align-items: center;
    gap: 0.4rem;
  }

  label {
    display: block;
    font-size: 0.78rem;
    font-weight: 600;
    color: var(--text-secondary, #a0a0b8);
    margin-bottom: 0.35rem;
    text-transform: uppercase;
    letter-spacing: 0.03em;
  }

  .section-label {
    display: flex;
    align-items: center;
    gap: 0.35rem;
    font-size: 0.78rem;
    font-weight: 600;
    color: var(--text-secondary, #a0a0b8);
    margin: 0 0 0.35rem;
    text-transform: uppercase;
    letter-spacing: 0.03em;
  }

  .code-row {
    display: flex;
    align-items: center;
    gap: 0.5rem;
  }

  .sync-code {
    font-size: 1.4rem;
    font-weight: 700;
    letter-spacing: 0.15em;
    color: var(--accent, #6366f1);
  }

  .sync-code.placeholder {
    color: var(--text-secondary, #a0a0b8);
    letter-spacing: 0.08em;
    font-weight: 600;
  }

  .hint {
    margin: 0.3rem 0 0;
    font-size: 0.75rem;
    color: var(--text-secondary, #a0a0b8);
  }

  hr {
    border: none;
    border-top: 1px solid var(--border, rgba(255, 255, 255, 0.08));
    margin: 1rem 0;
  }

  .link-row {
    display: flex;
    gap: 0.5rem;
  }

  .link-row input {
    flex: 1;
    padding: 0.45rem 0.65rem;
    border: 1px solid var(--border, rgba(255, 255, 255, 0.12));
    border-radius: 6px;
    background: var(--surface, #12121a);
    color: var(--text-primary, #e0e0f0);
    font-family: monospace;
    font-size: 0.9rem;
    letter-spacing: 0.08em;
  }

  .btn-primary {
    padding: 0.45rem 1rem;
    border: none;
    border-radius: 6px;
    background: var(--accent, #6366f1);
    color: #fff;
    font-weight: 600;
    font-size: 0.82rem;
    cursor: pointer;
  }

  .btn-primary:hover { background: var(--accent-hover, #818cf8); }
  .btn-primary:disabled { opacity: 0.5; cursor: default; }

  .btn-icon {
    background: transparent;
    border: none;
    color: var(--text-secondary, #a0a0b8);
    cursor: pointer;
    padding: 4px;
  }

  .btn-icon:disabled {
    opacity: 0.45;
    cursor: not-allowed;
  }

  .btn-secondary {
    margin-top: 0.45rem;
    padding: 0.35rem 0.8rem;
    border: 1px solid var(--border, rgba(255, 255, 255, 0.12));
    border-radius: 6px;
    background: transparent;
    color: var(--text-secondary, #a0a0b8);
    font-size: 0.78rem;
    cursor: pointer;
  }

  .btn-secondary:hover {
    color: var(--text-primary, #e0e0f0);
  }

  .btn-download {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    min-height: 34px;
    padding: 0.45rem 0.9rem;
    border-radius: 6px;
    border: none;
    text-decoration: none;
    background: var(--accent, #6366f1);
    color: #fff;
    font-size: 0.82rem;
    font-weight: 600;
  }

  .btn-download:hover {
    background: var(--accent-hover, #818cf8);
  }

  .download-context {
    margin: 0.4rem 0 0;
    font-size: 0.75rem;
    color: var(--text-secondary, #a0a0b8);
    line-height: 1.4;
  }

  .download-links {
    margin-top: 0.45rem;
    display: flex;
    flex-wrap: wrap;
    gap: 0.4rem;
  }

  .download-links a {
    font-size: 0.74rem;
    color: var(--accent, #6366f1);
    text-decoration: none;
    border: 1px solid var(--border, rgba(255, 255, 255, 0.12));
    border-radius: 999px;
    padding: 0.2rem 0.55rem;
  }

  .download-links a:hover {
    color: #fff;
    border-color: var(--accent, #6366f1);
  }

  .status {
    margin: 0.35rem 0 0;
    font-size: 0.78rem;
  }
  .status.error { color: #f87171; }
  .status.success { color: #34d399; }

  .btn-close {
    display: block;
    margin: 1.2rem auto 0;
    padding: 0.4rem 1.6rem;
    border: 1px solid var(--border, rgba(255, 255, 255, 0.12));
    border-radius: 6px;
    background: transparent;
    color: var(--text-secondary, #a0a0b8);
    font-size: 0.82rem;
    cursor: pointer;
  }
  .btn-close:hover { color: var(--text-primary, #e0e0f0); }
</style>
