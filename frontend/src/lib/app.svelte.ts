// Global app state (Svelte 5 runes) and wrappers around the Go bindings.
import * as api from '../../wailsjs/go/main/App'
import { EventsOn } from '../../wailsjs/runtime/runtime'
import type { Board, InitState, SaveStatus, Settings, UpdateInfo } from './types'

/** Splits a Go error ("ERR_CODE: message") into code and message. */
export function parseError(e: unknown): { code: string; message: string } {
  const text = String((e as any)?.message ?? e)
  const m = /^(ERR_[A-Z_]+): (.*)$/s.exec(text)
  return m ? { code: m[1], message: m[2] } : { code: '', message: text }
}

class AppState {
  board = $state<Board | null>(null)
  settings = $state<Settings>({ dataPath: '', autosaveSeconds: 2, theme: 'system', checkUpdates: true })
  loadError = $state('')
  selectedProjectId = $state<string | null>(null)
  view = $state<'board' | 'settings'>('board')
  saveStatus = $state<SaveStatus>('saved')
  saveError = $state('')
  /** External change of the data file; conflict=true when local changes are pending. */
  externalChange = $state<{ conflict: boolean } | null>(null)
  closeBlocked = $state<{ reason: 'conflict' | 'error'; error: string } | null>(null)
  /** Installed version ("dev" for development builds). */
  version = $state('')
  /** Update offered to the user (shown by UpdateDialog). */
  updateOffer = $state<UpdateInfo | null>(null)
  updateProgress = $state<{ done: number; total: number } | null>(null)

  // Monotonic for the whole session: the backend drops stale UpdateBoard calls.
  #seq = 0
  // Highest seq rejected / accepted by the backend: if a rejection is the most
  // recent outcome, the backend does not hold the board shown on screen.
  #lastRejected = 0
  #lastAccepted = 0

  get selectedProject() {
    return this.board?.projects.find((p) => p.id === this.selectedProjectId) ?? null
  }

  async init() {
    EventsOn('save:status', (e: { status: SaveStatus; error: string }) => {
      this.saveStatus = e.status
      this.saveError = e.error
    })
    EventsOn('file:external-change', (e: { conflict: boolean }) => {
      this.externalChange = { conflict: e.conflict }
    })
    EventsOn('app:close-blocked', (e: { reason: 'conflict' | 'error'; error: string }) => {
      this.closeBlocked = e
    })
    EventsOn('update:progress', (e: { done: number; total: number }) => {
      this.updateProgress = e
    })
    this.applyState((await api.Init()) as unknown as InitState)
    this.version = await api.GetVersion()
    if (this.settings.checkUpdates && this.version !== 'dev') this.#checkUpdatesInBackground()
  }

  // Startup check: silent when offline, when there is no release or no update.
  async #checkUpdatesInBackground() {
    try {
      const info = (await api.CheckUpdate()) as unknown as UpdateInfo
      if (info.available) this.updateOffer = info
    } catch {
      // ignore: the manual check in Settings shows errors
    }
  }

  /** Applies a state returned by Init / SetDataPath / CreateNewFile. */
  applyState(s: InitState) {
    this.settings = s.settings
    this.loadError = s.loadError
    this.externalChange = null
    this.setBoard(s.board ?? null)
    this.saveStatus = 'saved'
    this.saveError = ''
  }

  /** Replaces the board (e.g. after a reload) keeping the selection valid. */
  setBoard(b: Board | null) {
    this.board = b
    if (!b || !b.projects.some((p) => p.id === this.selectedProjectId)) {
      this.selectedProjectId = b?.projects[0]?.id ?? null
    }
  }

  /**
   * Applies a pure mutation and sends the new board to the backend, which
   * saves it after the autosave delay. Errors thrown by fn propagate.
   */
  mutate(fn: (b: Board) => Board) {
    if (!this.board) return
    const next = fn($state.snapshot(this.board) as Board)
    this.board = next
    this.#send(next).catch(() => {}) // error already shown by #send
  }

  async #send(b: Board) {
    const seq = ++this.#seq
    try {
      await api.UpdateBoard(seq, $state.snapshot(b) as any)
      this.#lastAccepted = Math.max(this.#lastAccepted, seq)
    } catch (e) {
      this.#lastRejected = Math.max(this.#lastRejected, seq)
      this.saveStatus = 'error'
      this.saveError = parseError(e).message
      throw e
    }
  }

  /**
   * Manual save (e.g. retry after a save error). If the backend rejected the
   * last board it is sent again first, so what gets saved is what is shown.
   * Status updates arrive via save:status.
   */
  async saveNow() {
    try {
      if (this.board && this.#lastRejected > this.#lastAccepted) await this.#send(this.board)
      await api.SaveNow()
    } catch (e) {
      this.saveStatus = 'error'
      this.saveError = parseError(e).message
    }
  }

  async reload() {
    const b = (await api.Reload()) as unknown as Board
    this.setBoard(b)
    this.externalChange = null
  }

  async resolveConflict(choice: 'reload' | 'keep') {
    const b = (await api.ResolveConflict(choice)) as unknown as Board | null
    if (b) this.setBoard(b)
    this.externalChange = null
  }

  async saveSettings(s: Settings) {
    this.settings = (await api.SaveSettings(s as any)) as unknown as Settings
  }
}

export const app = new AppState()
