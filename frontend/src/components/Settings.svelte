<script lang="ts">
  import * as api from '../../wailsjs/go/main/App'
  import { app, parseError } from '../lib/app.svelte'
  import { dialogs } from '../lib/dialogs.svelte'
  import { applyTheme } from '../lib/theme'
  import type { InitState, Theme } from '../lib/types'

  let path = $state(app.settings.dataPath)
  let autosave = $state(app.settings.autosaveSeconds)
  let error = $state('')
  let info = $state('')
  let busy = $state(false)

  function done(s: InitState, message: string) {
    app.applyState(s)
    path = s.settings.dataPath
    info = message
  }

  async function run(fn: () => Promise<void>) {
    error = ''
    info = ''
    busy = true
    try {
      await fn()
    } finally {
      busy = false
    }
  }

  async function setPath(p: string, confirmWrite = false): Promise<void> {
    try {
      const s = (await api.SetDataPath(p, confirmWrite)) as unknown as InitState
      done(s, confirmWrite ? 'Dati salvati nel nuovo file.' : 'File caricato.')
    } catch (e) {
      const { code, message } = parseError(e)
      switch (code) {
        case 'ERR_NEEDS_CONFIRM': {
          const ok = await dialogs.confirm({
            title: 'Il file non esiste',
            message: `"${p}" non esiste.\nVuoi salvare qui i dati attuali e usare questo file d'ora in poi?`,
            confirmLabel: 'Salva qui',
          })
          if (ok) await setPath(p, true)
          return
        }
        case 'ERR_FILE_NOT_FOUND':
          error =
            'Il file non esiste. Se Nextcloud sta ancora sincronizzando attendi e riprova; per partire da zero usa "Crea nuovo file…".'
          return
        case 'ERR_UNSAVED_CHANGES':
          error =
            'Ci sono modifiche non salvate (conflitto aperto o salvataggio fallito). Risolvi prima il conflitto, oppure indica un file nuovo in cui salvarle.'
          return
        default:
          error = message
      }
    }
  }

  const apply = () => run(() => setPath(path.trim()))

  const browse = () =>
    run(async () => {
      const p = await api.PickExistingFile()
      if (p) await setPath(p)
    })

  const createNew = () =>
    run(async () => {
      const p = await api.PickNewFile()
      if (!p) return
      try {
        done((await api.CreateNewFile(p)) as unknown as InitState, 'Nuovo file creato.')
      } catch (e) {
        const { code, message } = parseError(e)
        if (code === 'ERR_FILE_EXISTS') error = 'Il file esiste già: usa "Sfoglia…" per aprirlo.'
        else if (code === 'ERR_UNSAVED_CHANGES') error = 'Ci sono modifiche non salvate: risolvile prima di cambiare file.'
        else error = message
      }
    })

  async function saveAutosave() {
    const n = Math.round(Number(autosave))
    if (!Number.isFinite(n)) return
    try {
      await app.saveSettings({ ...app.settings, autosaveSeconds: n })
      autosave = app.settings.autosaveSeconds
    } catch (e) {
      error = parseError(e).message
    }
  }

  async function setTheme(theme: Theme) {
    applyTheme(theme)
    try {
      await app.saveSettings({ ...app.settings, theme })
    } catch (e) {
      error = parseError(e).message
    }
  }
</script>

<div class="settings">
  <header>
    <h1>Impostazioni</h1>
    <button onclick={() => (app.view = 'board')}>Chiudi</button>
  </header>

  <section>
    <h2>File dati</h2>
    <p class="muted">
      Tutti i progetti e i task stanno in un unico file JSON. Mettilo nella cartella sincronizzata da Nextcloud per vederlo
      da tutti i computer.
    </p>
    <form
      class="row"
      onsubmit={(e) => {
        e.preventDefault()
        apply()
      }}
    >
      <input class="path" bind:value={path} placeholder="es. C:\Users\…\Nextcloud\zenith.json" spellcheck="false" />
      <button class="primary" type="submit" disabled={busy || !path.trim()}>Applica</button>
    </form>
    <div class="row">
      <button onclick={browse} disabled={busy}>Sfoglia…</button>
      <button onclick={createNew} disabled={busy}>Crea nuovo file…</button>
    </div>
    {#if error}<p class="error-text">{error}</p>{/if}
    {#if info}<p class="ok">{info}</p>{/if}
  </section>

  <section>
    <h2>Salvataggio automatico</h2>
    <label class="row">
      <span>Salva</span>
      <input type="number" min="1" max="300" bind:value={autosave} onchange={saveAutosave} />
      <span>secondi dopo l'ultima modifica</span>
    </label>
  </section>

  <section>
    <h2>Tema</h2>
    <div class="row">
      {#each [['system', 'Sistema'], ['light', 'Chiaro'], ['dark', 'Scuro']] as [value, label] (value)}
        <label class="radio">
          <input
            type="radio"
            name="theme"
            {value}
            checked={app.settings.theme === value}
            onchange={() => setTheme(value as Theme)}
          />
          {label}
        </label>
      {/each}
    </div>
  </section>
</div>

<style>
  .settings {
    max-width: 720px;
    padding: 24px 32px;
  }

  header {
    display: flex;
    align-items: center;
    justify-content: space-between;
  }

  h1 {
    font-size: 20px;
    margin: 0;
  }

  section {
    margin-top: 28px;
  }

  h2 {
    font-size: 15px;
    margin: 0 0 8px;
  }

  p {
    line-height: 1.5;
  }

  .row {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-top: 8px;
  }

  .path {
    flex: 1;
    min-width: 0;
  }

  input[type='number'] {
    width: 80px;
  }

  .radio {
    display: flex;
    align-items: center;
    gap: 4px;
    margin-right: 12px;
  }

  .ok {
    color: #3aa55d;
  }
</style>
