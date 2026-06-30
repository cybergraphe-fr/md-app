<script lang="ts">
  import {
    previewWidth,
    previewWidthPresets,
    setPreviewWidth,
    PREVIEW_WIDTH_MIN,
    PREVIEW_WIDTH_MAX,
  } from '$lib/stores/files';
  import { MoveHorizontal } from 'lucide-svelte';

  let open = $state(false);
  let pickerEl = $state<HTMLElement | undefined>(undefined);

  // Valeur numérique pour le curseur (en mode « pleine largeur », on l'aligne sur le max).
  let sliderValue = $derived(
    typeof $previewWidth === 'number' ? $previewWidth : PREVIEW_WIDTH_MAX
  );
  let currentLabel = $derived(
    $previewWidth === 'full' ? 'Pleine largeur' : `${$previewWidth} px`
  );

  function toggle(): void {
    open = !open;
  }

  function handleWindowClick(e: MouseEvent): void {
    if (open && pickerEl && !pickerEl.contains(e.target as Node)) {
      open = false;
    }
  }

  function onSlider(e: Event): void {
    setPreviewWidth(Number((e.currentTarget as HTMLInputElement).value));
  }
</script>

<svelte:window onclick={handleWindowClick} />

<div class="pw-picker" bind:this={pickerEl}>
  <button
    class="btn btn-icon pw-trigger"
    title="Largeur de l'aperçu ({currentLabel})"
    aria-label="Largeur de l'aperçu"
    onclick={toggle}
  >
    <MoveHorizontal size={15} />
  </button>

  {#if open}
    <div class="pw-dropdown">
      <div class="pw-header">Largeur de l'aperçu</div>
      <div class="pw-sub">Confort de lecture web / formats paysage</div>

      <div class="pw-presets">
        {#each previewWidthPresets as preset}
          <button
            class="pw-preset"
            class:active={$previewWidth === preset.value}
            onclick={() => setPreviewWidth(preset.value)}
          >
            {preset.label}
          </button>
        {/each}
      </div>

      <div class="pw-separator"></div>

      <div class="pw-custom">
        <div class="pw-custom-head">
          <span>Sur mesure</span>
          <span class="pw-value">{currentLabel}</span>
        </div>
        <input
          type="range"
          min={PREVIEW_WIDTH_MIN}
          max={PREVIEW_WIDTH_MAX}
          step="20"
          value={sliderValue}
          class:dim={$previewWidth === 'full'}
          oninput={onSlider}
          aria-label="Largeur en pixels"
        />
        <div class="pw-scale">
          <span>{PREVIEW_WIDTH_MIN}</span>
          <span>{PREVIEW_WIDTH_MAX}</span>
        </div>
      </div>
    </div>
  {/if}
</div>

<style>
  .pw-picker { position: relative; }
  .pw-trigger {
    display: flex;
    align-items: center;
    color: var(--text-secondary);
  }

  .pw-dropdown {
    position: absolute;
    top: calc(100% + 6px);
    right: 0;
    z-index: 100;
    width: 248px;
    background: var(--bg-surface);
    border: 1px solid var(--border);
    border-radius: var(--radius);
    box-shadow: var(--shadow-lg);
    backdrop-filter: var(--glass-blur);
    -webkit-backdrop-filter: var(--glass-blur);
    padding-bottom: 0.65rem;
    animation: pw-in 0.15s ease-out;
  }

  @keyframes pw-in {
    from { opacity: 0; transform: translateY(-4px); }
    to { opacity: 1; transform: none; }
  }

  .pw-header {
    padding: 0.6rem 0.75rem 0;
    font-size: 12px;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.04em;
    color: var(--text-primary);
    font-family: var(--font-ui);
  }
  .pw-sub {
    padding: 0 0.75rem 0.5rem;
    font-size: 10px;
    color: var(--text-muted);
    font-family: var(--font-ui);
  }

  .pw-presets {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 0.35rem;
    padding: 0 0.75rem;
  }
  .pw-preset {
    padding: 0.4rem 0.5rem;
    font-size: 13px;
    font-family: var(--font-ui);
    border: 1px solid var(--border);
    border-radius: var(--radius-sm);
    background: transparent;
    color: var(--text-primary);
    cursor: pointer;
    transition: background 0.1s, border-color 0.1s;
  }
  .pw-preset:hover { background: var(--bg-hover); }
  .pw-preset.active {
    background: var(--accent-surface);
    color: var(--accent);
    border-color: var(--accent);
    font-weight: 600;
  }

  .pw-separator {
    height: 1px;
    background: var(--border);
    margin: 0.65rem 0.75rem 0.5rem;
  }

  .pw-custom { padding: 0 0.75rem; }
  .pw-custom-head {
    display: flex;
    justify-content: space-between;
    align-items: baseline;
    font-size: 11px;
    font-family: var(--font-ui);
    color: var(--text-muted);
    margin-bottom: 0.3rem;
  }
  .pw-value { color: var(--accent); font-weight: 600; }

  .pw-custom input[type='range'] {
    width: 100%;
    accent-color: var(--accent);
    cursor: pointer;
  }
  .pw-custom input[type='range'].dim { opacity: 0.45; }

  .pw-scale {
    display: flex;
    justify-content: space-between;
    font-size: 9px;
    color: var(--text-muted);
    font-family: var(--font-ui);
    margin-top: 0.15rem;
  }
</style>
