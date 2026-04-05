<script>
  import { onMount } from 'svelte';
  import { route, navigate } from './lib/router.js';
  import { api } from './lib/api.js';
  import Nav from './components/Nav.svelte';
  import Login from './pages/Login.svelte';
  import Setup from './pages/Setup.svelte';
  import Dashboard from './pages/Dashboard.svelte';
  import Settings from './pages/Settings.svelte';
  import ApiKeys from './pages/ApiKeys.svelte';
  import Dns from './pages/Dns.svelte';

  let ready = false;
  let loggedIn = false;
  let mode = '';

  onMount(async () => {
    try {
      const data = await api.check();
      if (data.setup_needed) {
        navigate('/setup');
      } else if (!data.logged_in) {
        navigate('/login');
      } else {
        loggedIn = true;
      }
    } catch (e) {
      navigate('/login');
    } finally {
      ready = true;
    }
  });

  $: currentRoute = $route;
  $: showNav = loggedIn && currentRoute !== '/login' && currentRoute !== '/setup';

  // Track login state from route changes
  $: if (currentRoute === '/login' || currentRoute === '/setup') {
    loggedIn = false;
  }

  // After navigating away from login/setup, assume logged in
  $: if (ready && currentRoute !== '/login' && currentRoute !== '/setup' && currentRoute !== '') {
    loggedIn = true;
  }

  // Load mode for nav display
  $: if (loggedIn && !mode) {
    api.getSettings().then(d => {
      mode = d.settings?.mail_mode || '';
    }).catch(() => {});
  }
</script>

<div class="min-h-screen bg-zinc-950 text-zinc-100">
  {#if !ready}
    <div class="flex h-screen items-center justify-center">
      <p class="text-sm text-zinc-500">Loading...</p>
    </div>
  {:else}
    {#if showNav}
      <Nav {currentRoute} {mode} />
    {/if}

    {#if currentRoute === '/login'}
      <Login />
    {:else if currentRoute === '/setup'}
      <Setup />
    {:else if currentRoute === '/settings'}
      <Settings />
    {:else if currentRoute === '/api-keys'}
      <ApiKeys />
    {:else if currentRoute === '/dns'}
      <Dns />
    {:else}
      <Dashboard />
    {/if}
  {/if}
</div>
