// @vitest-environment jsdom
// Behaviour tests of the UI against mocked Wails bindings.
import { fireEvent, render, screen, waitFor, within } from '@testing-library/svelte'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const mocks = vi.hoisted(() => {
  // jsdom has no matchMedia; theme.ts needs it at import time.
  window.matchMedia = ((q: string) => ({
    matches: false,
    media: q,
    addEventListener() {},
    removeEventListener() {},
  })) as any
  const handlers: Record<string, (data: any) => void> = {}
  return {
    handlers,
    api: {
      Init: vi.fn(),
      UpdateBoard: vi.fn(),
      SaveNow: vi.fn(),
      Reload: vi.fn(),
      ResolveConflict: vi.fn(),
      GetSettings: vi.fn(),
      SaveSettings: vi.fn(),
      SetDataPath: vi.fn(),
      CreateNewFile: vi.fn(),
      PickExistingFile: vi.fn(),
      PickNewFile: vi.fn(),
      OpenURL: vi.fn(),
      Quit: vi.fn(),
    },
  }
})

vi.mock('../wailsjs/go/main/App', () => mocks.api)
vi.mock('../wailsjs/runtime/runtime', () => ({
  EventsOn: (name: string, cb: (data: any) => void) => {
    mocks.handlers[name] = cb
    return () => {}
  },
}))

import App from './App.svelte'
import { app } from './lib/app.svelte'
import { dialogs } from './lib/dialogs.svelte'
import type { Board, InitState, Settings } from './lib/types'

const api = mocks.api
const emit = (name: string, data: any) => mocks.handlers[name](data)

const settings = (s: Partial<Settings> = {}): Settings => ({
  dataPath: 'C:\\Nextcloud\\zenith.json',
  autosaveSeconds: 2,
  theme: 'system',
  ...s,
})

function sampleBoard(): Board {
  return {
    version: 1,
    columns: [
      { id: 'c_impl', name: 'Implementare' },
      { id: 'c_doing', name: 'In corso' },
      { id: 'c_done', name: 'Terminato' },
    ],
    projects: [
      {
        id: 'p_1',
        name: 'Alfa',
        tasks: [
          { id: 't_1', title: 'Primo', description: '', columnId: 'c_impl', order: 0 },
          { id: 't_2', title: 'Secondo', description: '', columnId: 'c_impl', order: 1 },
        ],
      },
      {
        id: 'p_2',
        name: 'Beta',
        tasks: [{ id: 't_3', title: 'Terzo', description: '', columnId: 'c_impl', order: 0 }],
      },
    ],
  }
}

/** Board sent by the last UpdateBoard call. */
const lastSent = (): Board => api.UpdateBoard.mock.calls.at(-1)![1]

async function start(state: Partial<InitState> = {}) {
  api.Init.mockResolvedValue({ settings: settings(), board: sampleBoard(), loadError: '', ...state })
  render(App)
  await screen.findByText('Zenith')
}

beforeEach(() => {
  for (const f of Object.values(api)) f.mockReset()
  api.UpdateBoard.mockResolvedValue(undefined)
  api.SaveSettings.mockImplementation(async (s: Settings) => s)
  app.view = 'board'
  app.closeBlocked = null
  app.selectedProjectId = null
  dialogs.close(null)
})

describe('avvio', () => {
  it('primo avvio: schermata vuota che invita ad aprire le Impostazioni', async () => {
    await start({ settings: settings({ dataPath: '' }), board: null })
    expect(screen.getByText('Nessun file dati configurato')).toBeTruthy()
    expect(screen.queryByText('Progetti')).toBeNull()
    await userEvent.click(screen.getByRole('button', { name: 'Apri Impostazioni' }))
    expect(screen.getByRole('heading', { name: 'Impostazioni' })).toBeTruthy()
  })

  it('file illeggibile: mostra l’errore di caricamento', async () => {
    await start({ board: null, loadError: 'Il file dati "x" non è leggibile' })
    expect(screen.getByText('Impossibile aprire il file dati')).toBeTruthy()
    expect(screen.getByText(/non è leggibile/)).toBeTruthy()
  })
})

describe('progetti', () => {
  it('crea un progetto e lo invia al backend', async () => {
    await start({ board: { ...sampleBoard(), projects: [] } })
    await userEvent.click(screen.getByRole('button', { name: '+ Nuovo progetto' }))
    await userEvent.type(screen.getByRole('textbox'), 'Test')
    await userEvent.click(screen.getByRole('button', { name: 'Crea' }))

    expect(lastSent().projects.map((p) => p.name)).toEqual(['Test'])
    expect(screen.getByRole('heading', { name: 'Test' })).toBeTruthy()
  })

  it('rinomina un progetto', async () => {
    await start()
    await fireEvent.click(screen.getAllByLabelText('Rinomina progetto')[0])
    const input = screen.getByRole('textbox') as HTMLInputElement
    expect(input.value).toBe('Alfa')
    await userEvent.clear(input)
    await userEvent.type(input, 'Test2')
    await userEvent.click(within(screen.getByRole('dialog')).getByRole('button', { name: 'Salva' }))
    expect(lastSent().projects[0].name).toBe('Test2')
  })

  it('elimina un progetto con conferma che indica i task', async () => {
    await start()
    await fireEvent.click(screen.getAllByLabelText('Elimina progetto')[0])
    expect(screen.getByText(/Verranno eliminati anche 2 task/)).toBeTruthy()

    await userEvent.click(screen.getByRole('button', { name: 'Annulla' }))
    expect(api.UpdateBoard).not.toHaveBeenCalled()

    await fireEvent.click(screen.getAllByLabelText('Elimina progetto')[0])
    await userEvent.click(screen.getByRole('button', { name: 'Elimina' }))
    expect(lastSent().projects.map((p) => p.id)).toEqual(['p_2'])
  })
})

describe('colonne', () => {
  it('aggiunge una colonna e la sposta', async () => {
    await start()
    await userEvent.click(screen.getByRole('button', { name: '+ Aggiungi colonna' }))
    await userEvent.type(screen.getByRole('textbox'), 'Revisione')
    await userEvent.click(screen.getByRole('button', { name: 'Aggiungi' }))
    expect(lastSent().columns.map((c) => c.name)).toEqual(['Implementare', 'In corso', 'Terminato', 'Revisione'])

    const left = screen.getAllByLabelText('Sposta a sinistra')
    await fireEvent.click(left[3])
    expect(lastSent().columns.map((c) => c.name)).toEqual(['Implementare', 'In corso', 'Revisione', 'Terminato'])
  })

  it('elimina una colonna e i suoi task in tutti i progetti', async () => {
    await start()
    await fireEvent.click(screen.getAllByLabelText('Elimina colonna')[0])
    expect(screen.getByText(/Verranno eliminati 3 task in tutti i progetti/)).toBeTruthy()
    await userEvent.click(screen.getByRole('button', { name: 'Elimina' }))
    const b = lastSent()
    expect(b.columns.map((c) => c.id)).toEqual(['c_doing', 'c_done'])
    expect(b.projects.every((p) => p.tasks.length === 0)).toBe(true)
  })

  it("l'ultima colonna non è eliminabile", async () => {
    const board = sampleBoard()
    board.columns = [board.columns[0]]
    await start({ board })
    expect((screen.getByLabelText('Elimina colonna') as HTMLButtonElement).disabled).toBe(true)
  })
})

describe('task', () => {
  it('crea un task con descrizione in fondo alla prima colonna', async () => {
    await start()
    await userEvent.click(screen.getByRole('button', { name: '+ Nuovo task' }))
    const dialog = screen.getByRole('dialog', { name: 'Nuovo task' })
    const [title, desc] = [within(dialog).getAllByRole('textbox')[0], within(dialog).getAllByRole('textbox')[1]]
    await userEvent.type(title, 'Nuovo')
    await userEvent.type(desc, '**importante**')
    await userEvent.click(within(dialog).getByRole('button', { name: 'Salva' }))

    const task = lastSent().projects[0].tasks.find((t) => t.title === 'Nuovo')!
    expect(task).toMatchObject({ columnId: 'c_impl', order: 2, description: '**importante**' })
  })

  it('modifica ed elimina un task (con conferma)', async () => {
    await start()
    await userEvent.click(screen.getByRole('button', { name: /Primo/ }))
    const dialog = screen.getByRole('dialog', { name: 'Modifica task' })
    const title = within(dialog).getAllByRole('textbox')[0]
    await userEvent.clear(title)
    await userEvent.type(title, 'Rinominato')
    await userEvent.click(within(dialog).getByRole('button', { name: 'Salva' }))
    expect(lastSent().projects[0].tasks[0].title).toBe('Rinominato')

    await userEvent.click(screen.getByRole('button', { name: /Rinominato/ }))
    await userEvent.click(within(screen.getByRole('dialog', { name: 'Modifica task' })).getByRole('button', { name: 'Elimina' }))
    expect(screen.getByText('Eliminare "Rinominato"?')).toBeTruthy()
    await userEvent.click(within(screen.getByRole('dialog', { name: 'Elimina task' })).getByRole('button', { name: 'Elimina' }))
    expect(lastSent().projects[0].tasks.map((t) => t.id)).toEqual(['t_2'])
  })

  it('anteprima Markdown sicura: formatta, non esegue HTML, apre solo link sicuri', async () => {
    const board = sampleBoard()
    board.projects[0].tasks[0].description =
      '**grassetto**\n\n[sito](https://example.com) [calc](file:///C:/Windows/System32/calc.exe)\n\n<img src=x onerror="window.__xss=1">'
    await start({ board })
    await userEvent.click(screen.getByRole('button', { name: /Primo/ }))
    const dialog = screen.getByRole('dialog', { name: 'Modifica task' })

    expect(dialog.querySelector('strong')?.textContent).toBe('grassetto')
    expect(dialog.querySelector('[onerror]')).toBeNull()
    expect((window as any).__xss).toBeUndefined()

    await fireEvent.click(within(dialog).getByText('sito'))
    expect(api.OpenURL).toHaveBeenCalledWith('https://example.com')
    const calc = within(dialog).queryByText('calc')
    if (calc) await fireEvent.click(calc)
    expect(api.OpenURL).toHaveBeenCalledTimes(1)
  })

  it('drag & drop: lo spostamento tra colonne arriva al backend con seq crescente', async () => {
    await start()
    const zones = document.querySelectorAll('.tasks')
    const [impl, doing] = [zones[0], zones[1]]
    const t1 = sampleBoard().projects[0].tasks[0]
    const t2 = sampleBoard().projects[0].tasks[1]
    // svelte-dnd-action fires finalize on both the source and the target zone
    await fireEvent(impl, new CustomEvent('finalize', { detail: { items: [t2], info: {} } }))
    await fireEvent(doing, new CustomEvent('finalize', { detail: { items: [t1], info: {} } }))

    const tasks = lastSent().projects[0].tasks
    expect(tasks.find((t) => t.id === 't_1')).toMatchObject({ columnId: 'c_doing', order: 0 })
    expect(tasks.find((t) => t.id === 't_2')).toMatchObject({ columnId: 'c_impl', order: 0 })

    const seqs = api.UpdateBoard.mock.calls.map((c) => c[0])
    expect(seqs).toEqual([...seqs].sort((a, b) => a - b))
    expect(new Set(seqs).size).toBe(seqs.length)
  })

  it('drag & drop: riordino su/giù nella stessa colonna', async () => {
    await start()
    const impl = document.querySelectorAll('.tasks')[0]
    const [t1, t2] = sampleBoard().projects[0].tasks
    await fireEvent(impl, new CustomEvent('finalize', { detail: { items: [t2, t1], info: {} } }))
    const tasks = lastSent().projects[0].tasks
    expect(tasks.find((t) => t.id === 't_2')!.order).toBe(0)
    expect(tasks.find((t) => t.id === 't_1')!.order).toBe(1)
  })
})

describe('impostazioni', () => {
  async function openSettings() {
    // sidebar button (the empty state also has "Apri Impostazioni")
    const btn = screen.getAllByRole('button', { name: /Impostazioni/ }).find((b) => !b.textContent!.includes('Apri'))!
    await userEvent.click(btn)
  }

  it('percorso inesistente con dati caricati: chiede conferma e poi salva lì', async () => {
    await start()
    await openSettings()
    api.SetDataPath.mockRejectedValueOnce('ERR_NEEDS_CONFIRM: il file non esiste').mockResolvedValueOnce({
      settings: settings({ dataPath: 'D:\\nuovo.json' }),
      board: sampleBoard(),
      loadError: '',
    })
    const input = screen.getByPlaceholderText(/Nextcloud/)
    await userEvent.clear(input)
    await userEvent.type(input, 'D:\\nuovo.json')
    await userEvent.click(screen.getByRole('button', { name: 'Applica' }))

    await screen.findByText('Il file non esiste')
    await userEvent.click(screen.getByRole('button', { name: 'Salva qui' }))
    await screen.findByText('Dati salvati nel nuovo file.')
    expect(api.SetDataPath.mock.calls).toEqual([
      ['D:\\nuovo.json', false],
      ['D:\\nuovo.json', true],
    ])
  })

  it('percorso inesistente ad app vergine: errore che suggerisce di attendere Nextcloud', async () => {
    await start({ board: null, settings: settings({ dataPath: '' }) })
    await openSettings()
    api.SetDataPath.mockRejectedValueOnce('ERR_FILE_NOT_FOUND: il file non esiste')
    await userEvent.type(screen.getByPlaceholderText(/Nextcloud/), 'D:\\zenith.json')
    await userEvent.click(screen.getByRole('button', { name: 'Applica' }))
    expect(await screen.findByText(/Nextcloud sta ancora sincronizzando/)).toBeTruthy()
    expect(api.SetDataPath).toHaveBeenCalledTimes(1)
  })

  it('crea nuovo file: carica la board vuota', async () => {
    await start({ board: null, settings: settings({ dataPath: '' }) })
    await openSettings()
    api.PickNewFile.mockResolvedValue('D:\\zenith.json')
    api.CreateNewFile.mockResolvedValue({
      settings: settings({ dataPath: 'D:\\zenith.json' }),
      board: { ...sampleBoard(), projects: [] },
      loadError: '',
    })
    await userEvent.click(screen.getByRole('button', { name: 'Crea nuovo file…' }))
    await screen.findByText('Nuovo file creato.')
    expect(api.CreateNewFile).toHaveBeenCalledWith('D:\\zenith.json')
    expect(screen.getByText('Progetti')).toBeTruthy()
  })

  it('tema e intervallo vengono salvati e il tema si applica subito', async () => {
    await start()
    await openSettings()
    await userEvent.click(screen.getByLabelText('Scuro'))
    expect(document.documentElement.dataset.theme).toBe('dark')
    expect(api.SaveSettings).toHaveBeenLastCalledWith(expect.objectContaining({ theme: 'dark' }))

    const n = screen.getByRole('spinbutton')
    await userEvent.clear(n)
    await userEvent.type(n, '5')
    await fireEvent.change(n)
    await waitFor(() =>
      expect(api.SaveSettings).toHaveBeenLastCalledWith(expect.objectContaining({ autosaveSeconds: 5 })),
    )
  })
})

describe('salvataggio, modifiche esterne, uscita', () => {
  it("l'indicatore segue gli eventi di salvataggio", async () => {
    await start()
    expect(screen.getByText('Salvato')).toBeTruthy()
    emit('save:status', { status: 'dirty', error: '' })
    expect(await screen.findByText('Modifiche non salvate')).toBeTruthy()
    emit('save:status', { status: 'paused', error: '' })
    expect(await screen.findByText('In pausa: conflitto')).toBeTruthy()
  })

  it('pulsante Salva: attivo solo con modifiche pendenti o errori, salva a mano', async () => {
    await start()
    const save = () => screen.getByRole('button', { name: 'Salva' }) as HTMLButtonElement
    expect(save().disabled).toBe(true)

    emit('save:status', { status: 'error', error: 'accesso negato' })
    expect(await screen.findByText('accesso negato')).toBeTruthy()
    expect(save().disabled).toBe(false)
    api.SaveNow.mockResolvedValueOnce(undefined)
    await userEvent.click(save())
    expect(api.SaveNow).toHaveBeenCalledTimes(1)

    api.SaveNow.mockRejectedValueOnce('disco pieno')
    await userEvent.click(save())
    expect(await screen.findByText('disco pieno')).toBeTruthy()

    emit('save:status', { status: 'paused', error: '' })
    await screen.findByText('In pausa: conflitto')
    expect(save().disabled).toBe(true)
  })

  it('modifica esterna senza modifiche pendenti: banner e Ricarica', async () => {
    await start()
    const external = sampleBoard()
    external.projects[0].name = 'Da altro PC'
    api.Reload.mockResolvedValue(external)
    emit('file:external-change', { conflict: false })
    await userEvent.click(await screen.findByRole('button', { name: 'Ricarica' }))
    expect(await screen.findByRole('heading', { name: 'Da altro PC' })).toBeTruthy()
    expect(screen.queryByText(/modificato da un altro dispositivo/)).toBeNull()
  })

  it('conflitto: entrambe le scelte chiamano il backend', async () => {
    await start()
    api.ResolveConflict.mockResolvedValue(null)
    emit('file:external-change', { conflict: true })
    await userEvent.click(await screen.findByRole('button', { name: /Mantieni le mie/ }))
    expect(api.ResolveConflict).toHaveBeenCalledWith('keep')
    expect(screen.queryByRole('dialog')).toBeNull()

    api.ResolveConflict.mockResolvedValue(sampleBoard())
    emit('file:external-change', { conflict: true })
    await userEvent.click(await screen.findByRole('button', { name: /Ricarica \(perdi/ }))
    expect(api.ResolveConflict).toHaveBeenLastCalledWith('reload')
  })

  it('reload fallito nel conflitto: errore visibile e conflitto ancora aperto', async () => {
    await start()
    api.ResolveConflict.mockRejectedValueOnce('ERR_MALFORMED: file JSON malformato')
    emit('file:external-change', { conflict: true })
    await userEvent.click(await screen.findByRole('button', { name: /Ricarica \(perdi/ }))
    expect(await screen.findByText(/file JSON malformato/)).toBeTruthy()
    expect(screen.getByRole('button', { name: /Mantieni le mie/ })).toBeTruthy()
  })

  it('uscita bloccata per errore: Riprova / Esci senza salvare', async () => {
    await start()
    emit('app:close-blocked', { reason: 'error', error: 'disco pieno' })
    expect(await screen.findByText('disco pieno')).toBeTruthy()
    await userEvent.click(screen.getByRole('button', { name: 'Esci senza salvare' }))
    expect(api.Quit).toHaveBeenLastCalledWith(true)
    await userEvent.click(screen.getByRole('button', { name: 'Riprova' }))
    expect(api.Quit).toHaveBeenLastCalledWith(false)
  })

  it('uscita bloccata per conflitto: Mantieni le mie e esci', async () => {
    await start()
    api.ResolveConflict.mockResolvedValue(null)
    emit('app:close-blocked', { reason: 'conflict', error: '' })
    await userEvent.click(await screen.findByRole('button', { name: 'Mantieni le mie e esci' }))
    await waitFor(() => expect(api.Quit).toHaveBeenCalledWith(false))
    expect(api.ResolveConflict).toHaveBeenCalledWith('keep')
  })
})
