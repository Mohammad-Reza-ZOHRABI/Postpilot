<script>
  import { onMount } from 'svelte';

  export let type = 'error';
  export let message = '';
  export let dismissable = true;

  let visible = true;

  onMount(() => {
    if (dismissable) {
      const timer = setTimeout(() => { visible = false; }, 5000);
      return () => clearTimeout(timer);
    }
  });
</script>

{#if visible && message}
  <div class="border-l-4 px-3 py-2 text-sm {type === 'success' ? 'border-emerald-500 bg-emerald-500/5 text-emerald-400' : 'border-red-500 bg-red-500/5 text-red-400'}">
    {message}
  </div>
{/if}
