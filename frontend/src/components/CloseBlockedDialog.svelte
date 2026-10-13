<script lang="ts">
  import * as api from '../../wailsjs/go/main/App'
  import { app, parseError } from '../lib/app.svelte'
  import Modal from './Modal.svelte'

  let error = $state('')

  // Quit(false) saves first and re-emits app:close-blocked if it fails.
  function retry() {
    app.closeBlocked = null
    api.Quit(false)
  }

  async function keepAndQuit() {
    try {
      await app.resolveConflict('keep')
      app.closeBlocked = null
      api.Quit(false)
    } catch (e) {
      error = parseError(e).message
    }
  }
</script>

{#if app.closeBlocked}
  <div class="top">
    <Modal title="Modifiche non salvate" onclose={() => (app.closeBlocked = null)}>
      {#if app.closeBlocked.reason === 'conflict'}
        <p>
          Il file è stato modificato da un altro dispositivo e hai modifiche non salvate. Vuoi sovrascrivere il file con le
          tue modifiche prima di uscire?
        </p>
      {:else}
        <p>Non è stato possibile salvare le modifiche prima di uscire.</p>
        {#if app.closeBlocked.error}<p class="error-text">{app.closeBlocked.error}</p>{/if}
      {/if}
      {#if error}<p class="error-text">{error}</p>{/if}
      {#snippet actions()}
        <button onclick={() => (app.closeBlocked = null)}>Annulla</button>
        <button class="danger" onclick={() => api.Quit(true)}>Esci senza salvare</button>
        {#if app.closeBlocked?.reason === 'conflict'}
          <button class="primary" onclick={keepAndQuit}>Mantieni le mie e esci</button>
        {:else}
          <button class="primary" onclick={retry}>Riprova</button>
        {/if}
      {/snippet}
    </Modal>
  </div>
{/if}

<style>
  .top :global(.overlay) {
    z-index: 200;
  }

  p {
    margin: 0 0 8px;
    line-height: 1.5;
  }
</style>
