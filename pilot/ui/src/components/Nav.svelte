<script>
  import { navigate } from '../lib/router.js';
  import { api } from '../lib/api.js';
  import { theme, toggleTheme } from '../lib/theme.js';

  export let currentRoute = '/';
  export let mode = '';
  export let role = '';

  $: links = [
    { path: '/', label: 'Dashboard' },
    { path: '/settings', label: 'Settings', admin: true },
    { path: '/api-keys', label: 'API Keys' },
    { path: '/dns', label: 'DNS', admin: true },
    ...(role === 'admin' ? [{ path: '/users', label: 'Users' }] : []),
  ].filter(l => !l.admin || role === 'admin');

  function go(e, path) {
    e.preventDefault();
    navigate(path);
  }

  async function signOut() {
    try { await api.logout(); } catch (_) {}
    navigate('/login');
  }
</script>

<nav class="sticky top-0 z-50 border-b border-[var(--border)] bg-[var(--bg)]/90 backdrop-blur">
  <div class="mx-auto flex h-12 max-w-6xl items-center justify-between px-4">
    <div class="flex items-center gap-3">
      <a href="#/" on:click={(e) => go(e, '/')} class="flex items-center gap-2">
        <span class="flex h-7 w-7 items-center justify-center rounded bg-violet-500 text-sm font-bold text-white">P</span>
        <span class="text-sm font-semibold text-[var(--text)]">Postpilot</span>
      </a>
      {#if mode}
        <span class="rounded-full border border-[var(--border)] px-2 py-0.5 text-xs text-[var(--muted)]">{mode}</span>
      {/if}
    </div>

    <div class="flex items-center gap-1">
      {#each links as link}
        <a
          href="#{link.path}"
          on:click={(e) => go(e, link.path)}
          class="rounded px-3 py-1.5 text-sm transition-colors {currentRoute === link.path ? 'bg-[var(--input)] text-[var(--text)]' : 'text-[var(--muted)] hover:text-[var(--text)]'}"
        >
          {link.label}
        </a>
      {/each}
    </div>

    <div class="flex items-center gap-2">
      <button
        on:click={toggleTheme}
        class="rounded p-1.5 text-[var(--muted)] transition-colors hover:text-[var(--text)]"
        title="Toggle theme"
      >
        {#if $theme === 'dark'}
          <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="5"/><path d="M12 1v2m0 18v2M4.22 4.22l1.42 1.42m12.72 12.72l1.42 1.42M1 12h2m18 0h2M4.22 19.78l1.42-1.42M18.36 5.64l1.42-1.42"/></svg>
        {:else}
          <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path d="M21 12.79A9 9 0 1111.21 3 7 7 0 0021 12.79z"/></svg>
        {/if}
      </button>
      <button on:click={signOut} class="text-sm text-[var(--muted)] transition-colors hover:text-[var(--text)]">
        Sign out
      </button>
    </div>
  </div>
</nav>
