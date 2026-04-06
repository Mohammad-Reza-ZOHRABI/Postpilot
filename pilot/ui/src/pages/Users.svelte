<script>
  import { onMount } from 'svelte';
  import { api } from '../lib/api.js';
  import Alert from '../components/Alert.svelte';

  let loading = true;
  let error = '';
  let success = '';
  let users = [];

  let email = '';
  let password = '';
  let role = 'member';
  let creating = false;

  // TOTP info after creating a user
  let newUserTotp = null;

  onMount(loadUsers);

  async function loadUsers() {
    loading = true;
    error = '';
    try {
      const data = await api.listUsers();
      users = data.users || [];
    } catch (e) {
      error = e.message;
    } finally {
      loading = false;
    }
  }

  async function create() {
    if (!email.trim() || !password) return;
    error = '';
    success = '';
    creating = true;
    try {
      const data = await api.createUser(email.trim(), password, role);
      newUserTotp = { email: email.trim(), ...data };
      email = '';
      password = '';
      await loadUsers();
    } catch (e) {
      error = e.message;
    } finally {
      creating = false;
    }
  }

  async function toggleRole(user) {
    const newRole = user.role === 'admin' ? 'member' : 'admin';
    try {
      await api.updateUserRole(user.id, newRole);
      await loadUsers();
    } catch (e) {
      error = e.message;
    }
  }

  async function remove(user) {
    if (!confirm(`Delete ${user.email}?`)) return;
    try {
      await api.deleteUser(user.id);
      success = 'User deleted';
      await loadUsers();
    } catch (e) {
      error = e.message;
    }
  }
</script>

<div class="mx-auto max-w-4xl px-4 py-6">
  {#if error}<div class="mb-4"><Alert type="error" message={error} /></div>{/if}
  {#if success}<div class="mb-4"><Alert type="success" message={success} /></div>{/if}

  <!-- New user TOTP display -->
  {#if newUserTotp}
    <div class="mb-4 rounded-lg border border-emerald-500/30 bg-emerald-500/5 p-4">
      <p class="mb-2 text-sm font-medium text-emerald-400">User created: {newUserTotp.email}</p>
      <p class="mb-2 text-xs" style="color:var(--muted)">Share this QR code with the user to set up their authenticator app:</p>
      {#if newUserTotp.qr_data_url}
        <div class="mb-2 text-center">
          <img src={newUserTotp.qr_data_url} alt="QR Code" class="inline-block rounded bg-white p-2" width="160" />
        </div>
      {/if}
      <p class="text-center font-mono text-xs" style="color:var(--muted)">Secret: {newUserTotp.totp_secret}</p>
      <button on:click={() => newUserTotp = null} class="mt-3 w-full rounded py-1.5 text-xs" style="border:1px solid var(--border);color:var(--muted)">
        Dismiss
      </button>
    </div>
  {/if}

  <!-- Create form -->
  <form on:submit|preventDefault={create} class="mb-6 flex items-end gap-2">
    <div class="flex-1">
      <label for="u-email" class="mb-1 block text-xs" style="color:var(--text2)">Email</label>
      <input id="u-email" type="email" bind:value={email} required placeholder="user@example.com"
        class="w-full rounded border px-3 py-1.5 text-sm outline-none"
        style="border-color:var(--border);background:var(--input);color:var(--text)" />
    </div>
    <div class="w-40">
      <label for="u-pass" class="mb-1 block text-xs" style="color:var(--text2)">Password</label>
      <input id="u-pass" type="password" bind:value={password} required minlength="12" placeholder="12+ chars"
        class="w-full rounded border px-3 py-1.5 text-sm outline-none"
        style="border-color:var(--border);background:var(--input);color:var(--text)" />
    </div>
    <div class="w-28">
      <label for="u-role" class="mb-1 block text-xs" style="color:var(--text2)">Role</label>
      <select id="u-role" bind:value={role}
        class="w-full rounded border px-3 py-1.5 text-sm outline-none"
        style="border-color:var(--border);background:var(--input);color:var(--text)">
        <option value="member">Member</option>
        <option value="admin">Admin</option>
      </select>
    </div>
    <button type="submit" disabled={creating}
      class="rounded px-4 py-1.5 text-sm font-medium text-white transition-colors disabled:opacity-50"
      style="background:var(--accent)">
      {creating ? '...' : 'Add'}
    </button>
  </form>

  <!-- Users table -->
  {#if loading}
    <p class="text-sm" style="color:var(--muted)">Loading...</p>
  {:else if users.length === 0}
    <p class="py-8 text-center text-sm" style="color:var(--muted)">No users</p>
  {:else}
    <div class="overflow-x-auto rounded-lg border" style="border-color:var(--border)">
      <table class="w-full text-sm">
        <thead>
          <tr class="border-b text-left text-xs uppercase tracking-wide" style="border-color:var(--border);color:var(--muted)">
            <th class="px-3 py-2">Email</th>
            <th class="px-3 py-2">Role</th>
            <th class="px-3 py-2">Created</th>
            <th class="px-3 py-2"></th>
          </tr>
        </thead>
        <tbody>
          {#each users as user}
            <tr class="border-b" style="border-color:var(--border);color:var(--text2)">
              <td class="px-3 py-2 text-xs" style="color:var(--text)">{user.email}</td>
              <td class="px-3 py-2">
                <button on:click={() => toggleRole(user)}
                  class="rounded-full border px-2 py-0.5 text-xs transition-colors {user.role === 'admin' ? 'border-violet-500/50 text-violet-400' : 'text-[var(--muted)]'}"
                  style="border-color:var(--border)">
                  {user.role}
                </button>
              </td>
              <td class="px-3 py-2 text-xs" style="color:var(--muted)">{user.created_at}</td>
              <td class="px-3 py-2">
                <button on:click={() => remove(user)} class="text-xs text-red-400 transition-colors hover:text-red-300">
                  Delete
                </button>
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  {/if}
</div>
