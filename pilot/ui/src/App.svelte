<script>
  import { onMount } from 'svelte';
  import { route, navigate } from './lib/router.js';
  import { api } from './lib/api.js';
  import './lib/theme.js';
  import Nav from './components/Nav.svelte';
  import Login from './pages/Login.svelte';
  import Setup from './pages/Setup.svelte';
  import Dashboard from './pages/Dashboard.svelte';
  import Settings from './pages/Settings.svelte';
  import ApiKeys from './pages/ApiKeys.svelte';
  import Dns from './pages/Dns.svelte';
  import Users from './pages/Users.svelte';

  let ready = false;
  let loggedIn = false;
  let mode = '';
  let role = '';

  onMount(async () => {
    try {
      const data = await api.check();
      if (data.setup_needed) {
        navigate('/setup');
      } else if (!data.logged_in) {
        navigate('/login');
      } else {
        loggedIn = true;
        role = data.role || 'admin';
      }
    } catch (e) {
      navigate('/login');
    } finally {
      ready = true;
    }
  });

  $: currentRoute = $route;
  $: showNav = loggedIn && currentRoute !== '/login' && currentRoute !== '/setup';
  $: if (currentRoute === '/login' || currentRoute === '/setup') loggedIn = false;
  $: if (ready && currentRoute !== '/login' && currentRoute !== '/setup' && currentRoute !== '') loggedIn = true;
  $: if (loggedIn && !mode) {
    api.getSettings().then(d => { mode = d.settings?.mail_mode || ''; }).catch(() => {});
  }
</script>

<div class="min-h-screen" style="background:var(--bg);color:var(--text)">
  {#if !ready}
    <div class="flex h-screen items-center justify-center">
      <p class="text-sm" style="color:var(--muted)">Loading...</p>
    </div>
  {:else}
    {#if showNav}
      <Nav {currentRoute} {mode} {role} />
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
    {:else if currentRoute === '/users'}
      <Users />
    {:else}
      <Dashboard />
    {/if}
  {/if}
</div>
