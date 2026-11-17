<script setup lang="ts">
  import { onMounted, ref } from 'vue';

  import { admin, formatTime, type AdminAccount } from '../api';
  import DataTable from './data-table.vue';

  const props = defineProps<{ meId: string }>();

  const accounts = ref<AdminAccount[]>([]);
  const error = ref('');
  const busy = ref(false);
  const username = ref('');
  const password = ref('');
  const operator = ref(false);
  /** The account whose password is being reset, if any. */
  const resetting = ref<AdminAccount | null>(null);
  const newPassword = ref('');

  async function run(action: () => Promise<unknown>) {
    busy.value = true;
    error.value = '';
    try {
      await action();
      accounts.value = (await admin.accounts()).accounts;
    } catch (err) {
      error.value = (err as Error).message;
    } finally {
      busy.value = false;
    }
  }

  const create = () =>
    run(async () => {
      await admin.createAccount(username.value, password.value, operator.value);
      username.value = '';
      password.value = '';
      operator.value = false;
    });

  function toggleDisabled(account: AdminAccount) {
    const disable = !account.disabledAt;
    if (
      disable &&
      !window.confirm(
        `Disable ${account.username}? Its devices stop syncing and it cannot sign in until enabled again.`,
      )
    ) {
      return;
    }
    return run(() => admin.setDisabled(account.id, disable));
  }

  function toggleOperator(account: AdminAccount) {
    const grant = !account.operator;
    const message = grant
      ? `Make ${account.username} an operator? It can then change every account, token and team.`
      : `Remove ${account.username} as an operator?`;
    if (!window.confirm(message)) {
      return;
    }
    return run(() => admin.setOperator(account.id, grant));
  }

  function disableOtp(account: AdminAccount) {
    if (
      !window.confirm(
        `Turn off one-time codes for ${account.username}? The password alone signs in.`,
      )
    ) {
      return;
    }
    return run(() => admin.disableOtp(account.id));
  }

  const resetPassword = () =>
    run(async () => {
      if (!resetting.value) {
        return;
      }
      await admin.setPassword(resetting.value.id, newPassword.value);
      resetting.value = null;
      newPassword.value = '';
    });

  onMounted(() => run(async () => undefined));
</script>

<template>
  <section class="admin">
    <form class="inline" @submit.prevent="create">
      <input v-model="username" placeholder="username" autocomplete="off" required />
      <input
        v-model="password"
        type="password"
        placeholder="password"
        autocomplete="new-password"
        required
      />
      <label class="check"><input v-model="operator" type="checkbox" /> operator</label>
      <button class="primary" type="submit" :disabled="busy">Create account</button>
    </form>

    <form v-if="resetting" class="inline" @submit.prevent="resetPassword">
      <span>New password for {{ resetting.username }}</span>
      <input v-model="newPassword" type="password" autocomplete="new-password" required />
      <button class="primary" type="submit" :disabled="busy">Set password</button>
      <button class="link" type="button" @click="resetting = null">Cancel</button>
      <span class="meta">Signs its browser sessions out; device tokens keep working.</span>
    </form>

    <p v-if="error" class="error">{{ error }}</p>

    <DataTable
      :columns="[
        { key: 'username', label: 'Account' },
        { key: 'id', label: 'Id', mono: true },
        { key: 'createdAt', label: 'Created' },
        { key: 'state', label: 'State' },
        { key: 'operator', label: 'Operator' },
        { key: 'otp', label: 'One-time codes' },
        { key: 'deviceCount', label: 'Devices' },
        { key: 'actions', label: '' },
      ]"
      :rows="accounts"
      empty="No accounts."
    >
      <template #createdAt="{ row }">{{ formatTime((row as AdminAccount).createdAt) }}</template>
      <template #state="{ row }">
        <span class="tag">{{ (row as AdminAccount).disabledAt ? 'disabled' : 'active' }}</span>
      </template>
      <template #operator="{ row }">{{ (row as AdminAccount).operator ? 'yes' : 'no' }}</template>
      <template #actions="{ row }">
        <span class="actions">
          <button class="link" :disabled="busy" @click="toggleDisabled(row as AdminAccount)">
            {{ (row as AdminAccount).disabledAt ? 'Enable' : 'Disable' }}
          </button>
          <button class="link" :disabled="busy" @click="resetting = row as AdminAccount">
            Reset password
          </button>
          <button
            v-if="(row as AdminAccount).otp !== 'off'"
            class="link"
            :disabled="busy"
            @click="disableOtp(row as AdminAccount)"
          >
            Turn off codes
          </button>
          <button class="link" :disabled="busy" @click="toggleOperator(row as AdminAccount)">
            {{ (row as AdminAccount).operator ? 'Remove operator' : 'Make operator' }}
          </button>
          <span v-if="(row as AdminAccount).id === props.meId" class="meta">(you)</span>
        </span>
      </template>
    </DataTable>
  </section>
</template>
