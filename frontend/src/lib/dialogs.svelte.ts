// Promise-based confirm/prompt dialogs (window.confirm/prompt are not
// reliable inside the Wails webview). Rendered by components/Dialogs.svelte.

export interface ConfirmOptions {
  title: string
  message: string
  confirmLabel?: string
  danger?: boolean
}

export interface PromptOptions {
  title: string
  label: string
  value?: string
  confirmLabel?: string
}

type Pending =
  | { kind: 'confirm'; opts: ConfirmOptions; resolve: (ok: boolean) => void }
  | { kind: 'prompt'; opts: PromptOptions; resolve: (value: string | null) => void }

class Dialogs {
  pending = $state<Pending | null>(null)

  confirm(opts: ConfirmOptions): Promise<boolean> {
    this.close(null) // a new dialog cancels the previous one
    return new Promise((resolve) => (this.pending = { kind: 'confirm', opts, resolve }))
  }

  prompt(opts: PromptOptions): Promise<string | null> {
    this.close(null)
    return new Promise((resolve) => (this.pending = { kind: 'prompt', opts, resolve }))
  }

  /** true/string = confirmed, false/null = cancelled. */
  close(result: boolean | string | null) {
    const p = this.pending
    this.pending = null
    if (p?.kind === 'confirm') p.resolve(result === true)
    else if (p?.kind === 'prompt') p.resolve(typeof result === 'string' ? result : null)
  }
}

export const dialogs = new Dialogs()
