<script>
  import { onMount } from 'svelte';
  import { api } from '../lib/api.js';
  import Alert from '../components/Alert.svelte';

  let loading = true;
  let saving = false;
  let error = '';
  let success = '';

  let settings = {
    mail_mode: 'direct',
    postfix_hostname: '',
    postfix_origin: '',
    message_size_limit: 10485760,
    mynetworks: '127.0.0.0/8',
    relay_host: '',
    relay_port: 587,
    relay_user: '',
    relay_password: '',
    dkim_enabled: false,
    dkim_domain: '',
    dkim_selector: 'mail',
    dkim_key_size: 2048,
    max_stored_messages: 1000,
  };

  onMount(async () => {
    try {
      const data = await api.getSettings();
      settings = { ...settings, ...data.settings };
    } catch (e) {
      error = e.message || 'Failed to load settings';
    } finally {
      loading = false;
    }
  });

  async function save() {
    error = '';
    success = '';
    saving = true;
    try {
      await api.saveSettings(settings);
      success = 'Settings saved';
    } catch (e) {
      error = e.message || 'Failed to save';
    } finally {
      saving = false;
    }
  }

  const modes = [
    { value: 'direct', label: 'Direct' },
    { value: 'relay', label: 'Relay' },
    { value: 'queue', label: 'Queue only' },
  ];
</script>

<div class="mx-auto max-w-3xl px-4 py-6">
  {#if loading}
    <p class="text-sm text-zinc-500">Loading...</p>
  {:else}
    {#if error}
      <div class="mb-4"><Alert type="error" message={error} /></div>
    {/if}
    {#if success}
      <div class="mb-4"><Alert type="success" message={success} /></div>
    {/if}

    <form on:submit|preventDefault={save} class="space-y-0">
      <!-- Mode -->
      <div class="py-4">
        <p class="mb-2 text-xs uppercase tracking-wide text-zinc-500">Mode</p>
        <div class="flex gap-3">
          {#each modes as m}
            <label class="flex cursor-pointer items-center gap-2 rounded border px-3 py-1.5 text-sm transition-colors {settings.mail_mode === m.value ? 'border-violet-500 text-zinc-100' : 'border-zinc-700 text-zinc-400 hover:border-zinc-600'}">
              <input
                type="radio"
                bind:group={settings.mail_mode}
                value={m.value}
                class="sr-only"
              />
              {m.label}
            </label>
          {/each}
        </div>
      </div>

      <!-- Identity -->
      <div class="border-t border-zinc-800 py-4">
        <p class="mb-2 text-xs uppercase tracking-wide text-zinc-500">Identity</p>
        <div class="grid grid-cols-2 gap-3">
          <div>
            <label for="hostname" class="mb-1 block text-xs text-zinc-400">Hostname</label>
            <input id="hostname" type="text" bind:value={settings.postfix_hostname} class="w-full rounded border border-zinc-700 bg-zinc-800 px-3 py-1.5 text-sm text-zinc-100 outline-none focus:border-violet-500" placeholder="mail.example.com" />
          </div>
          <div>
            <label for="origin" class="mb-1 block text-xs text-zinc-400">Origin</label>
            <input id="origin" type="text" bind:value={settings.postfix_origin} class="w-full rounded border border-zinc-700 bg-zinc-800 px-3 py-1.5 text-sm text-zinc-100 outline-none focus:border-violet-500" placeholder="example.com" />
          </div>
        </div>
        <div class="mt-3 grid grid-cols-2 gap-3">
          <div>
            <label for="maxsize" class="mb-1 block text-xs text-zinc-400">Max message size (bytes)</label>
            <input id="maxsize" type="number" bind:value={settings.message_size_limit} class="w-full rounded border border-zinc-700 bg-zinc-800 px-3 py-1.5 text-sm text-zinc-100 outline-none focus:border-violet-500" />
          </div>
          <div>
            <label for="networks" class="mb-1 block text-xs text-zinc-400">Networks</label>
            <input id="networks" type="text" bind:value={settings.mynetworks} class="w-full rounded border border-zinc-700 bg-zinc-800 px-3 py-1.5 text-sm text-zinc-100 outline-none focus:border-violet-500" placeholder="127.0.0.0/8" />
          </div>
        </div>
      </div>

      <!-- Relay -->
      <div class="border-t border-zinc-800 py-4">
        <details>
          <summary class="cursor-pointer text-xs uppercase tracking-wide text-zinc-500 select-none">Relay settings</summary>
          <div class="mt-3 grid grid-cols-2 gap-3">
            <div>
              <label for="relay-host" class="mb-1 block text-xs text-zinc-400">Relay host</label>
              <input id="relay-host" type="text" bind:value={settings.relay_host} class="w-full rounded border border-zinc-700 bg-zinc-800 px-3 py-1.5 text-sm text-zinc-100 outline-none focus:border-violet-500" placeholder="smtp.provider.com" />
            </div>
            <div>
              <label for="relay-port" class="mb-1 block text-xs text-zinc-400">Port</label>
              <input id="relay-port" type="number" bind:value={settings.relay_port} class="w-full rounded border border-zinc-700 bg-zinc-800 px-3 py-1.5 text-sm text-zinc-100 outline-none focus:border-violet-500" />
            </div>
          </div>
          <div class="mt-3 grid grid-cols-2 gap-3">
            <div>
              <label for="relay-user" class="mb-1 block text-xs text-zinc-400">Username</label>
              <input id="relay-user" type="text" bind:value={settings.relay_user} class="w-full rounded border border-zinc-700 bg-zinc-800 px-3 py-1.5 text-sm text-zinc-100 outline-none focus:border-violet-500" />
            </div>
            <div>
              <label for="relay-pass" class="mb-1 block text-xs text-zinc-400">Password</label>
              <input id="relay-pass" type="password" bind:value={settings.relay_password} class="w-full rounded border border-zinc-700 bg-zinc-800 px-3 py-1.5 text-sm text-zinc-100 outline-none focus:border-violet-500" />
            </div>
          </div>
        </details>
      </div>

      <!-- DKIM -->
      <div class="border-t border-zinc-800 py-4">
        <p class="mb-2 text-xs uppercase tracking-wide text-zinc-500">DKIM</p>
        <label class="mb-3 flex cursor-pointer items-center gap-2 text-sm text-zinc-300">
          <input type="checkbox" bind:checked={settings.dkim_enabled} class="h-4 w-4 rounded border-zinc-700 bg-zinc-800 text-violet-500" />
          Enable DKIM signing
        </label>
        {#if settings.dkim_enabled}
          <div class="grid grid-cols-3 gap-3">
            <div>
              <label for="dkim-domain" class="mb-1 block text-xs text-zinc-400">Domain</label>
              <input id="dkim-domain" type="text" bind:value={settings.dkim_domain} class="w-full rounded border border-zinc-700 bg-zinc-800 px-3 py-1.5 text-sm text-zinc-100 outline-none focus:border-violet-500" />
            </div>
            <div>
              <label for="dkim-sel" class="mb-1 block text-xs text-zinc-400">Selector</label>
              <input id="dkim-sel" type="text" bind:value={settings.dkim_selector} class="w-full rounded border border-zinc-700 bg-zinc-800 px-3 py-1.5 text-sm text-zinc-100 outline-none focus:border-violet-500" />
            </div>
            <div>
              <label for="dkim-bits" class="mb-1 block text-xs text-zinc-400">Key size</label>
              <select id="dkim-bits" bind:value={settings.dkim_key_size} class="w-full rounded border border-zinc-700 bg-zinc-800 px-3 py-1.5 text-sm text-zinc-100 outline-none focus:border-violet-500">
                <option value={1024}>1024</option>
                <option value={2048}>2048</option>
                <option value={4096}>4096</option>
              </select>
            </div>
          </div>
        {/if}
      </div>

      <!-- Storage -->
      <div class="border-t border-zinc-800 py-4">
        <p class="mb-2 text-xs uppercase tracking-wide text-zinc-500">Storage</p>
        <div class="max-w-xs">
          <label for="max-msg" class="mb-1 block text-xs text-zinc-400">Max stored messages</label>
          <input id="max-msg" type="number" bind:value={settings.max_stored_messages} class="w-full rounded border border-zinc-700 bg-zinc-800 px-3 py-1.5 text-sm text-zinc-100 outline-none focus:border-violet-500" />
        </div>
      </div>

      <!-- Save -->
      <div class="border-t border-zinc-800 pt-4">
        <button
          type="submit"
          disabled={saving}
          class="rounded bg-violet-500 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-violet-400 disabled:opacity-50"
        >
          {saving ? 'Saving...' : 'Save settings'}
        </button>
      </div>
    </form>
  {/if}
</div>
