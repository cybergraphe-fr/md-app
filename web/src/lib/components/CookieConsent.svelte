<script lang="ts">
  let visible = $state(!document.cookie.includes('md-cookie-consent=accepted'));

  function accept(): void {
    document.cookie = 'md-cookie-consent=accepted;path=/;max-age=31536000;SameSite=Lax';
    visible = false;
  }
</script>

{#if visible}
  <div class="cookie-banner" role="alert">
    <div class="cookie-content">
      <p>
        Ce site utilise un <strong>cookie fonctionnel</strong> pour sauvegarder votre espace de
        travail. Aucun cookie publicitaire ni de traçage n'est utilisé.
      </p>
      <button class="cookie-accept" onclick={accept}>Accepter</button>
    </div>
  </div>
{/if}

<style>
  .cookie-banner {
    position: fixed;
    bottom: 0;
    left: 0;
    right: 0;
    z-index: 9999;
    padding: 0.75rem 1rem;
    background: var(--surface-elevated, #1e1e2e);
    border-top: 1px solid var(--border, rgba(255, 255, 255, 0.08));
    backdrop-filter: blur(16px);
    animation: slideUp 0.3s ease-out;
  }

  @keyframes slideUp {
    from { transform: translateY(100%); opacity: 0; }
    to   { transform: translateY(0);    opacity: 1; }
  }

  .cookie-content {
    max-width: 960px;
    margin: 0 auto;
    display: flex;
    align-items: center;
    gap: 1rem;
    flex-wrap: wrap;
  }

  .cookie-content p {
    margin: 0;
    flex: 1 1 300px;
    font-size: 0.82rem;
    line-height: 1.4;
    color: var(--text-secondary, #a0a0b8);
  }

  .cookie-accept {
    flex-shrink: 0;
    padding: 0.45rem 1.2rem;
    border: none;
    border-radius: 6px;
    background: var(--accent, #6366f1);
    color: #fff;
    font-size: 0.82rem;
    font-weight: 600;
    cursor: pointer;
    transition: background 0.15s;
  }

  .cookie-accept:hover {
    background: var(--accent-hover, #818cf8);
  }

  @media print {
    .cookie-banner { display: none; }
  }
</style>
