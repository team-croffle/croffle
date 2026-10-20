<script setup lang="ts" generic="T">
  // One table component for every view: the dashboard of 0.1 is four lists, and
  // four bespoke tables would be four places to fix a column later.
  defineProps<{
    columns: { key: string; label: string; mono?: boolean }[];
    rows: T[];
    empty: string;
  }>();

  /**
   * Reads a column off a row.
   *
   * The cast lives here rather than in the template: a generic type argument
   * inside an interpolation reads as an HTML tag to the formatter.
   */
  function cell(row: T, key: string): unknown {
    return (row as Record<string, unknown>)[key];
  }
</script>

<template>
  <table v-if="rows.length">
    <thead>
      <tr>
        <th v-for="column in columns" :key="column.key">{{ column.label }}</th>
      </tr>
    </thead>
    <tbody>
      <tr v-for="(row, index) in rows" :key="index">
        <td v-for="column in columns" :key="column.key" :class="{ mono: column.mono }">
          <slot :name="column.key" :row="row">{{ cell(row, column.key) }}</slot>
        </td>
      </tr>
    </tbody>
  </table>
  <p v-else class="empty">{{ empty }}</p>
</template>
