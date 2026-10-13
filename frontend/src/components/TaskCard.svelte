<script lang="ts">
  import type { Task } from '../lib/types'

  let { task, onopen }: { task: Task; onopen: () => void } = $props()

  // First non-empty line of the description, without common Markdown markers.
  const preview = $derived(
    task.description
      .split('\n')
      .map((l) => l.replace(/^[#>*\-+\s]+|[*_`]/g, '').trim())
      .find((l) => l) ?? '',
  )
</script>

<button class="card" onclick={onopen}>
  <span class="title">{task.title}</span>
  {#if preview}<span class="preview muted">{preview}</span>{/if}
</button>

<style>
  .card {
    display: flex;
    flex-direction: column;
    gap: 4px;
    width: 100%;
    text-align: left;
    background: var(--card);
    border: 1px solid var(--border);
    border-radius: 8px;
    padding: 10px 12px;
    box-shadow: var(--shadow);
  }

  .card:hover {
    background: var(--card);
    border-color: var(--accent);
  }

  .title {
    font-weight: 500;
    overflow-wrap: anywhere;
  }

  .preview {
    font-size: 12px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
</style>
