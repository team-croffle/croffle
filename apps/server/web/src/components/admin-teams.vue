<script setup lang="ts">
  import { onMounted, ref } from 'vue';

  import { admin, formatTime, type AdminTeam, type TeamMember } from '../api';
  import DataTable from './data-table.vue';

  const teams = ref<AdminTeam[]>([]);
  const error = ref('');
  const busy = ref(false);
  const newName = ref('');
  /** The team whose members are open below the list. */
  const open = ref<AdminTeam | null>(null);
  const members = ref<TeamMember[]>([]);
  const memberName = ref('');
  const renameTo = ref('');

  async function refresh() {
    teams.value = (await admin.teams()).teams;
    if (open.value) {
      const current = teams.value.find((t) => t.id === open.value?.id) ?? null;
      open.value = current;
      members.value = current ? (await admin.members(current.id)).members : [];
    }
  }

  async function run(action: () => Promise<unknown>) {
    busy.value = true;
    error.value = '';
    try {
      await action();
      await refresh();
    } catch (err) {
      error.value = (err as Error).message;
    } finally {
      busy.value = false;
    }
  }

  const create = () =>
    run(async () => {
      await admin.createTeam(newName.value);
      newName.value = '';
    });

  const show = (team: AdminTeam) =>
    run(async () => {
      open.value = team;
      renameTo.value = team.name;
    });

  const rename = () =>
    run(async () => {
      if (open.value) {
        await admin.renameTeam(open.value.id, renameTo.value);
      }
    });

  const addMember = () =>
    run(async () => {
      if (open.value) {
        await admin.addMember(open.value.id, memberName.value);
        memberName.value = '';
      }
    });

  function removeMember(member: TeamMember) {
    const team = open.value;
    if (!team) {
      return;
    }
    if (
      !window.confirm(
        `Remove ${member.username} from ${team.name}? Its devices stop receiving the team's rows; what they already have stays on them.`,
      )
    ) {
      return;
    }
    return run(() => admin.removeMember(team.id, member.accountId));
  }

  onMounted(() => run(async () => undefined));
</script>

<template>
  <section class="admin">
    <form class="inline" @submit.prevent="create">
      <input v-model="newName" placeholder="team name" maxlength="64" autocomplete="off" required />
      <button class="primary" type="submit" :disabled="busy">Create team</button>
    </form>

    <p v-if="error" class="error">{{ error }}</p>

    <DataTable
      :columns="[
        { key: 'name', label: 'Team' },
        { key: 'id', label: 'Id', mono: true },
        { key: 'createdAt', label: 'Created' },
        { key: 'memberCount', label: 'Members' },
        { key: 'actions', label: '' },
      ]"
      :rows="teams"
      empty="No teams."
    >
      <template #createdAt="{ row }">{{ formatTime((row as AdminTeam).createdAt) }}</template>
      <template #actions="{ row }">
        <button class="link" :disabled="busy" @click="show(row as AdminTeam)">Members</button>
      </template>
    </DataTable>

    <template v-if="open">
      <h2>
        {{ open.name }} <code>{{ open.id }}</code>
      </h2>
      <p class="notice">
        A member that joins receives the team's earlier rows at its next sync (the desktop takes a
        fresh snapshot once).
      </p>
      <form class="inline" @submit.prevent="rename">
        <input v-model="renameTo" maxlength="64" required />
        <button class="link" type="submit" :disabled="busy">Rename</button>
      </form>
      <form class="inline" @submit.prevent="addMember">
        <input v-model="memberName" placeholder="account" autocomplete="off" required />
        <button class="primary" type="submit" :disabled="busy">Add member</button>
      </form>
      <DataTable
        :columns="[
          { key: 'username', label: 'Account' },
          { key: 'accountId', label: 'Id', mono: true },
          { key: 'role', label: 'Role' },
          { key: 'createdAt', label: 'Added' },
          { key: 'actions', label: '' },
        ]"
        :rows="members"
        empty="No members."
      >
        <template #createdAt="{ row }">{{ formatTime((row as TeamMember).createdAt) }}</template>
        <template #actions="{ row }">
          <button class="link" :disabled="busy" @click="removeMember(row as TeamMember)">
            Remove
          </button>
        </template>
      </DataTable>
    </template>
  </section>
</template>
