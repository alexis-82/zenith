<script lang="ts">
  import { app, parseError } from '../lib/app.svelte'
  import Modal from './Modal.svelte'

  let error = $state('')
  let busy = $state(false)

  async function run(fn: () => Promise<void>) {
    error = ''
    busy = true
    try {
      await fn()
    } catch (e) {
      error = parseError(e).message
    } finally {
      busy = false
    }
  }
</script>

{#if app.externalChange && !app.externalChange.conflict}
  <div class="banner" role="status">
    <span>Il file è stato modificato da un altro dispositivo.</span>
    {#if error}<span class="error-text">{error}</span>{/if}
    <button class="primary" disabled={busy} onclick={() => run(() => app.reload())}>Ricarica</button>
  </div>
{:else if app.externalChange?.conflict}
  <Modal title="Conflitto con un altro dispositivo">
    <p>
      Il file dati è stato modificato da un altro dispositivo mentre avevi modifiche non ancora salvate. Il salvataggio
      automatico è in pausa: scegli quale versione tenere.
    </p>
    {#if error}<p class="error-text">{error}</p>{/if}
    {#snippet actions()}
      <button disabled={busy} onclick={() => run(() => app.resolveConflict('reload'))}>Ricarica (perdi le tue modifiche)</button>
      <button class="primary" disabled={busy} onclick={() => run(() => app.resolveConflict('keep'))}>
        Mantieni le mie (sovrascrivi il file)
      </button>
    {/snippet}
  </Modal>
{/if}

<style>
  .banner {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 8px 24px;
    background: var(--warning-bg);
    color: var(--warning-text);
  }

  .banner span:first-child {
    flex: 1;
  }

  p {
    margin: 0 0 8px;
    line-height: 1.5;
  }
</style>
