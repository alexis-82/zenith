<script lang="ts" module>
  // Open modals, oldest first: only the topmost one reacts to Escape
  // (every modal listens on window, so stopPropagation is not enough).
  const stack: object[] = []
</script>

<script lang="ts">
  import { onDestroy, type Snippet } from 'svelte'

  const id = {}
  stack.push(id)
  onDestroy(() => stack.splice(stack.indexOf(id), 1))

  let {
    title,
    width = '420px',
    onclose,
    children,
    actions,
  }: {
    title: string
    width?: string
    onclose?: () => void
    children: Snippet
    actions?: Snippet
  } = $props()

  function onkeydown(e: KeyboardEvent) {
    if (e.key === 'Escape' && onclose && stack.at(-1) === id) onclose()
  }
</script>

<svelte:window {onkeydown} />

<div class="overlay">
  <div class="modal" style:width role="dialog" aria-modal="true" aria-label={title}>
    <h2>{title}</h2>
    <div class="body">{@render children()}</div>
    {#if actions}
      <div class="actions">{@render actions()}</div>
    {/if}
  </div>
</div>

<style>
  .overlay {
    position: fixed;
    inset: 0;
    background: var(--overlay);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 100;
  }

  .modal {
    max-width: calc(100vw - 32px);
    max-height: calc(100vh - 32px);
    display: flex;
    flex-direction: column;
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: 10px;
    box-shadow: 0 10px 30px rgba(0, 0, 0, 0.25);
    padding: 18px 20px;
  }

  h2 {
    margin: 0 0 12px;
    font-size: 16px;
  }

  .body {
    overflow: auto;
    min-height: 0;
  }

  .actions {
    display: flex;
    justify-content: flex-end;
    flex-wrap: wrap;
    gap: 8px;
    margin-top: 16px;
  }
</style>
