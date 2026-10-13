<script lang="ts">
  import { app } from '../lib/app.svelte'

  const labels = {
    saved: 'Salvato',
    dirty: 'Modifiche non salvate',
    saving: 'Salvataggio…',
    error: 'Errore di salvataggio',
    paused: 'In pausa: conflitto',
  }

  // Manual save is useful when there is something pending or a save failed;
  // during a conflict the choice is made in the conflict dialog.
  const canSave = $derived(app.saveStatus === 'dirty' || app.saveStatus === 'error')
</script>

{#if app.board}
  <div class="status {app.saveStatus}" title={app.saveError || labels[app.saveStatus]}>
    <span class="dot"></span>
    <span class="label">{labels[app.saveStatus]}</span>
    <button
      class="save"
      disabled={!canSave}
      title={app.saveStatus === 'paused' ? 'Risolvi prima il conflitto' : 'Salva ora sul file'}
      onclick={() => app.saveNow()}>Salva</button
    >
  </div>
  {#if app.saveStatus === 'error' && app.saveError}
    <div class="error-detail">{app.saveError}</div>
  {/if}
{/if}

<style>
  .status {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: 12px;
    color: var(--text-muted);
  }

  .dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    background: #3aa55d;
  }

  .label {
    flex: 1;
  }

  .save {
    padding: 2px 10px;
    font-size: 12px;
    background: var(--soft-button);
    border-color: var(--input-border);
  }

  .save:hover:not(:disabled) {
    background: var(--sidebar-item);
  }

  .error-detail {
    margin-top: 4px;
    font-size: 11px;
    color: var(--danger);
    overflow-wrap: anywhere;
  }

  .dirty .dot,
  .saving .dot {
    background: #d9a400;
  }

  .error,
  .paused {
    color: var(--danger);
  }

  .error .dot,
  .paused .dot {
    background: var(--danger);
  }
</style>
