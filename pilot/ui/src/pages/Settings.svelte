<script>
  import { onMount } from 'svelte';
  import { api } from '../lib/api.js';
  import { navigate } from '../lib/router.js';
  import Alert from '../components/Alert.svelte';
  import SetupWizard from '../components/SetupWizard.svelte';
  import SettingsAdvanced from '../components/SettingsAdvanced.svelte';

  let loading = true;
  let saving = false;
  let error = '';
  let success = '';

  // Default to 'easy' on first visit, persist choice
  let viewMode = localStorage.getItem('pp_settings_view') || 'easy';
  $: localStorage.setItem('pp_settings_view', viewMode);

  // Settings object — uses BACKEND key names
  let settings = {
    mail_mode: 'catch',
    postfix_hostname: '',
    postfix_myorigin: '',
    postfix_message_size: 10485760,
    postfix_mynetworks: '127.0.0.0/8',
    postfix_relay_host: '',
    postfix_relay_user: '',
    postfix_relay_pass: '',
    dkim_enabled: false,
    dkim_domain: '',
    dkim_selector: 'mail',
    dkim_key_size: '2048',
    mp_max_messages: 1000,
  };

  onMount(async () => {
    try {
      const data = await api.getSettings();
      if (data.settings) {
        // Merge server values, converting dkim_enabled to boolean
        const s = data.settings;
        if (s.dkim_enabled !== undefined) s.dkim_enabled = s.dkim_enabled === 'true';
        settings = { ...settings, ...s };
      }
    } catch (e) {
      error = e.message || 'Failed to load settings';
    } finally {
      loading = false;
    }
  });

  async function save() {
    error = '';
    success = '';
    saving = true;
    try {
      // Convert boolean to string for backend
      const payload = { ...settings, dkim_enabled: settings.dkim_enabled ? 'true' : 'false' };
      await api.saveSettings(payload);
      success = 'Settings saved';
    } catch (e) {
      error = e.message || 'Failed to save';
    } finally {
      saving = false;
    }
  }

  async function handleWizardComplete(event) {
    success = 'Configuration saved';
    // Reload settings so advanced view is in sync
    try {
      const data = await api.getSettings();
      if (data.settings) {
        const s = data.settings;
        if (s.dkim_enabled !== undefined) s.dkim_enabled = s.dkim_enabled === 'true';
        settings = { ...settings, ...s };
      }
    } catch (_) {}
    viewMode = 'advanced';
    if (event.detail?.redirectToDns) {
      navigate('/dns');
    }
  }
</script>

<div class="mx-auto max-w-3xl px-4 py-6">
  <!-- Header with toggle -->
  <div class="mb-6 flex items-center justify-between">
    <h1 class="text-lg font-semibold text-[var(--text)]">Settings</h1>
    <div class="flex rounded-md border border-[var(--border)] p-0.5">
      <button
        type="button"
        on:click={() => viewMode = 'easy'}
        class="rounded px-3 py-1 text-xs font-medium transition-colors {viewMode === 'easy' ? 'bg-[var(--input)] text-[var(--text)]' : 'text-[var(--muted)] hover:text-[var(--text2)]'}"
      >
        Easy Setup
      </button>
      <button
        type="button"
        on:click={() => viewMode = 'advanced'}
        class="rounded px-3 py-1 text-xs font-medium transition-colors {viewMode === 'advanced' ? 'bg-[var(--input)] text-[var(--text)]' : 'text-[var(--muted)] hover:text-[var(--text2)]'}"
      >
        Advanced
      </button>
    </div>
  </div>

  {#if loading}
    <p class="text-sm text-[var(--muted)]">Loading...</p>
  {:else}
    {#if error}
      <div class="mb-4"><Alert type="error" message={error} /></div>
    {/if}
    {#if success}
      <div class="mb-4"><Alert type="success" message={success} /></div>
    {/if}

    {#if viewMode === 'easy'}
      <SetupWizard on:complete={handleWizardComplete} />
    {:else}
      <SettingsAdvanced bind:settings {saving} on:save={save} />
    {/if}
  {/if}
</div>
