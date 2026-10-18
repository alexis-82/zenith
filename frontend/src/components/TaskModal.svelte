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
  // Only used for a new task; existing tasks move by drag & drop.
  let columnId = $state(untrack(() => app.board?.columns[0]?.id ?? ''))
  let mode = $state<'edit' | 'preview'>(initial?.description ? 'preview' : 'edit')
  let error = $state('')

  const html = $derived(renderMarkdown(description))

  function save(e?: SubmitEvent) {
    e?.preventDefault()
    try {
      app.mutate((b) =>
        task ? updateTask(b, projectId, task.id, title, description) : addTask(b, projectId, columnId, title, description),
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

  // Images are embedded as data URLs: the data file is shared between
  // computers, so a local file path would not work elsewhere.
  const MAX_IMAGE_BYTES = 2 * 1024 * 1024
  let textarea = $state<HTMLTextAreaElement>()
  let imageInput = $state<HTMLInputElement>()

  function onImageSelected(e: Event) {
    const input = e.currentTarget as HTMLInputElement
    const file = input.files?.[0]
    input.value = '' // allow picking the same file again
    if (!file) return
    if (!file.type.startsWith('image/')) {
      error = 'Il file scelto non è un\'immagine'
      return
    }
    if (file.size > MAX_IMAGE_BYTES) {
      error = 'Immagine troppo grande (massimo 2 MB)'
      return
    }
    const reader = new FileReader()
    reader.onload = () => insertImage(file.name, reader.result as string)
    reader.onerror = () => (error = 'Impossibile leggere l\'immagine')
    reader.readAsDataURL(file)
  }

  // Inserts a reference-style image at the cursor (![name][img-N]) and puts
  // the long data URL in a definition at the end, keeping the text readable.
  function insertImage(fileName: string, dataUrl: string) {
    let n = 1
    while (description.includes(`[img-${n}]:`)) n++
    const ref = `img-${n}`
    const alt = fileName.replace(/\.[^.]+$/, '').replace(/[[\]]/g, '')
    const tag = `![${alt}][${ref}]`

    let pos = textarea?.selectionStart ?? description.length
    let insert = tag
    const defsStart = description.search(/\n\[img-\d+\]: data:/)
    if (defsStart >= 0 && pos > defsStart) {
      // Never inside the definitions: put the image on a new line at the end
      // of the text, keeping the blank line a definition needs before it.
      pos = defsStart
      while (pos > 0 && /\s/.test(description[pos - 1])) pos--
      insert = '\n' + tag
    }

    const text = description.slice(0, pos) + insert + description.slice(pos)
    description = text.replace(/\s+$/, '') + `\n\n[${ref}]: ${dataUrl}\n`
    error = ''
  }
</script>

<Modal title={task ? 'Modifica task' : 'Nuovo task'} width="640px" {onclose}>
  <form id="task-form" onsubmit={save}>
    {#if !task}
      <label>
        <span class="muted">Colonna</span>
        <select bind:value={columnId}>
          {#each app.board?.columns ?? [] as column (column.id)}
            <option value={column.id}>{column.name}</option>
          {/each}
        </select>
      </label>
    {/if}

    <label>
      <span class="muted">Titolo</span>
      <input bind:value={title} use:focus />
    </label>

    <div class="desc-header">
      <span class="muted">Descrizione (Markdown)</span>
      {#if mode === 'edit'}
        <button type="button" class="add-image" onclick={() => imageInput?.click()}>🖼 Aggiungi immagine</button>
        <input
          bind:this={imageInput}
          type="file"
          accept="image/png,image/jpeg,image/gif,image/webp"
          hidden
          onchange={onImageSelected}
        />
      {/if}
      <div class="tabs">
        <button type="button" class:on={mode === 'edit'} onclick={() => (mode = 'edit')}>Modifica</button>
        <button type="button" class:on={mode === 'preview'} onclick={() => (mode = 'preview')}>Anteprima</button>
      </div>
    </div>

    {#if mode === 'edit'}
      <textarea bind:this={textarea} bind:value={description} rows="12" placeholder="Testo, **grassetto**, - elenchi, [link](https://…)"></textarea>
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

  .add-image {
    margin-left: auto;
    margin-right: 8px;
    padding: 4px 10px;
    font-size: 12px;
  }

  .preview :global(img) {
    max-width: 100%;
    height: auto;
  }

  .delete {
    margin-right: auto;
  }
</style>
