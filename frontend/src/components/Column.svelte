<script lang="ts">
  import { dndzone, type DndEvent } from 'svelte-dnd-action'
  import { app } from '../lib/app.svelte'
  import { applyColumnOrder, countTasksInColumn, deleteColumn, moveColumn, renameColumn, tasksInColumn } from '../lib/board'
  import { dialogs } from '../lib/dialogs.svelte'
  import type { Column, Project, Task } from '../lib/types'
  import TaskCard from './TaskCard.svelte'

  let {
    project,
    column,
    index,
    total,
    onopen,
  }: { project: Project; column: Column; index: number; total: number; onopen: (t: Task) => void } = $props()

  // Local copy required by svelte-dnd-action; resynced whenever the board changes.
  let items = $state<Task[]>([])
  $effect(() => {
    items = tasksInColumn(project, column.id)
  })

  function consider(e: CustomEvent<DndEvent<Task>>) {
    items = e.detail.items
  }

  function finalize(e: CustomEvent<DndEvent<Task>>) {
    items = e.detail.items
    const ids = items.map((t) => t.id)
    app.mutate((b) => applyColumnOrder(b, project.id, column.id, ids))
  }

  async function rename() {
    const name = await dialogs.prompt({ title: 'Rinomina colonna', label: 'Nome della colonna', value: column.name })
    if (name !== null) app.mutate((b) => renameColumn(b, column.id, name))
  }

  async function remove() {
    if (!app.board) return
    const n = countTasksInColumn(app.board, column.id)
    const ok = await dialogs.confirm({
      title: 'Elimina colonna',
      message:
        `Eliminare la colonna "${column.name}"? La colonna sparirà da tutti i progetti.` +
        (n ? `\nVerranno eliminati ${n} task in tutti i progetti.` : ''),
      confirmLabel: 'Elimina',
      danger: true,
    })
    if (ok) app.mutate((b) => deleteColumn(b, column.id).board)
  }
</script>

<section class="column">
  <header>
    <span class="name" title={column.name}>{column.name}</span>
    <span class="count">{items.length}</span>
    <span class="actions">
      <button class="icon" title="Sposta a sinistra" aria-label="Sposta a sinistra" disabled={index === 0} onclick={() => app.mutate((b) => moveColumn(b, column.id, -1))}>←</button>
      <button class="icon" title="Sposta a destra" aria-label="Sposta a destra" disabled={index === total - 1} onclick={() => app.mutate((b) => moveColumn(b, column.id, 1))}>→</button>
      <button class="icon" title="Rinomina" aria-label="Rinomina colonna" onclick={rename}>✎</button>
      <button class="icon" title={total === 1 ? "Non puoi eliminare l'ultima colonna" : 'Elimina'} aria-label="Elimina colonna" disabled={total === 1} onclick={remove}>🗑</button>
    </span>
  </header>

  <div
    class="tasks"
    use:dndzone={{ items, type: 'task', flipDurationMs: 150, dropTargetStyle: {} }}
    onconsider={consider}
    onfinalize={finalize}
  >
    {#each items as task (task.id)}
      <div class="item">
        <TaskCard {task} onopen={() => onopen(task)} />
      </div>
    {/each}
  </div>
</section>

<style>
  .column {
    width: 280px;
    flex-shrink: 0;
    display: flex;
    flex-direction: column;
    max-height: 100%;
    background: var(--column);
    border-radius: 10px;
  }

  header {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 10px 10px 6px 12px;
    font-weight: 600;
  }

  .name {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .count {
    font-size: 12px;
    font-weight: 400;
    color: var(--text-muted);
  }

  .actions {
    margin-left: auto;
    display: flex;
    visibility: hidden;
  }

  .column:hover .actions {
    visibility: visible;
  }

  .tasks {
    flex: 1;
    min-height: 60px;
    overflow-y: auto;
    display: flex;
    flex-direction: column;
    gap: 8px;
    padding: 4px 10px 12px;
  }

  .tasks:global([aria-dropeffect]) {
    outline: none;
  }
</style>
