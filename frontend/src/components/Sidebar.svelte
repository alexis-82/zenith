<script lang="ts">
  import { app } from '../lib/app.svelte'
  import { addProject, deleteProject, renameProject } from '../lib/board'
  import { dialogs } from '../lib/dialogs.svelte'
  import type { Project } from '../lib/types'
  import SaveStatus from './SaveStatus.svelte'

  async function create() {
    const name = await dialogs.prompt({ title: 'Nuovo progetto', label: 'Nome del progetto', confirmLabel: 'Crea' })
    if (name === null) return
    let id = ''
    app.mutate((b) => {
      const r = addProject(b, name)
      id = r.id
      return r.board
    })
    app.selectedProjectId = id
    app.view = 'board'
  }

  async function rename(p: Project) {
    const name = await dialogs.prompt({ title: 'Rinomina progetto', label: 'Nome del progetto', value: p.name })
    if (name !== null) app.mutate((b) => renameProject(b, p.id, name))
  }

  async function remove(p: Project) {
    const n = p.tasks.length
    const ok = await dialogs.confirm({
      title: 'Elimina progetto',
      message: `Eliminare "${p.name}"?` + (n ? `\nVerranno eliminati anche ${n} task.` : ''),
      confirmLabel: 'Elimina',
      danger: true,
    })
    if (!ok) return
    app.mutate((b) => deleteProject(b, p.id))
    if (app.selectedProjectId === p.id) app.selectedProjectId = app.board?.projects[0]?.id ?? null
  }
</script>

<aside>
  <div class="brand">Zenith</div>

  {#if app.board}
    <div class="section-title">Progetti</div>
    <ul>
      {#each app.board.projects as p (p.id)}
        <li class:active={p.id === app.selectedProjectId && app.view === 'board'}>
          <button
            class="name"
            onclick={() => {
              app.selectedProjectId = p.id
              app.view = 'board'
            }}
            title={p.name}>{p.name}</button
          >
          <span class="actions">
            <button class="icon" title="Rinomina" aria-label="Rinomina progetto" onclick={() => rename(p)}>✎</button>
            <button class="icon" title="Elimina" aria-label="Elimina progetto" onclick={() => remove(p)}>🗑</button>
          </span>
        </li>
      {/each}
    </ul>
    <button class="new" onclick={create}>+ Nuovo progetto</button>
  {/if}

  <div class="spacer"></div>

  <div class="footer">
    <SaveStatus />
    <button class="settings-btn" class:active-btn={app.view === 'settings'} onclick={() => (app.view = app.view === 'settings' ? 'board' : 'settings')}>
      ⚙ Impostazioni
    </button>
  </div>
</aside>

<style>
  aside {
    width: 240px;
    flex-shrink: 0;
    display: flex;
    flex-direction: column;
    background: var(--surface);
    border-right: 1px solid var(--border);
    padding: 16px 10px;
  }

  .brand {
    font-size: 18px;
    font-weight: 700;
    letter-spacing: 0.5px;
    padding: 0 8px 16px;
  }

  .section-title {
    font-size: 11px;
    text-transform: uppercase;
    letter-spacing: 0.6px;
    color: var(--text-muted);
    padding: 0 8px 6px;
  }

  ul {
    list-style: none;
    margin: 0;
    padding: 0;
    overflow-y: auto;
  }

  li {
    display: flex;
    align-items: center;
    border-radius: 6px;
  }

  li:hover,
  li.active {
    background: var(--sidebar-item);
  }

  li.active .name {
    font-weight: 600;
  }

  .name {
    flex: 1;
    min-width: 0;
    text-align: left;
    border: none;
    background: transparent;
    padding: 7px 8px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .actions {
    display: none;
    padding-right: 4px;
  }

  li:hover .actions {
    display: flex;
  }

  .new {
    margin-top: 8px;
    border-style: dashed;
    border-color: var(--input-border);
    background: var(--soft-button);
    color: var(--text-muted);
  }

  .new:hover {
    background: var(--sidebar-item);
    color: var(--text);
  }

  .spacer {
    flex: 1;
  }

  .footer {
    display: flex;
    flex-direction: column;
    gap: 10px;
    padding: 0 4px;
  }

  .settings-btn {
    background: var(--soft-button);
    border-color: var(--input-border);
  }

  .settings-btn:hover,
  .active-btn {
    background: var(--sidebar-item);
  }
</style>
