import { mkdirSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';

import vue from '@vitejs/plugin-vue';
import { defineConfig, type Plugin } from 'vite';

const placeholder = `This directory is the dashboard build output, embedded into the binary.

Only this placeholder is tracked. It exists because //go:embed fails to compile
on a missing or empty directory, and the server has to build without the
dashboard having been built first.
`;

/**
 * Writes the tracked placeholder back after a build.
 *
 * `emptyOutDir` removes it, which would show up as a deleted file in git every
 * time anyone builds the dashboard.
 */
function keepEmbedPlaceholder(outDir: string): Plugin {
  return {
    name: 'croffle-keep-embed-placeholder',
    closeBundle() {
      mkdirSync(outDir, { recursive: true });
      writeFileSync(join(outDir, '.gitkeep'), placeholder);
    },
  };
}

export default defineConfig({
  plugins: [vue(), keepEmbedPlaceholder('embedded')],
  // Relative asset paths, because the server can be mounted under a base path
  // (CROFFLE_BASE_PATH). An absolute /assets/… would 404 behind a proxy that
  // forwards a subpath.
  base: './',
  build: {
    // Not "dist": the repository root gitignores every dist/ directory, and
    // this output has to contain one tracked placeholder so that the Go
    // //go:embed below it always compiles, dashboard built or not.
    outDir: 'embedded',
    emptyOutDir: true,
    // One small bundle: the Go binary embeds whatever is here, and a handful of
    // chunks for four tables buys nothing.
    chunkSizeWarningLimit: 1024,
  },
  server: {
    port: 5180,
    // `pnpm dev` talks to a server started with `make run`.
    proxy: { '/dash': 'http://127.0.0.1:8787' },
  },
});
