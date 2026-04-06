<script>
  import { api } from '../lib/api.js';
  import { navigate } from '../lib/router.js';
  import Alert from '../components/Alert.svelte';

  let step = 1;
  let email = '';
  let password = '';
  let confirmPassword = '';
  let totp_secret = '';
  let qr_data_url = '';
  let totp_code = '';
  let loading = false;
  let error = '';

  async function handleStep1() {
    error = '';
    if (password.length < 12) {
      error = 'Password must be at least 12 characters';
      return;
    }
    if (password !== confirmPassword) {
      error = 'Passwords do not match';
      return;
    }
    loading = true;
    try {
      const res = await api.setupStep1(email, password);
      totp_secret = res.totp_secret;
      qr_data_url = res.qr_data_url;
      step = 2;
    } catch (e) {
      error = e.message || 'Setup failed';
    } finally {
      loading = false;
    }
  }

  async function handleStep2() {
    error = '';
    loading = true;
    try {
      await api.setupStep2(email, password, totp_secret, totp_code);
      navigate('/login');
    } catch (e) {
      error = e.message || 'Verification failed';
    } finally {
      loading = false;
    }
  }
</script>

<div class="mx-auto mt-24 w-full max-w-sm px-4">
  <div class="rounded-lg border border-[var(--border)] bg-[var(--surface)] p-6">
    <div class="mb-4 flex items-center gap-2">
      <span class="flex h-8 w-8 items-center justify-center rounded bg-[var(--accent)] text-sm font-bold text-white">P</span>
      <h1 class="text-lg font-semibold text-[var(--text)]">Setup</h1>
    </div>

    <!-- Stepper -->
    <div class="mb-6 flex items-center justify-center gap-0">
      <span class="h-2.5 w-2.5 rounded-full {step >= 1 ? 'bg-[var(--accent)]' : 'bg-zinc-700'}"></span>
      <span class="h-px w-10 {step >= 2 ? 'bg-[var(--accent)]' : 'bg-zinc-700'}"></span>
      <span class="h-2.5 w-2.5 rounded-full {step >= 2 ? 'bg-[var(--accent)]' : 'bg-zinc-700'}"></span>
    </div>

    {#if error}
      <div class="mb-4">
        <Alert type="error" message={error} />
      </div>
    {/if}

    {#if step === 1}
      <form on:submit|preventDefault={handleStep1} class="space-y-3">
        <div>
          <label for="s-email" class="mb-1 block text-xs text-[var(--text2)]">Email</label>
          <input
            id="s-email"
            type="email"
            bind:value={email}
            required
            class="w-full rounded border border-[var(--border)] bg-[var(--input)] px-3 py-1.5 text-sm text-[var(--text)] placeholder-[var(--placeholder)] outline-none focus:border-[var(--accent)]"
          />
        </div>
        <div>
          <label for="s-pass" class="mb-1 block text-xs text-[var(--text2)]">Password (min 12 chars)</label>
          <input
            id="s-pass"
            type="password"
            bind:value={password}
            required
            minlength="12"
            class="w-full rounded border border-[var(--border)] bg-[var(--input)] px-3 py-1.5 text-sm text-[var(--text)] placeholder-[var(--placeholder)] outline-none focus:border-[var(--accent)]"
          />
        </div>
        <div>
          <label for="s-confirm" class="mb-1 block text-xs text-[var(--text2)]">Confirm password</label>
          <input
            id="s-confirm"
            type="password"
            bind:value={confirmPassword}
            required
            class="w-full rounded border border-[var(--border)] bg-[var(--input)] px-3 py-1.5 text-sm text-[var(--text)] placeholder-[var(--placeholder)] outline-none focus:border-[var(--accent)]"
          />
        </div>
        <button
          type="submit"
          disabled={loading}
          class="w-full rounded bg-[var(--accent)] py-2 text-sm font-medium text-white transition-colors hover:bg-[var(--accent-h)] disabled:opacity-50"
        >
          {loading ? 'Creating...' : 'Continue'}
        </button>
      </form>
    {:else}
      <form on:submit|preventDefault={handleStep2} class="space-y-4">
        <p class="text-xs text-[var(--text2)]">Scan this QR code with your authenticator app.</p>
        <div class="flex justify-center">
          <img src={qr_data_url} alt="TOTP QR Code" class="h-40 w-40 rounded" />
        </div>
        <div class="rounded border border-[var(--border)] bg-[var(--input)] px-3 py-2">
          <p class="mb-1 text-xs text-[var(--muted)]">Manual entry key</p>
          <p class="break-all font-mono text-xs text-[var(--text2)]">{totp_secret}</p>
        </div>
        <div>
          <label for="s-totp" class="mb-1 block text-xs text-[var(--text2)]">6-digit code</label>
          <input
            id="s-totp"
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
          {loading ? 'Verifying...' : 'Verify & finish'}
        </button>
      </form>
    {/if}
  </div>
</div>
