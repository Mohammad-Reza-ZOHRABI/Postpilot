<script>
  import { createEventDispatcher } from 'svelte';
  import { api } from '../lib/api.js';
  import Alert from './Alert.svelte';
  import HelpText from './HelpText.svelte';

  const dispatch = createEventDispatcher();

  let step = 1;
  let choice = '';
  let domain = '';
  let relayHost = '';
  let relayUser = '';
  let relayPass = '';
  let saving = false;
  let error = '';

  const choices = [
    {
      id: 'catch',
      title: 'Test locally',
      desc: 'Catch all outgoing mail without delivering it. Great for development.',
    },
    {
      id: 'relay',
      title: 'Use an SMTP provider',
      desc: 'Relay through Gmail, SendGrid, Amazon SES, or any SMTP service.',
    },
    {
      id: 'direct',
      title: 'Send from this server',
      desc: 'Deliver directly via DNS MX lookup. Requires SPF, DKIM, and PTR records.',
    },
  ];

  function next() {
    if (!choice) return;
    if (choice === 'catch') return saveCatch();
    step = 2;
  }

  async function saveCatch() {
    saving = true;
    error = '';
    try {
      await api.saveSettings({ mail_mode: 'catch' });
      dispatch('complete', {});
    } catch (e) {
      error = e.message;
    } finally {
      saving = false;
    }
  }

  async function saveRelay() {
    if (!relayHost) { error = 'Relay host is required'; return; }
    saving = true;
    error = '';
    try {
      await api.saveSettings({
        mail_mode: 'relay',
        postfix_relay_host: relayHost,
        postfix_relay_user: relayUser,
        postfix_relay_pass: relayPass,
      });
      dispatch('complete', {});
    } catch (e) {
      error = e.message;
    } finally {
      saving = false;
    }
  }

  async function saveDirect() {
    if (!domain) { error = 'Domain is required'; return; }
    saving = true;
    error = '';
    try {
      await api.saveSettings({
        mail_mode: 'direct',
        postfix_hostname: 'mail.' + domain,
        postfix_myorigin: domain,
        dkim_enabled: 'true',
        dkim_domain: domain,
        dkim_selector: 'mail',
        dkim_key_size: '2048',
      });
      dispatch('complete', { redirectToDns: true });
    } catch (e) {
      error = e.message;
    } finally {
      saving = false;
    }
  }
</script>

<div class="mx-auto max-w-xl">
  {#if error}
    <div class="mb-4"><Alert type="error" message={error} /></div>
  {/if}

  {#if step === 1}
    <p class="mb-4 text-sm text-[var(--text2)]">How do you want to handle outgoing email?</p>
    <div class="space-y-2">
      {#each choices as c}
        <button
          type="button"
          on:click={() => choice = c.id}
          class="w-full rounded-lg border p-4 text-left transition-colors {choice === c.id ? 'border-[var(--accent)] bg-[var(--accent)]/5' : 'border-[var(--border)] hover:border-[var(--border)]'}"
        >
          <div class="text-sm font-medium text-[var(--text)]">{c.title}</div>
          <div class="mt-0.5 text-xs text-[var(--muted)]">{c.desc}</div>
        </button>
      {/each}
    </div>
    <button
      type="button"
      disabled={!choice || saving}
      on:click={next}
      class="mt-4 w-full rounded bg-[var(--accent)] px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-[var(--accent-h)] disabled:opacity-40"
    >
      {saving ? 'Saving...' : choice === 'catch' ? 'Save & finish' : 'Continue'}
    </button>

  {:else if step === 2 && choice === 'relay'}
    <button type="button" on:click={() => step = 1} class="mb-4 text-xs text-[var(--muted)] hover:text-[var(--text2)]">&larr; Back</button>
    <p class="mb-4 text-sm text-[var(--text2)]">Enter your SMTP provider details.</p>
    <div class="space-y-3">
      <div>
        <label for="w-host" class="mb-1 block text-xs text-[var(--text2)]">SMTP host</label>
        <input id="w-host" type="text" bind:value={relayHost} placeholder="smtp.sendgrid.net:587" class="w-full rounded border border-[var(--border)] bg-[var(--input)] px-3 py-1.5 text-sm text-[var(--text)] outline-none focus:border-[var(--accent)]" />
        <HelpText text="Include the port, e.g. smtp.gmail.com:587" />
      </div>
      <div>
        <label for="w-user" class="mb-1 block text-xs text-[var(--text2)]">Username</label>
        <input id="w-user" type="text" bind:value={relayUser} placeholder="apikey" class="w-full rounded border border-[var(--border)] bg-[var(--input)] px-3 py-1.5 text-sm text-[var(--text)] outline-none focus:border-[var(--accent)]" />
        <HelpText text="API key or email used to authenticate" />
      </div>
      <div>
        <label for="w-pass" class="mb-1 block text-xs text-[var(--text2)]">Password</label>
        <input id="w-pass" type="password" bind:value={relayPass} class="w-full rounded border border-[var(--border)] bg-[var(--input)] px-3 py-1.5 text-sm text-[var(--text)] outline-none focus:border-[var(--accent)]" />
        <HelpText text="API secret or app password" />
      </div>
    </div>
    <button
      type="button"
      disabled={saving}
      on:click={saveRelay}
      class="mt-4 w-full rounded bg-[var(--accent)] px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-[var(--accent-h)] disabled:opacity-40"
    >
      {saving ? 'Saving...' : 'Save & finish'}
    </button>

  {:else if step === 2 && choice === 'direct'}
    <button type="button" on:click={() => step = 1} class="mb-4 text-xs text-[var(--muted)] hover:text-[var(--text2)]">&larr; Back</button>
    <p class="mb-4 text-sm text-[var(--text2)]">Enter your sending domain. We'll configure everything else automatically.</p>
    <div>
      <label for="w-domain" class="mb-1 block text-xs text-[var(--text2)]">Domain</label>
      <input id="w-domain" type="text" bind:value={domain} placeholder="example.com" class="w-full rounded border border-[var(--border)] bg-[var(--input)] px-3 py-1.5 text-sm text-[var(--text)] outline-none focus:border-[var(--accent)]" />
      <HelpText text="Your sending domain. Hostname will be set to mail.{domain}, DKIM enabled automatically." />
    </div>
    {#if domain}
      <div class="mt-3 rounded border border-[var(--border)] bg-[var(--surface)]/50 p-3 text-xs text-[var(--muted)]">
        <div>Hostname: <span class="text-[var(--text2)]">mail.{domain}</span></div>
        <div>Origin: <span class="text-[var(--text2)]">{domain}</span></div>
        <div>DKIM: <span class="text-[var(--text2)]">Enabled (2048-bit)</span></div>
      </div>
    {/if}
    <button
      type="button"
      disabled={!domain || saving}
      on:click={saveDirect}
      class="mt-4 w-full rounded bg-[var(--accent)] px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-[var(--accent-h)] disabled:opacity-40"
    >
      {saving ? 'Saving...' : 'Save & configure DNS'}
    </button>
  {/if}
</div>
