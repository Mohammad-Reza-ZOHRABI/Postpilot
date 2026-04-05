<script>
  import { api } from '../lib/api.js';
  import { navigate } from '../lib/router.js';
  import Alert from '../components/Alert.svelte';

  let email = '';
  let password = '';
  let totp_code = '';
  let loading = false;
  let error = '';

  async function handleSubmit() {
    error = '';
    loading = true;
    try {
      await api.login(email, password, totp_code);
      navigate('/');
    } catch (e) {
      error = e.message || 'Login failed';
    } finally {
      loading = false;
    }
  }
</script>

<div class="mx-auto mt-32 w-full max-w-sm px-4">
  <div class="rounded-lg border border-zinc-800 bg-zinc-900 p-6">
    <div class="mb-6 flex items-center gap-2">
      <span class="flex h-8 w-8 items-center justify-center rounded bg-violet-500 text-sm font-bold text-white">P</span>
      <h1 class="text-lg font-semibold text-zinc-100">Sign in</h1>
    </div>

    {#if error}
      <div class="mb-4">
        <Alert type="error" message={error} />
      </div>
    {/if}

    <form on:submit|preventDefault={handleSubmit} class="space-y-3">
      <div>
        <label for="email" class="mb-1 block text-xs text-zinc-400">Email</label>
        <input
          id="email"
          type="email"
          bind:value={email}
          required
          class="w-full rounded border border-zinc-700 bg-zinc-800 px-3 py-1.5 text-sm text-zinc-100 placeholder-zinc-500 outline-none focus:border-violet-500"
          placeholder="admin@example.com"
        />
      </div>
      <div>
        <label for="password" class="mb-1 block text-xs text-zinc-400">Password</label>
        <input
          id="password"
          type="password"
          bind:value={password}
          required
          class="w-full rounded border border-zinc-700 bg-zinc-800 px-3 py-1.5 text-sm text-zinc-100 placeholder-zinc-500 outline-none focus:border-violet-500"
        />
      </div>
      <div>
        <label for="totp" class="mb-1 block text-xs text-zinc-400">TOTP Code</label>
        <input
          id="totp"
          type="text"
          bind:value={totp_code}
          required
          inputmode="numeric"
          maxlength="6"
          class="w-full rounded border border-zinc-700 bg-zinc-800 px-3 py-1.5 text-sm text-zinc-100 placeholder-zinc-500 outline-none focus:border-violet-500"
          placeholder="000000"
        />
      </div>
      <button
        type="submit"
        disabled={loading}
        class="w-full rounded bg-violet-500 py-2 text-sm font-medium text-white transition-colors hover:bg-violet-400 disabled:opacity-50"
      >
        {loading ? 'Signing in...' : 'Sign in'}
      </button>
    </form>
  </div>
</div>
