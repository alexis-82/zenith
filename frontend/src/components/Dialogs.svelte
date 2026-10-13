<script lang="ts">
  import { dialogs } from '../lib/dialogs.svelte'
  import Modal from './Modal.svelte'

  let value = $state('')

  $effect(() => {
    if (dialogs.pending?.kind === 'prompt') value = dialogs.pending.opts.value ?? ''
  })

  function focus(node: HTMLElement) {
    node.focus()
    if (node instanceof HTMLInputElement) node.select()
  }

  function submitPrompt(e: SubmitEvent) {
    e.preventDefault()
    if (value.trim()) dialogs.close(value)
  }
</script>

{#if dialogs.pending?.kind === 'confirm'}
  {@const opts = dialogs.pending.opts}
  <Modal title={opts.title} onclose={() => dialogs.close(false)}>
    <p class="message">{opts.message}</p>
    {#snippet actions()}
      <button onclick={() => dialogs.close(false)}>Annulla</button>
      <button class={opts.danger ? 'primary danger-fill' : 'primary'} onclick={() => dialogs.close(true)} use:focus>
        {opts.confirmLabel ?? 'Conferma'}
      </button>
    {/snippet}
  </Modal>
{:else if dialogs.pending?.kind === 'prompt'}
  {@const opts = dialogs.pending.opts}
  <Modal title={opts.title} onclose={() => dialogs.close(null)}>
    <form id="prompt-form" onsubmit={submitPrompt}>
      <label>
        <span class="muted">{opts.label}</span>
        <input bind:value use:focus />
      </label>
    </form>
    {#snippet actions()}
      <button type="button" onclick={() => dialogs.close(null)}>Annulla</button>
      <button class="primary" type="submit" form="prompt-form" disabled={!value.trim()}>
        {opts.confirmLabel ?? 'Salva'}
      </button>
    {/snippet}
  </Modal>
{/if}

<style>
  .message {
    margin: 0;
    white-space: pre-line;
    line-height: 1.5;
  }

  label {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .danger-fill {
    background: var(--danger);
    border-color: var(--danger);
    color: #fff;
  }
</style>
