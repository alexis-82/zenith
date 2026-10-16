<script lang="ts">
  import * as api from '../../wailsjs/go/main/App'
  import { app, parseError } from '../lib/app.svelte'
  import { isSafeUrl, renderMarkdown } from '../lib/markdown'
  import Modal from './Modal.svelte'

  let installing = $state(false)
  let error = $state('')

  const percent = $derived(
    app.updateProgress && app.updateProgress.total > 0
      ? Math.floor((app.updateProgress.done / app.updateProgress.total) * 100)
      : 0,
  )

  function close() {
    if (installing) return
    app.updateOffer = null
    error = ''
  }

  // On success the backend starts the installer and quits the app.
  async function install() {
    error = ''
    installing = true
    app.updateProgress = null
    try {
      await api.InstallUpdate()
    } catch (e) {
      error = parseError(e).message
      installing = false
    }
  }

  // Release notes links open in the system browser, never inside the app.
  function onNotesClick(e: MouseEvent) {
    const a = (e.target as HTMLElement).closest('a')
    if (!a) return
    e.preventDefault()
    const href = a.getAttribute('href') ?? ''
    if (isSafeUrl(href)) api.OpenURL(href)
  }
</script>

{#if app.updateOffer}
  {@const offer = app.updateOffer}
  <Modal title="Aggiornamento disponibile" width="560px" onclose={close}>
    <p>
      È disponibile <strong>Zenith {offer.latest}</strong> (installata: {offer.current}).
    </p>

    {#if offer.notes.trim()}
      <!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
      <div class="notes markdown" onclick={onNotesClick}>{@html renderMarkdown(offer.notes)}</div>
    {/if}

    {#if !offer.canInstall}
      <p class="muted">
        La release non contiene un file installabile su questo sistema: scarica la nuova versione dalla pagina delle
        release di GitHub.
      </p>
    {/if}

    {#if installing}
      <div class="progress" role="progressbar" aria-valuenow={percent} aria-valuemin="0" aria-valuemax="100">
        <div class="bar" style:width="{percent}%"></div>
      </div>
      <p class="muted">
        {percent < 100 ? `Download in corso… ${percent}%` : 'Avvio dell’installazione…'}
      </p>
    {/if}

    {#if error}<p class="error-text">{error}</p>{/if}

    {#snippet actions()}
      <button onclick={close} disabled={installing}>Più tardi</button>
      {#if offer.canInstall}
        <button class="primary" onclick={install} disabled={installing}>Scarica e installa</button>
      {/if}
    {/snippet}
  </Modal>
{/if}

<style>
  p {
    margin: 0 0 10px;
    line-height: 1.5;
  }

  .notes {
    max-height: 40vh;
    overflow: auto;
    border: 1px solid var(--border);
    border-radius: 6px;
    padding: 10px 12px;
    margin-bottom: 10px;
  }

  .progress {
    height: 8px;
    border-radius: 4px;
    background: var(--surface-2);
    overflow: hidden;
    margin: 6px 0;
  }

  .bar {
    height: 100%;
    background: var(--accent);
    transition: width 0.2s;
  }
</style>
