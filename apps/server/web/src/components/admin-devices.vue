<script setup lang="ts">
  import { onMounted, ref } from 'vue';

  import { admin, formatTime, type AdminDevice, type IssuedToken } from '../api';
  import DataTable from './data-table.vue';
  import TokenDialog from './token-dialog.vue';

  const devices = ref<AdminDevice[]>([]);
  const error = ref('');
  const busy = ref(false);
  const filter = ref('');
  const username = ref('');
  const name = ref('');
  const noExpiry = ref(true);
  const expiresIn = ref('720h');
  const issued = ref<IssuedToken | null>(null);
  const issuedFor = ref('');

  async function run(action: () => Promise<unknown>) {
    busy.value = true;
    error.value = '';
    try {
      await action();
      devices.value = (await admin.devices(filter.value.trim())).devices;
    } catch (err) {
      error.value = (err as Error).message;
    } finally {
      busy.value = false;
    }
  }

  const issue = () =>
    run(async () => {
      issued.value = await admin.issueToken(
        username.value,
        name.value,
        noExpiry.value,
        expiresIn.value,
      );
      issuedFor.value = username.value;
      name.value = '';
    });

  function revoke(device: AdminDevice) {
    const label = device.name ? `${device.name} (${device.id})` : device.id;
    if (
      !window.confirm(
        `Revoke ${label} of ${device.username}? It stops syncing until it signs in again.`,
      )
    ) {
      return;
    }
    return run(() => admin.revokeDevice(device.id));
  }

  onMounted(() => run(async () => undefined));
</script>

<template>
  <section class="admin">
    <form class="inline" @submit.prevent="issue">
      <input v-model="username" placeholder="account" autocomplete="off" required />
      <input v-model="name" placeholder="device name (optional)" autocomplete="off" />
      <label class="check"><input v-model="noExpiry" type="checkbox" /> never expires</label>
      <input v-if="!noExpiry" v-model="expiresIn" class="short" placeholder="720h" required />
      <button class="primary" type="submit" :disabled="busy">Issue token</button>
    </form>

    <form class="inline" @submit.prevent="run(async () => undefined)">
      <input v-model="filter" placeholder="filter by account" autocomplete="off" />
      <button class="link" type="submit">Filter</button>
    </form>

    <p v-if="error" class="error">{{ error }}</p>

    <DataTable
      :columns="[
        { key: 'username', label: 'Account' },
        { key: 'id', label: 'Device', mono: true },
        { key: 'name', label: 'Name' },
        { key: 'lastSeenAt', label: 'Last seen' },
        { key: 'tokenExpiresAt', label: 'Token expires' },
        { key: 'state', label: 'State' },
        { key: 'actions', label: '' },
      ]"
      :rows="devices"
      empty="No devices."
    >
      <template #lastSeenAt="{ row }">{{ formatTime((row as AdminDevice).lastSeenAt) }}</template>
      <template #tokenExpiresAt="{ row }">
        {{
          (row as AdminDevice).tokenExpiresAt
            ? formatTime((row as AdminDevice).tokenExpiresAt)
            : 'never'
        }}
      </template>
      <template #state="{ row }">
        <span class="tag">{{ (row as AdminDevice).state }}</span>
      </template>
      <template #actions="{ row }">
        <button
          v-if="(row as AdminDevice).state !== 'revoked'"
          class="link"
          :disabled="busy"
          @click="revoke(row as AdminDevice)"
        >
          Revoke
        </button>
      </template>
    </DataTable>

    <TokenDialog v-if="issued" :issued="issued" :username="issuedFor" @close="issued = null" />
  </section>
</template>
