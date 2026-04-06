<script>
  import { api } from '../lib/api.js';
  import Alert from '../components/Alert.svelte';

  let domain = '';
  let loading = false;
  let error = '';
  let results = null;

  async function check() {
    if (!domain.trim()) return;
    error = '';
    loading = true;
    results = null;
    try {
      const data = await api.checkDns(domain.trim());
      results = data.results || [];
    } catch (e) {
      error = e.message || 'DNS check failed';
    } finally {
      loading = false;
    }
  }

  const dnsReference = [
    { type: 'A', name: 'mail.example.com', value: 'Your server IP' },
    { type: 'MX', name: 'example.com', value: 'mail.example.com (priority 10)' },
    { type: 'TXT', name: 'example.com', value: 'v=spf1 a mx ip4:YOUR_IP ~all' },
    { type: 'TXT', name: '_dmarc.example.com', value: 'v=DMARC1; p=quarantine; rua=mailto:admin@example.com' },
    { type: 'TXT', name: 'mail._domainkey.example.com', value: 'v=DKIM1; k=rsa; p=...' },
    { type: 'PTR', name: 'Your IP', value: 'mail.example.com' },
  ];
</script>

<div class="mx-auto max-w-4xl px-4 py-6">
  {#if error}
    <div class="mb-4"><Alert type="error" message={error} /></div>
  {/if}

  <!-- Check form -->
  <form on:submit|preventDefault={check} class="mb-6 flex items-end gap-2">
    <div class="flex-1">
      <label for="dns-domain" class="mb-1 block text-xs text-[var(--text2)]">Domain</label>
      <input
        id="dns-domain"
        type="text"
        bind:value={domain}
        required
        class="w-full rounded border border-[var(--border)] bg-[var(--input)] px-3 py-1.5 text-sm text-[var(--text)] outline-none focus:border-[var(--accent)]"
        placeholder="example.com"
      />
    </div>
    <button
      type="submit"
      disabled={loading}
      class="rounded bg-[var(--accent)] px-4 py-1.5 text-sm font-medium text-white transition-colors hover:bg-[var(--accent-h)] disabled:opacity-50"
    >
      {loading ? 'Checking...' : 'Check'}
    </button>
  </form>

  <!-- Results -->
  {#if results}
    <div class="mb-6 overflow-x-auto rounded-lg border border-[var(--border)]">
      <table class="w-full text-sm">
        <thead>
          <tr class="border-b border-[var(--border)] text-left text-xs uppercase tracking-wide text-[var(--muted)]">
            <th class="px-3 py-2">Type</th>
            <th class="px-3 py-2">Name</th>
            <th class="px-3 py-2">Expected</th>
            <th class="px-3 py-2">Current</th>
            <th class="px-3 py-2">Status</th>
          </tr>
        </thead>
        <tbody>
          {#each results as r}
            <tr class="border-b border-[var(--border)]/50 text-[var(--text2)]">
              <td class="px-3 py-2 font-mono text-xs">{r.type}</td>
              <td class="px-3 py-2 text-xs">{r.name}</td>
              <td class="max-w-[180px] truncate px-3 py-2 text-xs text-[var(--text2)]">{r.expected}</td>
              <td class="max-w-[180px] truncate px-3 py-2 text-xs text-[var(--text2)]">{r.current || '-'}</td>
              <td class="px-3 py-2">
                <span class="rounded-full border px-2 py-0.5 text-xs {r.ok ? 'border-emerald-500/50 text-emerald-400' : 'border-red-500/50 text-red-400'}">
                  {r.ok ? 'pass' : 'fail'}
                </span>
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  {/if}

  <!-- Reference -->
  <div>
    <p class="mb-2 text-xs uppercase tracking-wide text-[var(--muted)]">DNS records reference</p>
    <div class="overflow-x-auto rounded-lg border border-[var(--border)]">
      <table class="w-full text-sm">
        <thead>
          <tr class="border-b border-[var(--border)] text-left text-xs uppercase tracking-wide text-[var(--muted)]">
            <th class="px-3 py-2">Type</th>
            <th class="px-3 py-2">Name</th>
            <th class="px-3 py-2">Value</th>
          </tr>
        </thead>
        <tbody>
          {#each dnsReference as rec}
            <tr class="border-b border-[var(--border)]/50 text-[var(--text2)]">
              <td class="px-3 py-2 font-mono text-xs">{rec.type}</td>
              <td class="px-3 py-2 text-xs">{rec.name}</td>
              <td class="px-3 py-2 text-xs">{rec.value}</td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  </div>
</div>
