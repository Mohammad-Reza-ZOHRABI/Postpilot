<script>
  import { onMount } from 'svelte';
  import { api } from '../lib/api.js';
  import Alert from '../components/Alert.svelte';

  let loading = true;
  let error = '';
  let success = '';
  let keys = [];
  let newKey = '';

  let name = '';
  let permissions = 'send';
  let rate_limit = 100;
  let creating = false;

  onMount(async () => {
    await loadKeys();
  });

  async function loadKeys() {
    loading = true;
    error = '';
    try {
      const data = await api.getKeys();
      keys = data.keys || [];
    } catch (e) {
      error = e.message || 'Failed to load keys';
    } finally {
      loading = false;
    }
  }

  async function create() {
    if (!name.trim()) return;
    error = '';
    creating = true;
    try {
      const data = await api.createKey(name.trim(), permissions, rate_limit);
      newKey = data.key;
      name = '';
      await loadKeys();
    } catch (e) {
      error = e.message || 'Failed to create key';
    } finally {
      creating = false;
    }
  }

  async function revoke(id) {
    if (!confirm('Revoke this key? This cannot be undone.')) return;
    error = '';
    try {
      await api.revokeKey(id);
      success = 'Key revoked';
      await loadKeys();
    } catch (e) {
      error = e.message || 'Failed to revoke key';
    }
  }

  function copyKey() {
    navigator.clipboard.writeText(newKey).then(() => {
      success = 'Copied to clipboard';
    });
  }

  function formatDate(ts) {
    if (!ts) return 'Never';
    return new Date(ts).toLocaleDateString();
  }

  const curlExample = `curl -X POST https://your-server/api/v1/send \\
  -H "Authorization: Bearer pp_live_..." \\
  -H "Content-Type: application/json" \\
  -d '{
  "from": "you@example.com",
  "to": "user@example.com",
  "subject": "Hello",
  "text": "Hello from Postpilot"
}'`;
</script>

<div class="mx-auto max-w-5xl px-4 py-6">
  {#if error}
    <div class="mb-4"><Alert type="error" message={error} /></div>
  {/if}
  {#if success}
    <div class="mb-4"><Alert type="success" message={success} /></div>
  {/if}

  <!-- Create form -->
  <form on:submit|preventDefault={create} class="mb-4 flex items-end gap-2">
    <div class="flex-1">
      <label for="key-name" class="mb-1 block text-xs text-zinc-400">Name</label>
      <input
        id="key-name"
        type="text"
        bind:value={name}
        required
        class="w-full rounded border border-zinc-700 bg-zinc-800 px-3 py-1.5 text-sm text-zinc-100 outline-none focus:border-violet-500"
        placeholder="my-app"
      />
    </div>
    <div>
      <label for="key-perms" class="mb-1 block text-xs text-zinc-400">Permissions</label>
      <select id="key-perms" bind:value={permissions} class="rounded border border-zinc-700 bg-zinc-800 px-3 py-1.5 text-sm text-zinc-100 outline-none focus:border-violet-500">
        <option value="send">send</option>
        <option value="send,status">send + status</option>
        <option value="admin">admin</option>
      </select>
    </div>
    <div class="w-24">
      <label for="key-rate" class="mb-1 block text-xs text-zinc-400">Rate/min</label>
      <input id="key-rate" type="number" bind:value={rate_limit} class="w-full rounded border border-zinc-700 bg-zinc-800 px-3 py-1.5 text-sm text-zinc-100 outline-none focus:border-violet-500" />
    </div>
    <button
      type="submit"
      disabled={creating}
      class="rounded bg-violet-500 px-4 py-1.5 text-sm font-medium text-white transition-colors hover:bg-violet-400 disabled:opacity-50"
    >
      {creating ? 'Creating...' : 'Create'}
    </button>
  </form>

  <!-- New key display -->
  {#if newKey}
    <div class="mb-4 flex items-center gap-2 rounded border border-emerald-500/30 bg-emerald-500/5 px-3 py-2">
      <code class="flex-1 break-all font-mono text-sm text-emerald-400">{newKey}</code>
      <button on:click={copyKey} class="shrink-0 rounded border border-zinc-700 px-2 py-1 text-xs text-zinc-400 transition-colors hover:text-zinc-200">
        Copy
      </button>
    </div>
    <p class="mb-4 text-xs text-zinc-500">Save this key now. It won't be shown again.</p>
  {/if}

  <!-- Keys table -->
  {#if loading}
    <p class="text-sm text-zinc-500">Loading...</p>
  {:else if keys.length === 0}
    <p class="py-8 text-center text-sm text-zinc-500">No API keys yet</p>
  {:else}
    <div class="overflow-x-auto rounded-lg border border-zinc-800">
      <table class="w-full text-sm">
        <thead>
          <tr class="border-b border-zinc-800 text-left text-xs uppercase tracking-wide text-zinc-500">
            <th class="px-3 py-2">Name</th>
            <th class="px-3 py-2">Key</th>
            <th class="px-3 py-2">Permissions</th>
            <th class="px-3 py-2">Calls</th>
            <th class="px-3 py-2">Last used</th>
            <th class="px-3 py-2"></th>
          </tr>
        </thead>
        <tbody>
          {#each keys as key}
            <tr class="border-b border-zinc-800/50 text-zinc-300 {key.revoked_at ? 'opacity-40' : ''}">
              <td class="px-3 py-2 text-xs">{key.name}</td>
              <td class="px-3 py-2 font-mono text-xs text-zinc-500">{key.key_prefix}...</td>
              <td class="px-3 py-2 text-xs">{key.permissions}</td>
              <td class="px-3 py-2 text-xs">{key.call_count}</td>
              <td class="px-3 py-2 text-xs text-zinc-500">{formatDate(key.last_used_at)}</td>
              <td class="px-3 py-2">
                {#if !key.revoked_at}
                  <button on:click={() => revoke(key.id)} class="text-xs text-red-400 transition-colors hover:text-red-300">
                    Revoke
                  </button>
                {:else}
                  <span class="text-xs text-zinc-600">Revoked</span>
                {/if}
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  {/if}

  <!-- Quick start -->
  <div class="mt-6">
    <p class="mb-2 text-xs uppercase tracking-wide text-zinc-500">Quick start</p>
    <pre class="overflow-x-auto rounded-lg border border-zinc-800 bg-zinc-900 p-3 text-xs text-zinc-400"><code>{curlExample}</code></pre>
  </div>
</div>
