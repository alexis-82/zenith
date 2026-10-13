<script lang="ts">
  import { app } from '../lib/app.svelte'
  import { addColumn } from '../lib/board'
  import { dialogs } from '../lib/dialogs.svelte'
  import type { Project, Task } from '../lib/types'
  import Column from './Column.svelte'
  import TaskModal from './TaskModal.svelte'

  let { project }: { project: Project } = $props()

  // undefined = closed, null = new task
  let editing = $state<Task | null | undefined>(undefined)

  async function newColumn() {
    const name = await dialogs.prompt({ title: 'Nuova colonna', label: 'Nome della colonna', confirmLabel: 'Aggiungi' })
    if (name !== null) app.mutate((b) => addColumn(b, name))
  }
</script>

<div class="board-view">
  <header>
    <h1 title={project.name}>{project.name}</h1>
    <button class="primary" onclick={() => (editing = null)}>+ Nuovo task</button>
  </header>

  <div class="columns">
    {#each app.board?.columns ?? [] as column, i (column.id)}
      <Column {project} {column} index={i} total={app.board?.columns.length ?? 0} onopen={(t) => (editing = t)} />
    {/each}
    <button class="add-column" onclick={newColumn}>+ Aggiungi colonna</button>
  </div>
</div>

{#if editing !== undefined}
  <TaskModal projectId={project.id} task={editing} onclose={() => (editing = undefined)} />
{/if}

<style>
  .board-view {
    display: flex;
    flex-direction: column;
    height: 100%;
    min-width: 0;
  }

  header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 16px;
    padding: 18px 24px 12px;
  }

  h1 {
    margin: 0;
    font-size: 20px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .columns {
    flex: 1;
    min-height: 0;
    display: flex;
    align-items: flex-start;
    gap: 14px;
    padding: 4px 24px 20px;
    overflow-x: auto;
  }

  .add-column {
    width: 200px;
    flex-shrink: 0;
    border-style: dashed;
    background: transparent;
    color: var(--text-muted);
    padding: 10px;
  }
</style>
