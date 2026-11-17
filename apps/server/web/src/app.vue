<script setup lang="ts">
  import { computed, onMounted, ref } from 'vue';

  import {
    ApiError,
    admin,
    api,
    formatTime,
    type Commit,
    type Device,
    type Health,
    type Me,
    type Row,
    type ServerInfo,
    type Team,
  } from './api';
  import AdminAccounts from './components/admin-accounts.vue';
  import AdminDevices from './components/admin-devices.vue';
  import AdminServer from './components/admin-server.vue';
  import AdminTeams from './components/admin-teams.vue';
  import DataTable from './components/data-table.vue';
  import LoginForm from './components/login-form.vue';
  import SecurityPanel from './components/security-panel.vue';

  type Tab =
    | 'devices'
    | 'teams'
    | 'rows'
    | 'commits'
    | 'security'
    | 'admin-accounts'
    | 'admin-devices'
    | 'admin-teams'
    | 'admin-server';

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
  /** The operator's view of the server; null for a regular account. */
  const serverInfo = ref<ServerInfo | null>(null);
  const keyMissing = computed(() => serverInfo.value?.secretKey === 'missing');

  /** Reads the server info an operator's warning banner needs. */
  async function loadServerInfo() {
    serverInfo.value = null;
    if (!me.value?.operator) {
      return;
    }
    try {
      serverInfo.value = await admin.info();
    } catch {
      // The tabs show their own errors; the banner is a convenience.
    }
  }

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
        case 'security':
          me.value = await api.me();
          break;
        default:
          // The operator tabs load themselves when they mount.
          break;
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

  /** Revokes one of this account's own devices after an explicit confirmation. */
  async function revoke(device: Device) {
    const label = device.name ? `${device.name} (${device.id})` : device.id;
    if (!window.confirm(`Revoke ${label}? It stops syncing until it signs in again.`)) {
      return;
    }
    error.value = '';
    try {
      await api.revokeDevice(device.id);
    } catch (err) {
      error.value = (err as Error).message;
      return;
    }
    await load();
  }

  async function select(next: Tab) {
    tab.value = next;
    await load();
  }

  async function signedIn() {
    me.value = await api.me();
    tab.value = 'devices';
    await loadServerInfo();
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
      await loadServerInfo();
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
        <button :aria-current="tab === 'security'" @click="select('security')">Security</button>
        <template v-if="me.operator">
          <span class="divider" aria-hidden="true" />
          <button :aria-current="tab === 'admin-accounts'" @click="select('admin-accounts')">
            Accounts
          </button>
          <button :aria-current="tab === 'admin-devices'" @click="select('admin-devices')">
            All devices
          </button>
          <button :aria-current="tab === 'admin-teams'" @click="select('admin-teams')">
            Manage teams
          </button>
          <button :aria-current="tab === 'admin-server'" @click="select('admin-server')">
            Server
          </button>
        </template>
      </nav>

      <p v-if="me.operator && keyMissing" class="warning" role="alert">
        No secret key is set: one-time code secrets are stored in plain text in the database. Set
        <code>CROFFLE_SECRET_KEY</code> or <code>CROFFLE_SECRET_KEY_FILE</code> and restart — see
        <a
          href="https://github.com/team-croffle/croffle/blob/master/apps/server/README.md#secret-key"
          target="_blank"
          rel="noreferrer"
          >Secret key</a
        >
        in the server README.
      </p>

      <p v-if="tab.startsWith('admin-')" class="notice">
        Operator view: every account on this server. Each change here is written to the server log.
      </p>
      <p v-else class="notice">
        Scoped to this account: your own rows plus the rows shared with a team you are in. You can
        revoke your own devices here<template v-if="me.operator"
          >; the operator tabs manage every account</template
        ><template v-else>; use the <code>croffle-server</code> commands for anything else</template
        >.
      </p>

      <p v-if="error" class="error">{{ error }}</p>

      <DataTable
        v-if="tab === 'devices'"
        :columns="[
          { key: 'id', label: 'Device', mono: true },
          { key: 'name', label: 'Name' },
          { key: 'createdAt', label: 'Created' },
          { key: 'lastSeenAt', label: 'Last seen' },
          { key: 'tokenExpiresAt', label: 'Token expires' },
          { key: 'state', label: 'State' },
          { key: 'actions', label: '' },
        ]"
        :rows="devices"
        empty="No devices yet. Issue a token with `croffle-server token issue`."
      >
        <template #createdAt="{ row }">{{ formatTime((row as Device).createdAt) }}</template>
        <template #lastSeenAt="{ row }">{{ formatTime((row as Device).lastSeenAt) }}</template>
        <template #tokenExpiresAt="{ row }">
          {{
            (row as Device).tokenExpiresAt ? formatTime((row as Device).tokenExpiresAt) : 'never'
          }}
        </template>
        <template #state="{ row }">
          <span class="tag">{{ (row as Device).state }}</span>
        </template>
        <template #actions="{ row }">
          <button
            v-if="(row as Device).state !== 'revoked'"
            class="link"
            @click="revoke(row as Device)"
          >
            Revoke
          </button>
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

      <SecurityPanel v-else-if="tab === 'security'" :me="me" @changed="load" />

      <AdminAccounts v-else-if="tab === 'admin-accounts'" :me-id="me.accountId" />
      <AdminDevices v-else-if="tab === 'admin-devices'" />
      <AdminTeams v-else-if="tab === 'admin-teams'" />
      <AdminServer v-else-if="tab === 'admin-server'" @info="serverInfo = $event" />

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
