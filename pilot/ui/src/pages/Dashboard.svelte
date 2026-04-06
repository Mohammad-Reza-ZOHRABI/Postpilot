<script>
  import { onMount } from 'svelte';
  import { api } from '../lib/api.js';
  import { navigate } from '../lib/router.js';
  import Alert from '../components/Alert.svelte';

  let loading = true;
  let error = '';
  let stats = { sent_24h: 0, sent_7d: 0, failed_24h: 0, queued: 0 };
  let services = [];
  let recent = [];

  const onboardingSteps = [
    { num: 1, label: 'Configure mail settings', route: '/settings' },
    { num: 2, label: 'Set up DNS records', route: '/dns' },
    { num: 3, label: 'Create an API key', route: '/api-keys' },
    { num: 4, label: 'Send your first email', route: null },
  ];

  onMount(async () => {
    try {
      const data = await api.dashboard();
      stats = data.stats;
      services = data.services || [];
      recent = data.recent || [];
    } catch (e) {
      error = e.message || 'Failed to load dashboard';
    } finally {
      loading = false;
    }
  });

  function statusClass(status) {
    switch (status) {
      case 'sent':
      case 'delivered':
        return 'border-emerald-500/50 text-emerald-400';
      case 'failed':
      case 'bounced':
        return 'border-red-500/50 text-red-400';
      case 'queued':
      case 'pending':
        return 'border-amber-500/50 text-amber-400';
      default:
        return 'border-[var(--border-h)] text-[var(--text2)]';
    }
  }

  function formatTime(ts) {
    if (!ts) return '-';
    const d = new Date(ts);
    const now = new Date();
    const diff = now - d;
    if (diff < 60000) return 'just now';
    if (diff < 3600000) return Math.floor(diff / 60000) + 'm ago';
    if (diff < 86400000) return Math.floor(diff / 3600000) + 'h ago';
    return d.toLocaleDateString();
  }
</script>

<div class="mx-auto max-w-6xl px-4 py-6">
  {#if loading}
    <p class="text-sm text-[var(--muted)]">Loading...</p>
  {:else if error}
    <Alert type="error" message={error} />
  {:else}
    <!-- Stat cards -->
    <div class="mb-6 grid grid-cols-2 gap-3 lg:grid-cols-4">
      <div class="rounded-lg border border-[var(--border)] bg-[var(--surface)] px-4 py-3">
        <p class="text-2xl font-semibold text-[var(--text)]">{stats.sent_24h}</p>
        <p class="text-xs uppercase tracking-wide text-[var(--muted)]">Sent 24h</p>
      </div>
      <div class="rounded-lg border border-[var(--border)] bg-[var(--surface)] px-4 py-3">
        <p class="text-2xl font-semibold text-[var(--text)]">{stats.sent_7d}</p>
        <p class="text-xs uppercase tracking-wide text-[var(--muted)]">Sent 7d</p>
      </div>
      <div class="rounded-lg border border-[var(--border)] bg-[var(--surface)] px-4 py-3">
        <p class="text-2xl font-semibold text-[var(--text)]">{stats.failed_24h}</p>
        <p class="text-xs uppercase tracking-wide text-[var(--muted)]">Failed 24h</p>
      </div>
      <div class="rounded-lg border border-[var(--border)] bg-[var(--surface)] px-4 py-3">
        <p class="text-2xl font-semibold text-[var(--text)]">{stats.queued}</p>
        <p class="text-xs uppercase tracking-wide text-[var(--muted)]">Queued</p>
      </div>
    </div>

    <!-- Services -->
    {#if services.length > 0}
      <div class="mb-6 flex flex-wrap items-center gap-2">
        {#each services as svc}
          <span class="flex items-center gap-1.5 rounded-full border border-[var(--border)] px-2.5 py-1 text-xs text-[var(--text2)]">
            <span class="h-1.5 w-1.5 rounded-full {svc.running ? 'bg-emerald-400' : 'bg-red-400'}"></span>
            {svc.name}
          </span>
        {/each}
      </div>
    {/if}

    <!-- Recent emails -->
    {#if recent.length > 0}
      <div class="overflow-x-auto rounded-lg border border-[var(--border)]">
        <table class="w-full text-sm">
          <thead>
            <tr class="border-b border-[var(--border)] text-left text-xs uppercase tracking-wide text-[var(--muted)]">
              <th class="px-3 py-2">Time</th>
              <th class="px-3 py-2">From</th>
              <th class="px-3 py-2">To</th>
              <th class="px-3 py-2">Subject</th>
              <th class="px-3 py-2">Status</th>
            </tr>
          </thead>
          <tbody>
            {#each recent as msg}
              <tr class="border-b border-[var(--border)]/50 text-[var(--text2)]">
                <td class="whitespace-nowrap px-3 py-2 text-xs text-[var(--muted)]">{formatTime(msg.created_at)}</td>
                <td class="px-3 py-2 text-xs">{msg.from_addr}</td>
                <td class="px-3 py-2 text-xs">{msg.to_addr}</td>
                <td class="max-w-[200px] truncate px-3 py-2 text-xs">{msg.subject}</td>
                <td class="px-3 py-2">
                  <span class="rounded-full border px-2 py-0.5 text-xs {statusClass(msg.status)}">
                    {msg.status}
                  </span>
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    {:else}
      <!-- Onboarding -->
      <div class="rounded-lg border border-[var(--border)] bg-[var(--surface)] p-4">
        <p class="mb-3 text-sm text-[var(--text2)]">Get started</p>
        <div class="space-y-2">
          {#each onboardingSteps as s}
            <div class="flex items-center gap-3">
              <span class="flex h-6 w-6 items-center justify-center rounded-full border border-[var(--border)] text-xs text-[var(--text2)]">{s.num}</span>
              {#if s.route}
                <button on:click={() => navigate(s.route)} class="text-sm text-[var(--accent)] hover:text-[var(--accent-h)]">{s.label}</button>
              {:else}
                <span class="text-sm text-[var(--text2)]">{s.label}</span>
              {/if}
            </div>
          {/each}
        </div>
      </div>
    {/if}
  {/if}
</div>
