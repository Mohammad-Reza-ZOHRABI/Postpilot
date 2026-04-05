<script>
  import { navigate } from '../lib/router.js';
  import { api } from '../lib/api.js';

  export let currentRoute = '/';
  export let mode = '';

  const links = [
    { path: '/', label: 'Dashboard' },
    { path: '/settings', label: 'Settings' },
    { path: '/api-keys', label: 'API Keys' },
    { path: '/dns', label: 'DNS' },
  ];

  function go(e, path) {
    e.preventDefault();
    navigate(path);
  }

  async function signOut() {
    try {
      await api.logout();
    } catch (_) {}
    navigate('/login');
  }
</script>

<nav class="sticky top-0 z-50 border-b border-zinc-800 bg-zinc-950/90 backdrop-blur">
  <div class="mx-auto flex h-12 max-w-6xl items-center justify-between px-4">
    <div class="flex items-center gap-3">
      <a href="#/" on:click={(e) => go(e, '/')} class="flex items-center gap-2">
        <span class="flex h-7 w-7 items-center justify-center rounded bg-violet-500 text-sm font-bold text-white">P</span>
        <span class="text-sm font-semibold text-zinc-100">Postpilot</span>
      </a>
      {#if mode}
        <span class="rounded-full border border-zinc-700 px-2 py-0.5 text-xs text-zinc-400">{mode}</span>
      {/if}
    </div>

    <div class="flex items-center gap-1">
      {#each links as link}
        <a
          href="#{link.path}"
          on:click={(e) => go(e, link.path)}
          class="rounded px-3 py-1.5 text-sm transition-colors {currentRoute === link.path ? 'bg-zinc-800 text-zinc-100' : 'text-zinc-400 hover:text-zinc-200'}"
        >
          {link.label}
        </a>
      {/each}
    </div>

    <button on:click={signOut} class="text-sm text-zinc-500 transition-colors hover:text-zinc-300">
      Sign out
    </button>
  </div>
</nav>
