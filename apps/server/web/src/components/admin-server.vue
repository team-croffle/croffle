<script setup lang="ts">
  import { onMounted, ref } from 'vue';

  import { admin, formatBytes, formatTime, type AdminCommit, type ServerInfo } from '../api';
  import DataTable from './data-table.vue';

  const emit = defineEmits<{ info: [ServerInfo] }>();

  const info = ref<ServerInfo | null>(null);
  const commits = ref<AdminCommit[]>([]);
  const hasMore = ref(false);
  const cursor = ref(0);
  const error = ref('');
  const busy = ref(false);
  const vacuumed = ref('');

  async function run(action: () => Promise<unknown>) {
    busy.value = true;
    error.value = '';
    try {
      await action();
    } catch (err) {
      error.value = (err as Error).message;
    } finally {
      busy.value = false;
    }
  }

  async function loadInfo() {
    info.value = await admin.info();
    emit('info', info.value);
  }

  async function loadCommits(more = false) {
    const page = await admin.commits(more ? cursor.value : 0);
    commits.value = more ? [...commits.value, ...page.commits] : page.commits;
    hasMore.value = page.hasMore;
    cursor.value = page.serverSeq;
  }

  function vacuum() {
    if (
      !window.confirm(
        'Run VACUUM? It rewrites the whole database file; syncing and every other write wait until it finishes.',
      )
    ) {
      return;
    }
    return run(async () => {
      const result = await admin.vacuum();
      vacuumed.value = `${formatBytes(result.sizeBefore)} → ${formatBytes(result.sizeAfter)}`;
      await loadInfo();
    });
  }

  onMounted(() =>
    run(async () => {
      await loadInfo();
      await loadCommits();
    }),
  );
</script>

<template>
  <section class="admin">
    <p v-if="error" class="error">{{ error }}</p>

    <dl v-if="info" class="facts">
      <dt>Database</dt>
      <dd>
        <code>{{ info.path }}</code> · {{ formatBytes(info.sizeBytes) }} · schema
        {{ info.schemaVersion }}
      </dd>
      <dt>Counts</dt>
      <dd>
        {{ info.accounts }} accounts · {{ info.devices }} devices · {{ info.teams }} teams ·
        {{ info.rows }} rows · {{ info.commits }} commits
      </dd>
      <dt>Commit log</dt>
      <dd>
        seq {{ info.oldestSeq }}–{{ info.newestSeq }}, purged through {{ info.purgedThroughSeq }}
      </dd>
      <dt>Secret key</dt>
      <dd>
        {{ info.secretKey }} · one-time code secrets: {{ info.otpSecretsEncrypted }} encrypted,
        {{ info.otpSecretsPlain }} plain
      </dd>
    </dl>

    <p>
      <button class="primary" :disabled="busy" @click="vacuum">Vacuum</button>
      <span v-if="vacuumed" class="meta"> {{ vacuumed }}</span>
    </p>

    <h2>Commit log (every account)</h2>
    <p class="notice">
      Metadata only. The row contents are not shown here; <code>croffle-server log dump</code> has
      them.
    </p>
    <DataTable
      :columns="[
        { key: 'serverSeq', label: 'Seq' },
        { key: 'table', label: 'Table' },
        { key: 'rowId', label: 'Row', mono: true },
        { key: 'op', label: 'Op' },
        { key: 'ownerId', label: 'Owner', mono: true },
        { key: 'teamId', label: 'Team', mono: true },
        { key: 'version', label: 'Ver' },
        { key: 'updatedAtMs', label: 'Updated' },
      ]"
      :rows="commits"
      empty="The commit log is empty."
    >
      <template #teamId="{ row }">{{ (row as AdminCommit).teamId ?? '' }}</template>
      <template #updatedAtMs="{ row }">{{ formatTime((row as AdminCommit).updatedAtMs) }}</template>
    </DataTable>
    <div v-if="hasMore" class="pager">
      <button class="link" :disabled="busy" @click="run(() => loadCommits(true))">Load more</button>
      <span>{{ commits.length }} shown</span>
    </div>
  </section>
</template>
