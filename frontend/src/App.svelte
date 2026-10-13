<script lang="ts">
  import { onMount } from 'svelte'
  import Board from './components/Board.svelte'
  import CloseBlockedDialog from './components/CloseBlockedDialog.svelte'
  import Dialogs from './components/Dialogs.svelte'
  import EmptyState from './components/EmptyState.svelte'
  import ExternalChange from './components/ExternalChange.svelte'
  import Settings from './components/Settings.svelte'
  import Sidebar from './components/Sidebar.svelte'
  import { app } from './lib/app.svelte'
  import { applyTheme } from './lib/theme'

  let ready = $state(false)

  onMount(async () => {
    await app.init()
    applyTheme(app.settings.theme)
    ready = true
  })
</script>

{#if ready}
  <div class="layout">
    <Sidebar />
    <main>
      <ExternalChange />
      <div class="content">
        {#if app.view === 'settings'}
          <Settings />
        {:else if !app.board}
          <EmptyState />
        {:else if app.selectedProject}
          <Board project={app.selectedProject} />
        {:else}
          <div class="hint muted">Nessun progetto. Creane uno con "+ Nuovo progetto" nella barra laterale.</div>
        {/if}
      </div>
    </main>
  </div>
  <Dialogs />
  <CloseBlockedDialog />
{/if}

<style>
  .layout {
    display: flex;
    height: 100%;
  }

  main {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
  }

  .content {
    flex: 1;
    min-height: 0;
    overflow: auto;
  }

  .hint {
    margin: 15vh auto 0;
    text-align: center;
  }
</style>
