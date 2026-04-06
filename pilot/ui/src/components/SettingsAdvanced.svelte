<script>
  import { createEventDispatcher } from 'svelte';
  import HelpText from './HelpText.svelte';

  export let settings = {};
  export let saving = false;

  const dispatch = createEventDispatcher();

  const modes = [
    { value: 'direct', label: 'Direct' },
    { value: 'relay', label: 'Relay' },
    { value: 'catch', label: 'Catch' },
  ];

  function submit() { dispatch('save'); }
</script>

<form on:submit|preventDefault={submit} class="space-y-0">
  <!-- Mode -->
  <div class="py-4">
    <p class="mb-1 text-xs uppercase tracking-wide text-[var(--muted)]">Mode</p>
    <HelpText text="Direct sends via MX lookup, Relay forwards through a provider, Catch stores without delivering." />
    <div class="mt-2 flex gap-3">
      {#each modes as m}
        <label class="flex cursor-pointer items-center gap-2 rounded border px-3 py-1.5 text-sm transition-colors {settings.mail_mode === m.value ? 'border-[var(--accent)] text-[var(--text)]' : 'border-[var(--border)] text-[var(--text2)] hover:border-[var(--border-h)]'}">
          <input type="radio" bind:group={settings.mail_mode} value={m.value} class="sr-only" />
          {m.label}
        </label>
      {/each}
    </div>
  </div>

  <!-- Identity -->
  <div class="border-t border-[var(--border)] py-4">
    <p class="mb-2 text-xs uppercase tracking-wide text-[var(--muted)]">Identity</p>
    <div class="grid grid-cols-2 gap-3">
      <div>
        <label for="hostname" class="mb-1 block text-xs text-[var(--text2)]">Hostname</label>
        <input id="hostname" type="text" bind:value={settings.postfix_hostname} class="w-full rounded border border-[var(--border)] bg-[var(--input)] px-3 py-1.5 text-sm text-[var(--text)] outline-none focus:border-[var(--accent)]" placeholder="mail.example.com" />
        <HelpText text="FQDN of this mail server, used in SMTP greetings." />
      </div>
      <div>
        <label for="origin" class="mb-1 block text-xs text-[var(--text2)]">Origin</label>
        <input id="origin" type="text" bind:value={settings.postfix_myorigin} class="w-full rounded border border-[var(--border)] bg-[var(--input)] px-3 py-1.5 text-sm text-[var(--text)] outline-none focus:border-[var(--accent)]" placeholder="example.com" />
        <HelpText text="Domain used in From addresses, e.g. example.com" />
      </div>
    </div>
    <div class="mt-3 grid grid-cols-2 gap-3">
      <div>
        <label for="maxsize" class="mb-1 block text-xs text-[var(--text2)]">Max message size (bytes)</label>
        <input id="maxsize" type="number" bind:value={settings.postfix_message_size} class="w-full rounded border border-[var(--border)] bg-[var(--input)] px-3 py-1.5 text-sm text-[var(--text)] outline-none focus:border-[var(--accent)]" />
        <HelpText text="Default: 10485760 (10 MB). Increase for large attachments." />
      </div>
      <div>
        <label for="networks" class="mb-1 block text-xs text-[var(--text2)]">Networks</label>
        <input id="networks" type="text" bind:value={settings.postfix_mynetworks} class="w-full rounded border border-[var(--border)] bg-[var(--input)] px-3 py-1.5 text-sm text-[var(--text)] outline-none focus:border-[var(--accent)]" placeholder="127.0.0.0/8" />
        <HelpText text="IPs allowed to send. Default: 127.0.0.0/8 (localhost only)." />
      </div>
    </div>
  </div>

  <!-- Relay (shown only in relay mode) -->
  {#if settings.mail_mode === 'relay'}
  <div class="border-t border-[var(--border)] py-4">
    <p class="mb-2 text-xs uppercase tracking-wide text-[var(--muted)]">Relay</p>
    <div class="grid grid-cols-2 gap-3">
      <div>
        <label for="relay-host" class="mb-1 block text-xs text-[var(--text2)]">Host</label>
        <input id="relay-host" type="text" bind:value={settings.postfix_relay_host} class="w-full rounded border border-[var(--border)] bg-[var(--input)] px-3 py-1.5 text-sm text-[var(--text)] outline-none focus:border-[var(--accent)]" placeholder="smtp.sendgrid.net:587" />
        <HelpText text="SMTP server with port, e.g. smtp.gmail.com:587" />
      </div>
      <div>
        <label for="relay-user" class="mb-1 block text-xs text-[var(--text2)]">Username</label>
        <input id="relay-user" type="text" bind:value={settings.postfix_relay_user} class="w-full rounded border border-[var(--border)] bg-[var(--input)] px-3 py-1.5 text-sm text-[var(--text)] outline-none focus:border-[var(--accent)]" />
        <HelpText text="API key or email for authentication." />
      </div>
    </div>
    <div class="mt-3">
      <label for="relay-pass" class="mb-1 block text-xs text-[var(--text2)]">Password</label>
      <input id="relay-pass" type="password" bind:value={settings.postfix_relay_pass} class="w-full rounded border border-[var(--border)] bg-[var(--input)] px-3 py-1.5 text-sm text-[var(--text)] outline-none focus:border-[var(--accent)]" />
      <HelpText text="API secret or app password." />
    </div>
  </div>
  {/if}

  <!-- DKIM -->
  <div class="border-t border-[var(--border)] py-4">
    <p class="mb-2 text-xs uppercase tracking-wide text-[var(--muted)]">DKIM</p>
    <label class="mb-3 flex cursor-pointer items-center gap-2 text-sm text-[var(--text2)]">
      <input type="checkbox" bind:checked={settings.dkim_enabled} class="h-4 w-4 rounded border-[var(--border)] bg-[var(--input)] text-violet-500" />
      Enable DKIM signing
    </label>
    <HelpText text="Sign outgoing emails to prove authenticity and improve deliverability." />
    {#if settings.dkim_enabled}
      <div class="mt-3 grid grid-cols-3 gap-3">
        <div>
          <label for="dkim-domain" class="mb-1 block text-xs text-[var(--text2)]">Domain</label>
          <input id="dkim-domain" type="text" bind:value={settings.dkim_domain} class="w-full rounded border border-[var(--border)] bg-[var(--input)] px-3 py-1.5 text-sm text-[var(--text)] outline-none focus:border-[var(--accent)]" />
          <HelpText text="Usually same as Origin." />
        </div>
        <div>
          <label for="dkim-sel" class="mb-1 block text-xs text-[var(--text2)]">Selector</label>
          <input id="dkim-sel" type="text" bind:value={settings.dkim_selector} class="w-full rounded border border-[var(--border)] bg-[var(--input)] px-3 py-1.5 text-sm text-[var(--text)] outline-none focus:border-[var(--accent)]" />
          <HelpText text="DNS prefix. Default: mail" />
        </div>
        <div>
          <label for="dkim-bits" class="mb-1 block text-xs text-[var(--text2)]">Key size</label>
          <select id="dkim-bits" bind:value={settings.dkim_key_size} class="w-full rounded border border-[var(--border)] bg-[var(--input)] px-3 py-1.5 text-sm text-[var(--text)] outline-none focus:border-[var(--accent)]">
            <option value="2048">2048</option>
            <option value="4096">4096</option>
          </select>
          <HelpText text="2048 recommended. 4096 more secure." />
        </div>
      </div>
    {/if}
  </div>

  <!-- Storage -->
  <div class="border-t border-[var(--border)] py-4">
    <p class="mb-2 text-xs uppercase tracking-wide text-[var(--muted)]">Storage</p>
    <div class="max-w-xs">
      <label for="max-msg" class="mb-1 block text-xs text-[var(--text2)]">Max stored messages</label>
      <input id="max-msg" type="number" bind:value={settings.mp_max_messages} class="w-full rounded border border-[var(--border)] bg-[var(--input)] px-3 py-1.5 text-sm text-[var(--text)] outline-none focus:border-[var(--accent)]" />
      <HelpText text="Email log entries kept. Older ones are pruned." />
    </div>
  </div>

  <!-- Save -->
  <div class="border-t border-[var(--border)] pt-4">
    <button type="submit" disabled={saving} class="rounded bg-[var(--accent)] px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-[var(--accent-h)] disabled:opacity-50">
      {saving ? 'Saving...' : 'Save settings'}
    </button>
  </div>
</form>
