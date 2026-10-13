<script lang="ts">
  import { app } from '../lib/app.svelte'
</script>

<div class="empty">
  {#if app.loadError}
    <h2>Impossibile aprire il file dati</h2>
    <p class="error-text">{app.loadError}</p>
  {:else}
    <h2>Nessun file dati configurato</h2>
    <p class="muted">Scegli un file esistente (ad esempio nella cartella di Nextcloud) oppure creane uno nuovo.</p>
  {/if}
  <button class="primary" onclick={() => (app.view = 'settings')}>Apri Impostazioni</button>
</div>

<style>
  .empty {
    max-width: 520px;
    margin: 15vh auto 0;
    padding: 0 24px;
    text-align: center;
  }

  h2 {
    font-size: 18px;
  }

  p {
    line-height: 1.5;
    margin-bottom: 20px;
  }
</style>
