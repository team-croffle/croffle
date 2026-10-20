<script setup lang="ts">
  import { computed, onMounted, ref } from 'vue';

  import {
    ApiError,
    api,
    formatTime,
    type Commit,
    type Device,
    type Health,
    type Me,
    type Row,
    type Team,
  } from './api';
  import DataTable from './components/data-table.vue';
  import LoginForm from './components/login-form.vue';

  type Tab = 'devices' | 'teams' | 'rows' | 'commits';

  const me = ref<Me | null>(null);
  const health = ref<Health | null>(null);
  const booted = ref(false);
  const error = ref('');
  const tab = ref<Tab>('devices');

  const devices = ref<Device[]>([]);
  const teams = ref<Team[]>([]);
  const rows = ref<Row[]>([]);
  const rowsHasMore = ref(false);
  const commits = ref<Commit[]>([]);
  const commitsHasMore = ref(false);
  const commitCursor = ref(0);

  const retention = computed(() => {
    const value = health.value?.retention;
    if (!value) {
      return '';
    }
    // What explains a 410 to whoever is reading: a client cursor below the
    // purge point can no longer be served.
    return `log ${value.oldestCommitSeq}–${health.value?.retention?.purgedThroughSeq ? value.purgedThroughSeq : value.oldestCommitSeq}+, purged through ${value.purgedThroughSeq}`;
  });

  /** Reloads the active tab, mapping an expired session back to the login form. */
  async function load() {
    error.value = '';
    try {
      switch (tab.value) {
        case 'devices':
          devices.value = (await api.devices()).devices;
          break;
        case 'teams':
          teams.value = (await api.teams()).teams;
          break;
        case 'rows': {
          const page = await api.rows();
          rows.value = page.rows;
          rowsHasMore.value = page.hasMore;
          break;
        }
        case 'commits': {
          const page = await api.commits(0);
          commits.value = page.commits;
          commitsHasMore.value = page.hasMore;
          commitCursor.value = page.serverSeq;
          break;
        }
      }
    } catch (err) {
      if (err instanceof ApiError && err.status === 401) {
        me.value = null;
        return;
      }
      error.value = (err as Error).message;
    }
  }

  async function loadMoreRows() {
    const last = rows.value.at(-1);
    if (!last) {
      return;
    }
    const page = await api.rows(last.table, last.rowId);
    rows.value = [...rows.value, ...page.rows];
    rowsHasMore.value = page.hasMore;
  }

  async function loadMoreCommits() {
    const page = await api.commits(commitCursor.value);
    commits.value = [...commits.value, ...page.commits];
    commitsHasMore.value = page.hasMore;
    commitCursor.value = page.serverSeq;
  }

  async function select(next: Tab) {
    tab.value = next;
    await load();
  }

  async function signedIn() {
    me.value = await api.me();
    await load();
  }

  async function signOut() {
    await api.logout();
    me.value = null;
  }

  onMounted(async () => {
    // health is public, so the header can show the version even before a login.
    try {
      health.value = await api.health();
    } catch {
      // A server that cannot answer /health will fail the next call anyway.
    }
    try {
      me.value = await api.me();
      await load();
    } catch {
      me.value = null;
    } finally {
      booted.value = true;
    }
  });
</script>

<template>
  <div v-if="!booted" class="empty" style="padding: 24px">Loading…</div>

  <LoginForm v-else-if="!me" @signed-in="signedIn" />

  <template v-else>
    <header>
      <h1>Croffle Server</h1>
      <span class="meta">
        {{ health?.server }} · protocol {{ health?.protocol.min }}–{{ health?.protocol.max }} · db
        {{ health?.db }}
        <template v-if="retention"> · {{ retention }}</template>
      </span>
      <span class="spacer" />
      <span class="meta">{{ me.username }}</span>
      <button class="link" @click="signOut">Sign out</button>
    </header>

    <main>
      <nav>
        <button :aria-current="tab === 'devices'" @click="select('devices')">Devices</button>
        <button :aria-current="tab === 'teams'" @click="select('teams')">Teams</button>
        <button :aria-current="tab === 'rows'" @click="select('rows')">Rows</button>
        <button :aria-current="tab === 'commits'" @click="select('commits')">Commits</button>
      </nav>

      <p class="notice">
        Read-only, and scoped to this account: your own rows plus the rows shared with a team you
        are in. Use the <code>croffle-server</code> commands to change anything, or to look at
        another account.
      </p>

      <p v-if="error" class="error">{{ error }}</p>

      <DataTable
        v-if="tab === 'devices'"
        :columns="[
          { key: 'id', label: 'Device', mono: true },
          { key: 'name', label: 'Name' },
          { key: 'createdAt', label: 'Created' },
          { key: 'lastSeenAt', label: 'Last seen' },
          { key: 'state', label: 'State' },
        ]"
        :rows="devices"
        empty="No devices yet. Issue a token with `croffle-server token issue`."
      >
        <template #createdAt="{ row }">{{ formatTime((row as Device).createdAt) }}</template>
        <template #lastSeenAt="{ row }">{{ formatTime((row as Device).lastSeenAt) }}</template>
        <template #state="{ row }">
          <span class="tag">{{ (row as Device).state }}</span>
        </template>
      </DataTable>

      <DataTable
        v-else-if="tab === 'teams'"
        :columns="[
          { key: 'id', label: 'Team', mono: true },
          { key: 'name', label: 'Name' },
          { key: 'isMember', label: 'Member' },
        ]"
        :rows="teams"
        empty="No teams."
      >
        <template #isMember="{ row }">{{ (row as Team).isMember ? 'yes' : 'no' }}</template>
      </DataTable>

      <template v-else-if="tab === 'rows'">
        <DataTable
          :columns="[
            { key: 'table', label: 'Table' },
            { key: 'rowId', label: 'Row', mono: true },
            { key: 'version', label: 'Ver' },
            { key: 'updatedAtMs', label: 'Updated' },
            { key: 'teamId', label: 'Team', mono: true },
            { key: 'deleted', label: 'State' },
            { key: 'payload', label: 'Payload' },
          ]"
          :rows="rows"
          empty="Nothing synchronised yet."
        >
          <template #updatedAtMs="{ row }">{{ formatTime((row as Row).updatedAtMs) }}</template>
          <template #teamId="{ row }">{{ (row as Row).teamId ?? '' }}</template>
          <template #deleted="{ row }">
            <span class="tag">{{ (row as Row).deleted ? 'deleted' : 'live' }}</span>
          </template>
          <template #payload="{ row }">
            <!-- Collapsed by default: this is the user's calendar content, not
                 something a dashboard should put on screen unasked. -->
            <details class="payload">
              <summary>show</summary>
              <pre>{{ (row as Row).payload }}</pre>
            </details>
          </template>
        </DataTable>
        <div v-if="rowsHasMore" class="pager">
          <button class="link" @click="loadMoreRows">Load more</button>
          <span>{{ rows.length }} shown</span>
        </div>
      </template>

      <template v-else>
        <DataTable
          :columns="[
            { key: 'serverSeq', label: 'Seq' },
            { key: 'table', label: 'Table' },
            { key: 'rowId', label: 'Row', mono: true },
            { key: 'op', label: 'Op' },
            { key: 'version', label: 'Ver' },
            { key: 'updatedAtMs', label: 'Updated' },
            { key: 'lastWriterId', label: 'Writer', mono: true },
          ]"
          :rows="commits"
          empty="The commit log is empty."
        >
          <template #updatedAtMs="{ row }">{{ formatTime((row as Commit).updatedAtMs) }}</template>
          <template #lastWriterId="{ row }">{{ (row as Commit).lastWriterId ?? '' }}</template>
        </DataTable>
        <div v-if="commitsHasMore" class="pager">
          <button class="link" @click="loadMoreCommits">Load more</button>
          <span>{{ commits.length }} shown</span>
        </div>
      </template>
    </main>
  </template>
</template>
