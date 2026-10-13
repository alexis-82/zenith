<script lang="ts">
  import { untrack } from 'svelte'
  import * as api from '../../wailsjs/go/main/App'
  import { app } from '../lib/app.svelte'
  import { addTask, deleteTask, updateTask } from '../lib/board'
  import { dialogs } from '../lib/dialogs.svelte'
  import { isSafeUrl, renderMarkdown } from '../lib/markdown'
  import type { Task } from '../lib/types'
  import Modal from './Modal.svelte'

  let { projectId, task, onclose }: { projectId: string; task: Task | null; onclose: () => void } = $props()

  // The form edits a copy of the task as it was when the modal opened.
  const initial = untrack(() => task)
  let title = $state(initial?.title ?? '')
  let description = $state(initial?.description ?? '')
  let mode = $state<'edit' | 'preview'>(initial?.description ? 'preview' : 'edit')
  let error = $state('')

  const html = $derived(renderMarkdown(description))

  function save(e?: SubmitEvent) {
    e?.preventDefault()
    try {
      app.mutate((b) =>
        task ? updateTask(b, projectId, task.id, title, description) : addTask(b, projectId, title, description),
      )
      onclose()
    } catch (err) {
      error = (err as Error).message
    }
  }

  async function remove() {
    if (!task) return
    const ok = await dialogs.confirm({
      title: 'Elimina task',
      message: `Eliminare "${task.title}"?`,
      confirmLabel: 'Elimina',
      danger: true,
    })
    if (!ok) return
    app.mutate((b) => deleteTask(b, projectId, task.id))
    onclose()
  }

  // Links must never navigate the webview: open safe ones in the system browser.
  function onPreviewClick(e: MouseEvent) {
    const a = (e.target as HTMLElement).closest('a')
    if (!a) return
    e.preventDefault()
    const href = a.getAttribute('href') ?? ''
    if (isSafeUrl(href)) api.OpenURL(href)
  }

  function focus(node: HTMLElement) {
    node.focus()
  }
</script>

<Modal title={task ? 'Modifica task' : 'Nuovo task'} width="640px" {onclose}>
  <form id="task-form" onsubmit={save}>
    <label>
      <span class="muted">Titolo</span>
      <input bind:value={title} use:focus />
    </label>

    <div class="desc-header">
      <span class="muted">Descrizione (Markdown)</span>
      <div class="tabs">
        <button type="button" class:on={mode === 'edit'} onclick={() => (mode = 'edit')}>Modifica</button>
        <button type="button" class:on={mode === 'preview'} onclick={() => (mode = 'preview')}>Anteprima</button>
      </div>
    </div>

    {#if mode === 'edit'}
      <textarea bind:value={description} rows="12" placeholder="Testo, **grassetto**, - elenchi, [link](https://…)"></textarea>
    {:else}
      <!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
      <div class="preview markdown" onclick={onPreviewClick}>
        {#if description.trim()}
          {@html html}
        {:else}
          <span class="muted">Nessuna descrizione</span>
        {/if}
      </div>
    {/if}

    {#if error}<p class="error-text">{error}</p>{/if}
  </form>

  {#snippet actions()}
    {#if task}
      <button type="button" class="danger delete" onclick={remove}>Elimina</button>
    {/if}
    <button type="button" onclick={onclose}>Annulla</button>
    <button class="primary" type="submit" form="task-form" disabled={!title.trim()}>Salva</button>
  {/snippet}
</Modal>

<style>
  form {
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  label {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .desc-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
  }

  .tabs {
    display: flex;
  }

  .tabs button {
    border-radius: 0;
    padding: 4px 10px;
    font-size: 12px;
  }

  .tabs button:first-child {
    border-radius: 6px 0 0 6px;
  }

  .tabs button:last-child {
    border-radius: 0 6px 6px 0;
    border-left: none;
  }

  .tabs button.on {
    background: var(--surface-2);
    font-weight: 600;
  }

  textarea {
    resize: vertical;
    font-family: ui-monospace, Consolas, monospace;
    font-size: 13px;
    line-height: 1.5;
  }

  .preview {
    min-height: 220px;
    max-height: 50vh;
    overflow: auto;
    border: 1px solid var(--border);
    border-radius: 6px;
    padding: 10px 12px;
  }

  .delete {
    margin-right: auto;
  }
</style>
