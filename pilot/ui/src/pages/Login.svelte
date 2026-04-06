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
  <div class="rounded-lg border border-[var(--border)] bg-[var(--surface)] p-6">
    <div class="mb-6 flex items-center gap-2">
      <span class="flex h-8 w-8 items-center justify-center rounded bg-[var(--accent)] text-sm font-bold text-white">P</span>
      <h1 class="text-lg font-semibold text-[var(--text)]">Sign in</h1>
    </div>

    {#if error}
      <div class="mb-4">
        <Alert type="error" message={error} />
      </div>
    {/if}

    <form on:submit|preventDefault={handleSubmit} class="space-y-3">
      <div>
        <label for="email" class="mb-1 block text-xs text-[var(--text2)]">Email</label>
        <input
          id="email"
          type="email"
          bind:value={email}
          required
          class="w-full rounded border border-[var(--border)] bg-[var(--input)] px-3 py-1.5 text-sm text-[var(--text)] placeholder-[var(--placeholder)] outline-none focus:border-[var(--accent)]"
          placeholder="admin@example.com"
        />
      </div>
      <div>
        <label for="password" class="mb-1 block text-xs text-[var(--text2)]">Password</label>
        <input
          id="password"
          type="password"
          bind:value={password}
          required
          class="w-full rounded border border-[var(--border)] bg-[var(--input)] px-3 py-1.5 text-sm text-[var(--text)] placeholder-[var(--placeholder)] outline-none focus:border-[var(--accent)]"
        />
      </div>
      <div>
        <label for="totp" class="mb-1 block text-xs text-[var(--text2)]">TOTP Code</label>
        <input
          id="totp"
          type="text"
          bind:value={totp_code}
          required
          inputmode="numeric"
          maxlength="6"
          class="w-full rounded border border-[var(--border)] bg-[var(--input)] px-3 py-1.5 text-sm text-[var(--text)] placeholder-[var(--placeholder)] outline-none focus:border-[var(--accent)]"
          placeholder="000000"
        />
      </div>
      <button
        type="submit"
        disabled={loading}
        class="w-full rounded bg-[var(--accent)] py-2 text-sm font-medium text-white transition-colors hover:bg-[var(--accent-h)] disabled:opacity-50"
      >
        {loading ? 'Signing in...' : 'Sign in'}
      </button>
    </form>
  </div>
</div>
